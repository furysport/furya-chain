package keeper

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogotypes "github.com/cosmos/gogoproto/types"

	"github.com/furysport/furya-chain/v2/x/millions/types"
)

// TransferPaymentToRemoteZone Transfer a payment to a remote zone
// - wait for the ICA callback to move to OnTransferPaymentToRemoteZoneCompleted
// - or go to OnTransferPaymentToRemoteZoneCompleted directly if local zone (no-op)
func (k Keeper) TransferPaymentToRemoteZone(ctx sdk.Context, poolID uint64, paymentID uint64) error {
	// Acquire pool config
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return err
	}

	// Acquire the payment
	payment, err := k.GetPoolPayment(ctx, poolID, paymentID)
	if err != nil {
		return err
	}
	if payment.State != types.PaymentState_IbcTransfer {
		return errorsmod.Wrapf(types.ErrIllegalStateOperation, "state should be %s but is %s", types.PaymentState_IbcTransfer.String(), payment.State.String())
	}

	poolRunner := k.MustGetPoolRunner(pool.PoolType)
	if err := poolRunner.TransferPaymentToRemoteZone(ctx, pool, payment); err != nil {
		// Return with error (if any) here since it is the first operation and nothing needs to be saved to state
		return err
	}
	if pool.IsLocalZone(ctx) {
		// Move instantly to next stage since local ops don't wait for callbacks
		return k.OnTransferPaymentToRemoteZoneCompleted(ctx, poolID, paymentID, false)
	}
	return nil
}

// OnTransferPaymentToRemoteZoneCompleted Acknowledge the IBC transfer to a remote zone response
// then moves to DelegatePaymentOnRemoteZone in case of success
func (k Keeper) OnTransferPaymentToRemoteZoneCompleted(ctx sdk.Context, poolID uint64, paymentID uint64, isError bool) error {
	// Acquire the payment
	payment, err := k.GetPoolPayment(ctx, poolID, paymentID)
	if err != nil {
		return err
	}

	// Verify payment state
	if payment.State != types.PaymentState_IbcTransfer {
		return errorsmod.Wrapf(types.ErrIllegalStateOperation, "state should be %s but is %s", types.PaymentState_IbcTransfer.String(), payment.State.String())
	}

	if isError {
		k.UpdatePaymentStatus(ctx, poolID, paymentID, types.PaymentState_IbcTransfer, true)
		return nil
	}

	k.UpdatePaymentStatus(ctx, poolID, paymentID, types.PaymentState_IcaDelegate, false)
	return k.DelegatePaymentOnRemoteZone(ctx, poolID, paymentID)
}

// DelegatePaymentOnRemoteZone Delegates a payment on the remote zone (put it to work)
// - wait for the ICA callback to move to OnDelegatePaymentOnRemoteZoneCompleted
// - or go to OnDelegatePaymentOnRemoteZoneCompleted directly if local zone
func (k Keeper) DelegatePaymentOnRemoteZone(ctx sdk.Context, poolID uint64, paymentID uint64) error {
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return err
	}

	payment, err := k.GetPoolPayment(ctx, poolID, paymentID)
	if err != nil {
		return err
	}
	if payment.State != types.PaymentState_IcaDelegate {
		return errorsmod.Wrapf(types.ErrIllegalStateOperation, "state should be %s but is %s", types.PaymentState_IcaDelegate.String(), payment.State.String())
	}

	// Return with error (if any) here since it is the first operation and nothing needs to be saved to state
	poolRunner := k.MustGetPoolRunner(pool.PoolType)
	splits, err := poolRunner.DelegatePaymentOnRemoteZone(ctx, pool, payment)
	if pool.IsLocalZone(ctx) {
		// Always save state in case of local zone
		// TODO: we should probably return and error here since it is atomic with the payment transaction
		return k.OnDelegatePaymentOnRemoteZoneCompleted(ctx, poolID, paymentID, splits, err != nil)
	} else if err != nil {
		// Only save state in case of error for remote zones (otherwise wait for the callback)
		return k.OnDelegatePaymentOnRemoteZoneCompleted(ctx, poolID, paymentID, splits, true)
	}
	return nil
}

