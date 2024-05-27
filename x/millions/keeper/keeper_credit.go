package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogotypes "github.com/cosmos/gogoproto/types"

	"github.com/furysport/furya-chain/v2/x/millions/types"
)

// TransferCreditToRemoteZone Transfer a credit to a remote zone
// - wait for the ICA callback to move to OnTransferCreditToRemoteZoneCompleted
// - or go to OnTransferCreditToRemoteZoneCompleted directly if local zone (no-op)
func (k Keeper) TransferCreditToRemoteZone(ctx sdk.Context, poolID uint64, creditID uint64) error {
	// Acquire pool config
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return err
	}

	// Acquire the credit
	credit, err := k.GetPoolCredit(ctx, poolID, creditID)
	if err != nil {
		return err
	}
	if credit.State != types.CreditState_IbcTransfer {
		return errorsmod.Wrapf(types.ErrIllegalStateOperation, "state should be %s but is %s", types.CreditState_IbcTransfer.String(), credit.State.String())
	}

	poolRunner := k.MustGetPoolRunner(pool.PoolType)
	if err := poolRunner.TransferCreditToRemoteZone(ctx, pool, credit); err != nil {
		// Return with error (if any) here since it is the first operation and nothing needs to be saved to state
		return err
	}
	if pool.IsLocalZone(ctx) {
		// Move instantly to next stage since local ops don't wait for callbacks
		return k.OnTransferCreditToRemoteZoneCompleted(ctx, poolID, creditID, false)
	}
	return nil
}

// OnTransferCreditToRemoteZoneCompleted Acknowledge the IBC transfer to a remote zone response
// then moves to DelegateCreditOnRemoteZone in case of success
func (k Keeper) OnTransferCreditToRemoteZoneCompleted(ctx sdk.Context, poolID uint64, creditID uint64, isError bool) error {
	// Acquire the credit
	credit, err := k.GetPoolCredit(ctx, poolID, creditID)
	if err != nil {
		return err
	}

	// Verify credit state
	if credit.State != types.CreditState_IbcTransfer {
		return errorsmod.Wrapf(types.ErrIllegalStateOperation, "state should be %s but is %s", types.CreditState_IbcTransfer.String(), credit.State.String())
	}

	if isError {
		k.UpdateCreditStatus(ctx, poolID, creditID, types.CreditState_IbcTransfer, true)
		return nil
	}

	k.UpdateCreditStatus(ctx, poolID, creditID, types.CreditState_IcaDelegate, false)
	return k.DelegateCreditOnRemoteZone(ctx, poolID, creditID)
}

// DelegateCreditOnRemoteZone Delegates a credit on the remote zone (put it to work)
// - wait for the ICA callback to move to OnDelegateCreditOnRemoteZoneCompleted
// - or go to OnDelegateCreditOnRemoteZoneCompleted directly if local zone
func (k Keeper) DelegateCreditOnRemoteZone(ctx sdk.Context, poolID uint64, creditID uint64) error {
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return err
	}

	credit, err := k.GetPoolCredit(ctx, poolID, creditID)
	if err != nil {
		return err
	}
	if credit.State != types.CreditState_IcaDelegate {
		return errorsmod.Wrapf(types.ErrIllegalStateOperation, "state should be %s but is %s", types.CreditState_IcaDelegate.String(), credit.State.String())
	}

	// Return with error (if any) here since it is the first operation and nothing needs to be saved to state
	poolRunner := k.MustGetPoolRunner(pool.PoolType)
	splits, err := poolRunner.DelegateCreditOnRemoteZone(ctx, pool, credit)
	if pool.IsLocalZone(ctx) {
		// Always save state in case of local zone
		// TODO: we should probably return and error here since it is atomic with the credit transaction
		return k.OnDelegateCreditOnRemoteZoneCompleted(ctx, poolID, creditID, splits, err != nil)
	} else if err != nil {
		// Only save state in case of error for remote zones (otherwise wait for the callback)
		return k.OnDelegateCreditOnRemoteZoneCompleted(ctx, poolID, creditID, splits, true)
	}
	return nil
}

// OnDelegateCreditOnRemoteZoneCompleted Acknowledge the ICA operation to delegate a credit on the remote zone (credit has been put to work)
func (k Keeper) OnDelegateCreditOnRemoteZoneCompleted(ctx sdk.Context, poolID uint64, creditID uint64, splits []*types.SplitDelegation, isError bool) error {
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return err
	}
	credit, err := k.GetPoolCredit(ctx, poolID, creditID)
	if err != nil {
		return err
	}

	// Verify credit state
	if credit.State != types.CreditState_IcaDelegate {
		return errorsmod.Wrapf(types.ErrIllegalStateOperation, "state should be %s but is %s", types.CreditState_IcaDelegate.String(), credit.State.String())
	}

	if isError {
		k.UpdateCreditStatus(ctx, poolID, creditID, types.CreditState_IcaDelegate, true)
		return nil
	}

	// Update validators
	pool.ApplySplitDelegate(ctx, splits)
	k.updatePool(ctx, &pool)
	k.UpdateCreditStatus(ctx, poolID, creditID, types.CreditState_Success, false)

	if pool.State == types.PoolState_Closing {
		// Continue closing procedure
		// voluntary ignore errors
		if err := k.ClosePool(ctx, poolID); err != nil {
			k.Logger(ctx).With("ctx", "credit_completed", "pool_id", poolID).Error("Silently failed to continue close pool procedure: %v", err)
		}
	}

	return nil
}

