package keeper

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/furysport/furya-chain/v2/x/millions/types"
)

func (k msgServer) RestoreInterchainAccounts(goCtx context.Context, msg *types.MsgRestoreInterchainAccounts) (*types.MsgRestoreInterchainAccountsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Grab our pool instance, otherwise just return error
	pool, err := k.GetPool(ctx, msg.GetPoolId())
	if err != nil {
		return nil, types.ErrPoolNotFound
	}

	// We always want our pool to be ready
	if pool.State == types.PoolState_Created || pool.State == types.PoolState_Unspecified {
		return nil, types.ErrPoolNotReady
	}

	// Grab the app version to use
	appVersion, err := k.getPoolAppVersion(ctx, pool)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrFailedToRestorePool, err.Error())
	}

	// We don't want to run this against a local pool
	if pool.IsLocalZone(ctx) {
		return nil, errorsmod.Wrapf(types.ErrFailedToRestorePool, "Pool is local")
	}

	// Regenerate our credit port only if required (open active channel not found)
	icaCreditPortName := string(types.NewPoolName(pool.GetPoolId(), types.ICATypeCredit))
	if pool.GetIcaCreditPortId() == "" {
		// Unusual case - no port ID was set at Pool registration time (ex: genesis import without portID)
		pool.IcaCreditPortId = icaCreditPortName
		k.updatePool(ctx, &pool)
	}
	_, found := k.ICAControllerKeeper.GetOpenActiveChannel(ctx, pool.GetConnectionId(), pool.GetIcaCreditPortId())
	if !found {
		if err := k.ICAControllerKeeper.RegisterInterchainAccount(ctx, pool.GetConnectionId(), icaCreditPortName, appVersion); err != nil {
			return nil, errorsmod.Wrapf(types.ErrFailedToRestorePool, fmt.Sprintf("Unable to trigger credit account registration, err: %s", err.Error()))
		}
		k.restoreICACreditEntities(ctx, pool.GetPoolId())
		// Exit to prevent a double channel registration which might create unpredictable behaviours
		// The registration of the ICA PrizePool will either be automatically triggered once the ICA Credit gets ACK (see keeper.OnSetupPoolICACompleted)
		// or can be restored by calling this method again
		return &types.MsgRestoreInterchainAccountsResponse{}, nil
	}

	// Regenerate our prizepool port only if required (open active channel not found)
	icaPrizePoolPortName := string(types.NewPoolName(pool.GetPoolId(), types.ICATypePrizePool))
	if pool.GetIcaPrizepoolPortId() == "" {
		// Unusual case - no port ID was set at Pool ICA Credit ACK time
		pool.IcaPrizepoolPortId = icaPrizePoolPortName
		k.updatePool(ctx, &pool)
	}
	_, found = k.ICAControllerKeeper.GetOpenActiveChannel(ctx, pool.GetConnectionId(), pool.GetIcaPrizepoolPortId())
	if !found {
		if err := k.ICAControllerKeeper.RegisterInterchainAccount(ctx, pool.GetConnectionId(), icaPrizePoolPortName, appVersion); err != nil {
			return nil, errorsmod.Wrapf(types.ErrFailedToRestorePool, fmt.Sprintf("Unable to trigger prizepool account registration, err: %s", err.Error()))
		}
		k.restoreICAPrizePoolEntities(ctx, pool.GetPoolId())
	}

	return &types.MsgRestoreInterchainAccountsResponse{}, nil
}

// restoreICACreditEntities restores all blocked entities by putting them in error state instead of locked state
// thus leaving users the opportunity to retry the operation
func (k msgServer) restoreICACreditEntities(ctx sdk.Context, poolID uint64) {
	// Restore credits ICA locked operations on ICACredit account
	credits := k.Keeper.ListPoolCredits(ctx, poolID)
	for _, d := range credits {
		if d.State == types.CreditState_IcaDelegate {
			d.ErrorState = d.State
			d.State = types.CreditState_Failure
			k.Keeper.setPoolCredit(ctx, &d)
			k.Keeper.setAccountCredit(ctx, &d)
		}
	}
	// Restore withdrawals ICA locked operations on ICACredit account
	withdrawals := k.Keeper.ListPoolWithdrawals(ctx, poolID)
	for _, w := range withdrawals {
		if w.State == types.WithdrawalState_IcaUndelegate || w.State == types.WithdrawalState_IbcTransfer {
			w.ErrorState = w.State
			w.State = types.WithdrawalState_Failure
			k.Keeper.setPoolWithdrawal(ctx, w)
			k.Keeper.setAccountWithdrawal(ctx, w)
		}
	}
	// Restore draws ICA locked operations on ICACredit account
	draws := k.Keeper.ListPoolDraws(ctx, poolID)
	for _, d := range draws {
		if d.State == types.DrawState_IcaWithdrawRewards || d.State == types.DrawState_IbcTransfer {
			d.ErrorState = d.State
			d.State = types.DrawState_Failure
			k.Keeper.SetPoolDraw(ctx, d)
		}
	}
}

// restoreICAPrizePoolEntities restores all blocked entities by putting them in error state instead of locked state
// thus leaving users the opportunity to retry the operation
func (k msgServer) restoreICAPrizePoolEntities(ctx sdk.Context, poolID uint64) {
	// Restore draws ICQ locked operations on ICAPrizePool account
	draws := k.Keeper.ListPoolDraws(ctx, poolID)
	for _, d := range draws {
		if d.State == types.DrawState_IcqBalance {
			d.ErrorState = d.State
			d.State = types.DrawState_Failure
			k.Keeper.SetPoolDraw(ctx, d)
		}
	}
}