// OnDelegatePaymentOnRemoteZoneCompleted Acknowledge the ICA operation to delegate a payment on the remote zone (payment has been put to work)
func (k Keeper) OnDelegatePaymentOnRemoteZoneCompleted(ctx sdk.Context, poolID uint64, paymentID uint64, splits []*types.SplitDelegation, isError bool) error {
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return err
	}
	payment, err := k.GetPoolPayment(ctx, poolID, paymentID)
	if err != nil {
		return err
	}

	// Verify payment state
	if payment.State != types.PaymentState_IcaDelegate {
		return errorsmod.Wrapf(types.ErrIllegalStateOperation, "state should be %s but is %s", types.PaymentState_IcaDelegate.String(), payment.State.String())
	}

	if isError {
		k.UpdatePaymentStatus(ctx, poolID, paymentID, types.PaymentState_IcaDelegate, true)
		return nil
	}

	// Update validators
	pool.ApplySplitDelegate(ctx, splits)
	k.updatePool(ctx, &pool)
	k.UpdatePaymentStatus(ctx, poolID, paymentID, types.PaymentState_Success, false)

	if pool.State == types.PoolState_Closing {
		// Continue closing procedure
		// voluntary ignore errors
		if err := k.ClosePool(ctx, poolID); err != nil {
			k.Logger(ctx).With("ctx", "payment_completed", "pool_id", poolID).Error("Silently failed to continue close pool procedure: %v", err)
		}
	}

	return nil
}

// GetNextPaymentID gets the next payment ID
func (k Keeper) GetNextPaymentID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	nextPaymentId := gogotypes.UInt64Value{}

	b := store.Get(types.NextPaymentPrefix)
	if b == nil {
		panic(fmt.Errorf("getting at key (%v) should not have been nil", types.NextPaymentPrefix))
	}
	k.cdc.MustUnmarshal(b, &nextPaymentId)
	return nextPaymentId.GetValue()
}

// GetNextPaymentIdAndIncrement gets the next payment ID and store the incremented ID
func (k Keeper) GetNextPaymentIdAndIncrement(ctx sdk.Context) uint64 {
	nextPaymentId := k.GetNextPaymentID(ctx)
	k.SetNextPaymentID(ctx, nextPaymentId+1)
	return nextPaymentId
}

// SetNextPaymentID sets next payment ID
func (k Keeper) SetNextPaymentID(ctx sdk.Context, paymentId uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&gogotypes.UInt64Value{Value: paymentId})
	store.Set(types.NextPaymentPrefix, bz)
}

// GetPoolPayment returns a payment by ID for a given poolID
func (k Keeper) GetPoolPayment(ctx sdk.Context, poolID uint64, paymentID uint64) (types.Payment, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetPoolPaymentKey(poolID, paymentID))
	if bz == nil {
		return types.Payment{}, types.ErrPaymentNotFound
	}

	var payment types.Payment
	if err := k.cdc.Unmarshal(bz, &payment); err != nil {
		return types.Payment{}, err
	}

	return payment, nil
}