// GetNextCreditID gets the next credit ID
func (k Keeper) GetNextCreditID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	nextCreditId := gogotypes.UInt64Value{}

	b := store.Get(types.NextCreditPrefix)
	if b == nil {
		panic(fmt.Errorf("getting at key (%v) should not have been nil", types.NextCreditPrefix))
	}
	k.cdc.MustUnmarshal(b, &nextCreditId)
	return nextCreditId.GetValue()
}

// GetNextCreditIdAndIncrement gets the next credit ID and store the incremented ID
func (k Keeper) GetNextCreditIdAndIncrement(ctx sdk.Context) uint64 {
	nextCreditId := k.GetNextCreditID(ctx)
	k.SetNextCreditID(ctx, nextCreditId+1)
	return nextCreditId
}

// SetNextCreditID sets next credit ID
func (k Keeper) SetNextCreditID(ctx sdk.Context, creditId uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&gogotypes.UInt64Value{Value: creditId})
	store.Set(types.NextCreditPrefix, bz)
}

// GetPoolCredit returns a credit by ID for a given poolID
func (k Keeper) GetPoolCredit(ctx sdk.Context, poolID uint64, creditID uint64) (types.Credit, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetPoolCreditKey(poolID, creditID))
	if bz == nil {
		return types.Credit{}, types.ErrCreditNotFound
	}

	var credit types.Credit
	if err := k.cdc.Unmarshal(bz, &credit); err != nil {
		return types.Credit{}, err
	}

	return credit, nil
}

// AddCredit adds a credit to a pool and account
// A new creditID is generated if not provided
// - adds it to the pool {pool_id, credit_id}
// - adds it to the account {address, pool_id, credit_id} credits
func (k Keeper) AddCredit(ctx sdk.Context, credit *types.Credit) {
	// Automatically affect ID if missing
	if credit.GetCreditId() == types.UnknownID {
		credit.CreditId = k.GetNextCreditIdAndIncrement(ctx)
	}
	// Ensure payload is valid
	if err := credit.ValidateBasic(); err != nil {
		panic(err)
	}
	// Ensure we never override an existing entity
	if _, err := k.GetPoolCredit(ctx, credit.GetPoolId(), credit.GetCreditId()); err == nil {
		panic(errorsmod.Wrapf(types.ErrEntityOverride, "ID %d", credit.GetCreditId()))
	}

	isFirstAccountCredit := !k.hasPoolCredit(ctx, credit.CreditorAddress, credit.PoolId)

	// Update pool credits
	k.setPoolCredit(ctx, credit)

	// Update account credits
	k.setAccountCredit(ctx, credit)

	// Update pool TVL and Creditors Count
	pool, err := k.GetPool(ctx, credit.PoolId)
	if err != nil {
		panic(err)
	}
	pool.TvlAmount = pool.TvlAmount.Add(credit.GetAmount().Amount)
	if credit.IsSponsor {
		if pool.SponsorshipAmount.IsNil() {
			pool.SponsorshipAmount = sdk.ZeroInt()
		}
		pool.SponsorshipAmount = pool.SponsorshipAmount.Add(credit.GetAmount().Amount)
	}

	if isFirstAccountCredit {
		// First credit for account added
		pool.CreditorsCount++
	}

	k.updatePool(ctx, &pool)
}

// RemoveCredit removes a credit from a pool
// - removes it from the {pool_id, credit_id}
// - removes it from the {address, pool_id, credit_id} credits
func (k Keeper) RemoveCredit(ctx sdk.Context, credit *types.Credit) {
	// Ensure payload is valid
	if err := credit.ValidateBasic(); err != nil {
		panic(err)
	}
	store := ctx.KVStore(k.storeKey)

	addr := sdk.MustAccAddressFromBech32(credit.CreditorAddress)
	store.Delete(types.GetPoolCreditKey(credit.PoolId, credit.CreditId))
	store.Delete(types.GetAccountPoolCreditKey(addr, credit.PoolId, credit.CreditId))

	// Update pool TVL and Creditors Count
	pool, err := k.GetPool(ctx, credit.PoolId)
	if err != nil {
		panic(err)
	}
	pool.TvlAmount = pool.TvlAmount.Sub(credit.GetAmount().Amount)
	if credit.IsSponsor {
		pool.SponsorshipAmount = pool.SponsorshipAmount.Sub(credit.GetAmount().Amount)
	}

	if !k.hasPoolCredit(ctx, credit.CreditorAddress, credit.PoolId) {
		// Last credit for account removed
		pool.CreditorsCount--
	}

	k.updatePool(ctx, &pool)
}

