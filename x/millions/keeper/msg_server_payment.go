package keeper

import (
	"context"
	"strconv"
	"strings"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/furysport/furya-chain/v2/x/millions/types"
)

// Payment creates a payment from the transaction message
func (k msgServer) Payment(goCtx context.Context, msg *types.MsgPayment) (*types.MsgPaymentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get the pool to validate id and denom
	pool, err := k.GetPool(ctx, msg.GetPoolId())
	if err != nil {
		return nil, types.ErrPoolNotFound
	}

	if pool.State != types.PoolState_Ready && pool.State != types.PoolState_Paused {
		return nil, errorsmod.Wrapf(
			types.ErrInvalidPoolState, "cannot payment in pool during state %s", pool.State.String(),
		)
	}

	paymentorAddr, err := sdk.AccAddressFromBech32(msg.GetPaymentorAddress())
	if err != nil {
		return nil, types.ErrInvalidPaymentorAddress
	}

	if msg.Amount.Denom != pool.Denom {
		return nil, types.ErrInvalidPaymentDenom
	}

	// Make sure the payment is sufficient
	if msg.GetAmount().Amount.LT(pool.MinPaymentAmount) {
		return nil, types.ErrInsufficientPaymentAmount
	}

	winnerAddress := paymentorAddr
	if strings.TrimSpace(msg.GetWinnerAddress()) != "" {
		winnerAddress, err = sdk.AccAddressFromBech32(msg.GetWinnerAddress())
		if err != nil {
			return nil, types.ErrInvalidWinnerAddress
		}
	}

	if !paymentorAddr.Equals(winnerAddress) && msg.GetIsSponsor() {
		return nil, types.ErrInvalidSponsorWinnerCombo
	}

	// New payment instance
	payment := types.Payment{
		PoolId:           pool.PoolId,
		State:            types.PaymentState_IbcTransfer,
		PaymentorAddress: paymentorAddr.String(),
		Amount:           msg.Amount,
		WinnerAddress:    winnerAddress.String(),
		IsSponsor:        msg.GetIsSponsor(),
		CreatedAtHeight:  ctx.BlockHeight(),
		UpdatedAtHeight:  ctx.BlockHeight(),
		CreatedAt:        ctx.BlockTime(),
		UpdatedAt:        ctx.BlockTime(),
	}

	// Move funds
	poolRunner, err := k.GetPoolRunner(pool.PoolType)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidPoolType, err.Error())
	}
	if err := poolRunner.SendPaymentToPool(ctx, pool, payment); err != nil {
		return nil, err
	}

	// Store payment
	k.AddPayment(ctx, &payment)

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypePayment,
			sdk.NewAttribute(types.AttributeKeyPoolID, strconv.FormatUint(payment.PoolId, 10)),
			sdk.NewAttribute(types.AttributeKeyPaymentID, strconv.FormatUint(payment.PaymentId, 10)),
			sdk.NewAttribute(types.AttributeKeyPaymentor, payment.PaymentorAddress),
			sdk.NewAttribute(types.AttributeKeyWinner, payment.WinnerAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, payment.Amount.String()),
			sdk.NewAttribute(types.AttributeKeySponsor, strconv.FormatBool(msg.IsSponsor)),
		),
	})

	if err := k.TransferPaymentToRemoteZone(ctx, payment.GetPoolId(), payment.GetPaymentId()); err != nil {
		return nil, err
	}
	return &types.MsgPaymentResponse{PaymentId: payment.PaymentId}, nil
}