// AddPayment adds a payment to a pool and account
// A new paymentID is generated if not provided
// - adds it to the pool {pool_id, payment_id}
// - adds it to the account {address, pool_id, payment_id} payments
func (k Keeper) AddPayment(ctx sdk.Context, payment *types.Payment) {
	// Automatically affect ID if missing
	if payment.GetPaymentId() == types.UnknownID {
		payment.PaymentId = k.GetNextPaymentIdAndIncrement(ctx)
	}
	// Ensure payload is valid
	if err := payment.ValidateBasic(); err != nil {
		panic(err)
	}
	// Ensure we never override an existing entity
	if _, err := k.GetPoolPayment(ctx, payment.GetPoolId(), payment.GetPaymentId()); err == nil {
		panic(errorsmod.Wrapf(types.ErrEntityOverride, "ID %d", payment.GetPaymentId()))
	}

	isFirstAccountPayment := !k.hasPoolPayment(ctx, payment.PaymentorAddress, payment.PoolId)

	// Update pool payments
	k.setPoolPayment(ctx, payment)

	// Update account payments
	k.setAccountPayment(ctx, payment)

	// Update pool TVL and Paymentors Count
	pool, err := k.GetPool(ctx, payment.PoolId)
	if err != nil {
		panic(err)
	}
	pool.TvlAmount = pool.TvlAmount.Add(payment.GetAmount().Amount)
	if payment.IsSponsor {
		if pool.SponsorshipAmount.IsNil() {
			pool.SponsorshipAmount = sdk.ZeroInt()
		}
		pool.SponsorshipAmount = pool.SponsorshipAmount.Add(payment.GetAmount().Amount)
	}

	if isFirstAccountPayment {
		// First payment for account added
		pool.PaymentorsCount++
	}

	k.updatePool(ctx, &pool)
}

// RemovePayment removes a payment from a pool
// - removes it from the {pool_id, payment_id}
// - removes it from the {address, pool_id, payment_id} payments
func (k Keeper) RemovePayment(ctx sdk.Context, payment *types.Payment) {
	// Ensure payload is valid
	if err := payment.ValidateBasic(); err != nil {
		panic(err)
	}
	store := ctx.KVStore(k.storeKey)

	addr := sdk.MustAccAddressFromBech32(payment.PaymentorAddress)
	store.Delete(types.GetPoolPaymentKey(payment.PoolId, payment.PaymentId))
	store.Delete(types.GetAccountPoolPaymentKey(addr, payment.PoolId, payment.PaymentId))

	// Update pool TVL and Paymentors Count
	pool, err := k.GetPool(ctx, payment.PoolId)
	if err != nil {
		panic(err)
	}
	pool.TvlAmount = pool.TvlAmount.Sub(payment.GetAmount().Amount)
	if payment.IsSponsor {
		pool.SponsorshipAmount = pool.SponsorshipAmount.Sub(payment.GetAmount().Amount)
	}

	if !k.hasPoolPayment(ctx, payment.PaymentorAddress, payment.PoolId) {
		// Last payment for account removed
		pool.PaymentorsCount--
	}

	k.updatePool(ctx, &pool)
}

// UpdatePaymentStatus Update a given payment status by its ID
func (k Keeper) UpdatePaymentStatus(ctx sdk.Context, poolID uint64, paymentID uint64, status types.PaymentState, isError bool) {
	payment, err := k.GetPoolPayment(ctx, poolID, paymentID)
	if err != nil {
		panic(err)
	}

	if isError {
		payment.State = types.PaymentState_Failure
		payment.ErrorState = status
	} else {
		payment.State = status
		payment.ErrorState = types.PaymentState_Unspecified
	}

	payment.UpdatedAtHeight = ctx.BlockHeight()
	payment.UpdatedAt = ctx.BlockTime()
	k.setAccountPayment(ctx, &payment)
	k.setPoolPayment(ctx, &payment)
}

// EditPayment edits a payment winnerAddr and sponsor mode
func (k Keeper) EditPayment(ctx sdk.Context, poolID uint64, paymentID uint64, winnerAddr sdk.AccAddress, isSponsor bool) error {
	payment, err := k.GetPoolPayment(ctx, poolID, paymentID)
	if err != nil {
		return err
	}

	// Get pool to grab SponsorshipAmount
	pool, err := k.GetPool(ctx, payment.PoolId)
	if err != nil {
		return err
	}

	// Check incoming sponsor mode against the payment sponsor
	if isSponsor != payment.IsSponsor {
		if isSponsor {
			pool.SponsorshipAmount = pool.SponsorshipAmount.Add(payment.GetAmount().Amount)
			k.updatePool(ctx, &pool)
		} else {
			pool.SponsorshipAmount = pool.SponsorshipAmount.Sub(payment.GetAmount().Amount)
			k.updatePool(ctx, &pool)
		}
	}

	payment.WinnerAddress = winnerAddr.String()
	payment.IsSponsor = isSponsor
	payment.UpdatedAtHeight = ctx.BlockHeight()
	payment.UpdatedAt = ctx.BlockTime()

	k.setAccountPayment(ctx, &payment)
	k.setPoolPayment(ctx, &payment)

	return nil
}