// UpdateCreditStatus Update a given credit status by its ID
func (k Keeper) UpdateCreditStatus(ctx sdk.Context, poolID uint64, creditID uint64, status types.CreditState, isError bool) {
	credit, err := k.GetPoolCredit(ctx, poolID, creditID)
	if err != nil {
		panic(err)
	}

	if isError {
		credit.State = types.CreditState_Failure
		credit.ErrorState = status
	} else {
		credit.State = status
		credit.ErrorState = types.CreditState_Unspecified
	}

	credit.UpdatedAtHeight = ctx.BlockHeight()
	credit.UpdatedAt = ctx.BlockTime()
	k.setAccountCredit(ctx, &credit)
	k.setPoolCredit(ctx, &credit)
}

// EditCredit edits a credit winnerAddr and sponsor mode
func (k Keeper) EditCredit(ctx sdk.Context, poolID uint64, creditID uint64, winnerAddr sdk.AccAddress, isSponsor bool) error {
	credit, err := k.GetPoolCredit(ctx, poolID, creditID)
	if err != nil {
		return err
	}

	// Get pool to grab SponsorshipAmount
	pool, err := k.GetPool(ctx, credit.PoolId)
	if err != nil {
		return err
	}

	// Check incoming sponsor mode against the credit sponsor
	if isSponsor != credit.IsSponsor {
		if isSponsor {
			pool.SponsorshipAmount = pool.SponsorshipAmount.Add(credit.GetAmount().Amount)
			k.updatePool(ctx, &pool)
		} else {
			pool.SponsorshipAmount = pool.SponsorshipAmount.Sub(credit.GetAmount().Amount)
			k.updatePool(ctx, &pool)
		}
	}

	credit.WinnerAddress = winnerAddr.String()
	credit.IsSponsor = isSponsor
	credit.UpdatedAtHeight = ctx.BlockHeight()
	credit.UpdatedAt = ctx.BlockTime()

	k.setAccountCredit(ctx, &credit)
	k.setPoolCredit(ctx, &credit)

	return nil
}

// setPoolCredit sets a credit to the pool credit key
func (k Keeper) setPoolCredit(ctx sdk.Context, credit *types.Credit) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetPoolCreditKey(credit.PoolId, credit.CreditId)
	encodedCredit := k.cdc.MustMarshal(credit)
	store.Set(key, encodedCredit)
}

// setAccountCredit set the credit for the Account credit key
func (k Keeper) setAccountCredit(ctx sdk.Context, credit *types.Credit) {
	store := ctx.KVStore(k.storeKey)
	creditorAddress := sdk.MustAccAddressFromBech32(credit.CreditorAddress)
	key := types.GetAccountPoolCreditKey(creditorAddress, credit.PoolId, credit.CreditId)
	encodedCredit := k.cdc.MustMarshal(credit)
	store.Set(key, encodedCredit)
}

// hasPoolCredit returns whether or not an account has at least one credit for a given pool
func (k Keeper) hasPoolCredit(ctx sdk.Context, address string, poolID uint64) bool {
	addr := sdk.MustAccAddressFromBech32(address)

	kvStore := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(kvStore, types.GetAccountPoolCreditsKey(addr, poolID))

	defer iterator.Close()
	return iterator.Valid()
}

// ListPoolCredits returns all credits for a given poolID
// Warning: expensive operation
func (k Keeper) ListPoolCredits(ctx sdk.Context, poolID uint64) (credits []types.Credit) {
	creditStore := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(creditStore, types.GetPoolCreditsKey(poolID))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var credit types.Credit
		k.cdc.MustUnmarshal(iterator.Value(), &credit)
		credits = append(credits, credit)
	}
	return
}

// ListAccountPoolCredits return all the credits for and address and a poolID
// Warning: expensive operation
func (k Keeper) ListAccountPoolCredits(ctx sdk.Context, addr sdk.Address, poolID uint64) (credits []types.Credit) {
	creditStore := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(creditStore, types.GetAccountPoolCreditsKey(addr, poolID))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var credit types.Credit
		k.cdc.MustUnmarshal(iterator.Value(), &credit)
		credits = append(credits, credit)
	}
	return
}

// ListAccountCredits return credits all the credits for an address
// Warning: expensive operation
func (k Keeper) ListAccountCredits(ctx sdk.Context, addr sdk.Address) (credits []types.Credit) {
	creditStore := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(creditStore, types.GetAccountCreditsKey(addr))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var credit types.Credit
		k.cdc.MustUnmarshal(iterator.Value(), &credit)
		credits = append(credits, credit)
	}
	return
}

// ListCredits return all the credits for and address
// Warning: expensive operation
func (k Keeper) ListCredits(ctx sdk.Context) (credits []types.Credit) {
	creditStore := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(creditStore, types.GetCreditsKey())
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var credit types.Credit
		k.cdc.MustUnmarshal(iterator.Value(), &credit)
		credits = append(credits, credit)
	}
	return
}

func (k Keeper) UnsafeSetUnpersistedCredits(ctx sdk.Context) int {
	i := 0
	for _, credit := range k.ListCredits(ctx) {
		k.setPoolCredit(ctx, &credit)
		k.setAccountCredit(ctx, &credit)
		i++
	}

	return i
}