func (k msgServer) PaymentRetry(goCtx context.Context, msg *types.MsgPaymentRetry) (*types.MsgPaymentRetryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Ensure msg received is valid
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	payment, err := k.GetPoolPayment(ctx, msg.PoolId, msg.PaymentId)
	if err != nil {
		return nil, err
	}

	paymentorAddr, err := sdk.AccAddressFromBech32(msg.GetPaymentorAddress())
	if err != nil {
		return nil, types.ErrInvalidPaymentorAddress
	}

	// Verify that the paymentorAddr msg is the same as the entity paymentor
	if paymentorAddr.String() != payment.PaymentorAddress {
		return nil, types.ErrInvalidPaymentorAddress
	}

	newState := types.PaymentState_Unspecified
	if payment.State == types.PaymentState_IbcTransfer && ctx.BlockTime().After(payment.UpdatedAt.Add(types.IBCForceRetryNanos*time.Nanosecond)) {
		// Handle IBC stucked operation for more than 3 days (no timeout received)
		newState = types.PaymentState_IbcTransfer
		if err := k.TransferPaymentToRemoteZone(ctx, payment.PoolId, payment.PaymentId); err != nil {
			return nil, err
		}
	} else if payment.ErrorState == types.PaymentState_IbcTransfer {
		newState = types.PaymentState_IbcTransfer
		k.UpdatePaymentStatus(ctx, payment.PoolId, payment.PaymentId, newState, false)
		if err := k.TransferPaymentToRemoteZone(ctx, payment.PoolId, payment.PaymentId); err != nil {
			return nil, err
		}
	} else if payment.ErrorState == types.PaymentState_IcaDelegate {
		newState = types.PaymentState_IcaDelegate
		k.UpdatePaymentStatus(ctx, payment.PoolId, payment.PaymentId, newState, false)
		if err := k.DelegatePaymentOnRemoteZone(ctx, payment.PoolId, payment.PaymentId); err != nil {
			return nil, err
		}
	} else {
		return nil, errorsmod.Wrapf(
			types.ErrInvalidPaymentState,
			"state is %s instead of %s",
			payment.State.String(), types.PaymentState_Failure.String(),
		)
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypePaymentRetry,
			sdk.NewAttribute(types.AttributeKeyState, newState.String()),
			sdk.NewAttribute(types.AttributeKeyPoolID, strconv.FormatUint(payment.PoolId, 10)),
			sdk.NewAttribute(types.AttributeKeyPaymentID, strconv.FormatUint(payment.PaymentId, 10)),
			sdk.NewAttribute(types.AttributeKeyPaymentor, payment.PaymentorAddress),
			sdk.NewAttribute(types.AttributeKeyWinner, payment.WinnerAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, payment.Amount.String()),
		),
	})

	return &types.MsgPaymentRetryResponse{}, nil
}

func (k msgServer) PaymentEdit(goCtx context.Context, msg *types.MsgPaymentEdit) (*types.MsgPaymentEditResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var isSponsor bool

	// Ensure msg received is valid
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	payment, err := k.GetPoolPayment(ctx, msg.PoolId, msg.PaymentId)
	if err != nil {
		return nil, err
	}

	paymentorAddr, err := sdk.AccAddressFromBech32(msg.GetPaymentorAddress())
	if err != nil {
		return nil, types.ErrInvalidPaymentorAddress
	}

	// Verify that the paymentorAddr msg is the same as the entity paymentor
	if paymentorAddr.String() != payment.PaymentorAddress {
		return nil, types.ErrInvalidPaymentorAddress
	}

	winnerAddress, err := sdk.AccAddressFromBech32(payment.WinnerAddress)
	if err != nil {
		return nil, types.ErrInvalidWinnerAddress
	}

	if strings.TrimSpace(msg.GetWinnerAddress()) != "" {
		winnerAddress, err = sdk.AccAddressFromBech32(msg.GetWinnerAddress())
		if err != nil {
			return nil, types.ErrInvalidWinnerAddress
		}
	}

	if msg.IsSponsor != nil {
		isSponsor = msg.IsSponsor.Value
	} else {
		isSponsor = payment.IsSponsor
	}

	if !paymentorAddr.Equals(winnerAddress) && isSponsor {
		return nil, types.ErrInvalidSponsorWinnerCombo
	}

	// Edit payment
	if err := k.EditPayment(ctx, msg.PoolId, payment.PaymentId, winnerAddress, isSponsor); err != nil {
		return nil, err
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypePaymentEdit,
			sdk.NewAttribute(types.AttributeKeyPoolID, strconv.FormatUint(payment.PoolId, 10)),
			sdk.NewAttribute(types.AttributeKeyPaymentID, strconv.FormatUint(payment.PaymentId, 10)),
			sdk.NewAttribute(types.AttributeKeyPaymentor, paymentorAddr.String()),
			sdk.NewAttribute(types.AttributeKeyWinner, winnerAddress.String()),
			sdk.NewAttribute(types.AttributeKeySponsor, strconv.FormatBool(isSponsor)),
		),
	})

	return &types.MsgPaymentEditResponse{}, nil
}