// setPoolPayment sets a payment to the pool payment key
func (k Keeper) setPoolPayment(ctx sdk.Context, payment *types.Payment) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetPoolPaymentKey(payment.PoolId, payment.PaymentId)
	encodedPayment := k.cdc.MustMarshal(payment)
	store.Set(key, encodedPayment)
}

// setAccountPayment set the payment for the Account payment key
func (k Keeper) setAccountPayment(ctx sdk.Context, payment *types.Payment) {
	store := ctx.KVStore(k.storeKey)
	paymentorAddress := sdk.MustAccAddressFromBech32(payment.PaymentorAddress)
	key := types.GetAccountPoolPaymentKey(paymentorAddress, payment.PoolId, payment.PaymentId)
	encodedPayment := k.cdc.MustMarshal(payment)
	store.Set(key, encodedPayment)
}

// hasPoolPayment returns whether or not an account has at least one payment for a given pool
func (k Keeper) hasPoolPayment(ctx sdk.Context, address string, poolID uint64) bool {
	addr := sdk.MustAccAddressFromBech32(address)

	kvStore := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(kvStore, types.GetAccountPoolPaymentsKey(addr, poolID))

	defer iterator.Close()
	return iterator.Valid()
}

// ListPoolPayments returns all payments for a given poolID
// Warning: expensive operation
func (k Keeper) ListPoolPayments(ctx sdk.Context, poolID uint64) (payments []types.Payment) {
	paymentStore := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(paymentStore, types.GetPoolPaymentsKey(poolID))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var payment types.Payment
		k.cdc.MustUnmarshal(iterator.Value(), &payment)
		payments = append(payments, payment)
	}
	return
}

// ListAccountPoolPayments return all the payments for and address and a poolID
// Warning: expensive operation
func (k Keeper) ListAccountPoolPayments(ctx sdk.Context, addr sdk.Address, poolID uint64) (payments []types.Payment) {
	paymentStore := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(paymentStore, types.GetAccountPoolPaymentsKey(addr, poolID))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var payment types.Payment
		k.cdc.MustUnmarshal(iterator.Value(), &payment)
		payments = append(payments, payment)
	}
	return
}

// ListAccountPayments return payments all the payments for an address
// Warning: expensive operation
func (k Keeper) ListAccountPayments(ctx sdk.Context, addr sdk.Address) (payments []types.Payment) {
	paymentStore := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(paymentStore, types.GetAccountPaymentsKey(addr))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var payment types.Payment
		k.cdc.MustUnmarshal(iterator.Value(), &payment)
		payments = append(payments, payment)
	}
	return
}

// ListPayments return all the payments for and address
// Warning: expensive operation
func (k Keeper) ListPayments(ctx sdk.Context) (payments []types.Payment) {
	paymentStore := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(paymentStore, types.GetPaymentsKey())
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var payment types.Payment
		k.cdc.MustUnmarshal(iterator.Value(), &payment)
		payments = append(payments, payment)
	}
	return
}

func (k Keeper) UnsafeSetUnpersistedPayments(ctx sdk.Context) int {
	i := 0
	for _, payment := range k.ListPayments(ctx) {
		k.setPoolPayment(ctx, &payment)
		k.setAccountPayment(ctx, &payment)
		i++
	}

	return i
}
