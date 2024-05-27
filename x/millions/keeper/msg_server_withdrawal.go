package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/furysport/furya-chain/v2/x/millions/types"
)

// WithdrawCredit withdraw a credit from the transaction message
func (k msgServer) WithdrawCredit(goCtx context.Context, msg *types.MsgWithdrawCredit) (*types.MsgWithdrawCreditResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pool, err := k.GetPool(ctx, msg.PoolId)
	if err != nil {
		return nil, types.ErrPoolNotFound
	}

	// Get the pool credit
	credit, err := k.GetPoolCredit(ctx, msg.PoolId, msg.CreditId)
	if err != nil {
		return nil, types.ErrCreditNotFound
	}

	// Verify addresses
	creditorAddr, err := sdk.AccAddressFromBech32(msg.CreditorAddress)
	if err != nil {
		return nil, types.ErrInvalidCreditorAddress
	}

	_, toAddr, err := pool.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidDestinationAddress, err.Error())
	}

	// Verify that the incoming message withdrawAddr matches the credit creditor_address
	if creditorAddr.String() != credit.CreditorAddress {
		return nil, types.ErrInvalidWithdrawalCreditorAddress
	}

	// Verify that the credit can be withdrawn
	if credit.State != types.CreditState_Success {
		return nil, types.ErrInvalidCreditState
	}

	// Prepare the withdrawal instance
	withdrawal := types.Withdrawal{
		PoolId:           msg.PoolId,
		CreditId:        msg.CreditId,
		WithdrawalId:     k.GetNextWithdrawalIdAndIncrement(ctx),
		State:            types.WithdrawalState_Pending,
		CreditorAddress: credit.CreditorAddress,
		ToAddress:        toAddr.String(),
		Amount:           credit.Amount,
		CreatedAtHeight:  ctx.BlockHeight(),
		UpdatedAtHeight:  ctx.BlockHeight(),
		CreatedAt:        ctx.BlockTime(),
		UpdatedAt:        ctx.BlockTime(),
	}

	// Adds the withdrawal and remove the credit
	k.AddWithdrawal(ctx, withdrawal)
	k.RemoveCredit(ctx, &credit)

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypeWithdrawCredit,
			sdk.NewAttribute(types.AttributeKeyPoolID, strconv.FormatUint(withdrawal.PoolId, 10)),
			sdk.NewAttribute(types.AttributeKeyWithdrawalID, strconv.FormatUint(withdrawal.WithdrawalId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreditID, strconv.FormatUint(credit.CreditId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreditor, withdrawal.CreditorAddress),
			sdk.NewAttribute(types.AttributeKeyRecipient, withdrawal.ToAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, withdrawal.Amount.String()),
		),
	})

	// Add epoch unbonding
	if err := k.AddEpochUnbonding(ctx, withdrawal, false); err != nil {
		return nil, err
	}

	return &types.MsgWithdrawCreditResponse{WithdrawalId: withdrawal.WithdrawalId}, err
}

// WithdrawCreditRetry allows the user to manually trigger the withdrawal of their credit in case of failure
func (k msgServer) WithdrawCreditRetry(goCtx context.Context, msg *types.MsgWithdrawCreditRetry) (*types.MsgWithdrawCreditRetryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Ensure msg received is valid
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	withdrawal, err := k.GetPoolWithdrawal(ctx, msg.PoolId, msg.WithdrawalId)
	if err != nil {
		return nil, err
	}

	creditorAddr, err := sdk.AccAddressFromBech32(msg.CreditorAddress)
	if err != nil {
		return nil, types.ErrInvalidCreditorAddress
	}

	// Verify that the creditorAddress msg is the same as the entity pool withdrawal creditor
	if creditorAddr.String() != withdrawal.CreditorAddress {
		return nil, types.ErrInvalidCreditorAddress
	}

	// State should be set to failure in order to retry something
	if withdrawal.State != types.WithdrawalState_Failure {
		return nil, errorsmod.Wrapf(
			types.ErrInvalidWithdrawalState,
			"state is %s instead of %s",
			withdrawal.State.String(), types.WithdrawalState_Failure.String(),
		)
	}

	newState := types.WithdrawalState_Unspecified
	if withdrawal.ErrorState == types.WithdrawalState_IbcTransfer {
		newState = types.WithdrawalState_IbcTransfer
		k.UpdateWithdrawalStatus(ctx, withdrawal.PoolId, withdrawal.WithdrawalId, newState, withdrawal.UnbondingEndsAt, false)
		if err := k.TransferWithdrawalToRecipient(ctx, withdrawal.PoolId, withdrawal.WithdrawalId); err != nil {
			return nil, err
		}
	} else if withdrawal.ErrorState == types.WithdrawalState_IcaUndelegate {
		// Only possible following a restore account which put entities in this error state
		// otherwise the retry is automatically triggered by the ICA callback
		err := k.AddEpochUnbonding(ctx, withdrawal, true)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errorsmod.Wrapf(
			types.ErrInvalidWithdrawalState,
			"error_state is %s instead of %s or %s",
			withdrawal.ErrorState.String(), types.WithdrawalState_IcaUndelegate.String(), types.WithdrawalState_IbcTransfer.String(),
		)
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypeWithdrawCreditRetry,
			sdk.NewAttribute(types.AttributeKeyState, newState.String()),
			sdk.NewAttribute(types.AttributeKeyPoolID, strconv.FormatUint(withdrawal.PoolId, 10)),
			sdk.NewAttribute(types.AttributeKeyWithdrawalID, strconv.FormatUint(withdrawal.WithdrawalId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreditor, withdrawal.CreditorAddress),
			sdk.NewAttribute(types.AttributeKeyRecipient, withdrawal.ToAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, withdrawal.Amount.String()),
		),
	})

	return &types.MsgWithdrawCreditRetryResponse{}, nil
}
