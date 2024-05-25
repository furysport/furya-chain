package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/furysport/furya-chain/v2/x/millions/types"
)

// WithdrawPayment withdraw a payment from the transaction message
func (k msgServer) WithdrawPayment(goCtx context.Context, msg *types.MsgWithdrawPayment) (*types.MsgWithdrawPaymentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	pool, err := k.GetPool(ctx, msg.PoolId)
	if err != nil {
		return nil, types.ErrPoolNotFound
	}

	// Get the pool payment
	payment, err := k.GetPoolPayment(ctx, msg.PoolId, msg.PaymentId)
	if err != nil {
		return nil, types.ErrPaymentNotFound
	}

	// Verify addresses
	paymentorAddr, err := sdk.AccAddressFromBech32(msg.PaymentorAddress)
	if err != nil {
		return nil, types.ErrInvalidPaymentorAddress
	}

	_, toAddr, err := pool.AccAddressFromBech32(msg.ToAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidDestinationAddress, err.Error())
	}

	// Verify that the incoming message withdrawAddr matches the payment paymentor_address
	if paymentorAddr.String() != payment.PaymentorAddress {
		return nil, types.ErrInvalidWithdrawalPaymentorAddress
	}

	// Verify that the payment can be withdrawn
	if payment.State != types.PaymentState_Success {
		return nil, types.ErrInvalidPaymentState
	}

	// Prepare the withdrawal instance
	withdrawal := types.Withdrawal{
		PoolId:           msg.PoolId,
		PaymentId:        msg.PaymentId,
		WithdrawalId:     k.GetNextWithdrawalIdAndIncrement(ctx),
		State:            types.WithdrawalState_Pending,
		PaymentorAddress: payment.PaymentorAddress,
		ToAddress:        toAddr.String(),
		Amount:           payment.Amount,
		CreatedAtHeight:  ctx.BlockHeight(),
		UpdatedAtHeight:  ctx.BlockHeight(),
		CreatedAt:        ctx.BlockTime(),
		UpdatedAt:        ctx.BlockTime(),
	}

	// Adds the withdrawal and remove the payment
	k.AddWithdrawal(ctx, withdrawal)
	k.RemovePayment(ctx, &payment)

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypeWithdrawPayment,
			sdk.NewAttribute(types.AttributeKeyPoolID, strconv.FormatUint(withdrawal.PoolId, 10)),
			sdk.NewAttribute(types.AttributeKeyWithdrawalID, strconv.FormatUint(withdrawal.WithdrawalId, 10)),
			sdk.NewAttribute(types.AttributeKeyPaymentID, strconv.FormatUint(payment.PaymentId, 10)),
			sdk.NewAttribute(types.AttributeKeyPaymentor, withdrawal.PaymentorAddress),
			sdk.NewAttribute(types.AttributeKeyRecipient, withdrawal.ToAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, withdrawal.Amount.String()),
		),
	})

	// Add epoch unbonding
	if err := k.AddEpochUnbonding(ctx, withdrawal, false); err != nil {
		return nil, err
	}

	return &types.MsgWithdrawPaymentResponse{WithdrawalId: withdrawal.WithdrawalId}, err
}

// WithdrawPaymentRetry allows the user to manually trigger the withdrawal of their payment in case of failure
func (k msgServer) WithdrawPaymentRetry(goCtx context.Context, msg *types.MsgWithdrawPaymentRetry) (*types.MsgWithdrawPaymentRetryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Ensure msg received is valid
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	withdrawal, err := k.GetPoolWithdrawal(ctx, msg.PoolId, msg.WithdrawalId)
	if err != nil {
		return nil, err
	}

	paymentorAddr, err := sdk.AccAddressFromBech32(msg.PaymentorAddress)
	if err != nil {
		return nil, types.ErrInvalidPaymentorAddress
	}

	// Verify that the paymentorAddress msg is the same as the entity pool withdrawal paymentor
	if paymentorAddr.String() != withdrawal.PaymentorAddress {
		return nil, types.ErrInvalidPaymentorAddress
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
			types.EventTypeWithdrawPaymentRetry,
			sdk.NewAttribute(types.AttributeKeyState, newState.String()),
			sdk.NewAttribute(types.AttributeKeyPoolID, strconv.FormatUint(withdrawal.PoolId, 10)),
			sdk.NewAttribute(types.AttributeKeyWithdrawalID, strconv.FormatUint(withdrawal.WithdrawalId, 10)),
			sdk.NewAttribute(types.AttributeKeyPaymentor, withdrawal.PaymentorAddress),
			sdk.NewAttribute(types.AttributeKeyRecipient, withdrawal.ToAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, withdrawal.Amount.String()),
		),
	})

	return &types.MsgWithdrawPaymentRetryResponse{}, nil
}
