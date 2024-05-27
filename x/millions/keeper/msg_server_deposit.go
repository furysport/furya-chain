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

// Credit creates a credit from the transaction message
func (k msgServer) Credit(goCtx context.Context, msg *types.MsgCredit) (*types.MsgCreditResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Get the pool to validate id and denom
	pool, err := k.GetPool(ctx, msg.GetPoolId())
	if err != nil {
		return nil, types.ErrPoolNotFound
	}

	if pool.State != types.PoolState_Ready && pool.State != types.PoolState_Paused {
		return nil, errorsmod.Wrapf(
			types.ErrInvalidPoolState, "cannot credit in pool during state %s", pool.State.String(),
		)
	}

	creditorAddr, err := sdk.AccAddressFromBech32(msg.GetCreditorAddress())
	if err != nil {
		return nil, types.ErrInvalidCreditorAddress
	}

	if msg.Amount.Denom != pool.Denom {
		return nil, types.ErrInvalidCreditDenom
	}

	// Make sure the credit is sufficient
	if msg.GetAmount().Amount.LT(pool.MinCreditAmount) {
		return nil, types.ErrInsufficientCreditAmount
	}

	winnerAddress := creditorAddr
	if strings.TrimSpace(msg.GetWinnerAddress()) != "" {
		winnerAddress, err = sdk.AccAddressFromBech32(msg.GetWinnerAddress())
		if err != nil {
			return nil, types.ErrInvalidWinnerAddress
		}
	}

	if !creditorAddr.Equals(winnerAddress) && msg.GetIsSponsor() {
		return nil, types.ErrInvalidSponsorWinnerCombo
	}

	// New credit instance
	credit := types.Credit{
		PoolId:           pool.PoolId,
		State:            types.CreditState_IbcTransfer,
		CreditorAddress: creditorAddr.String(),
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
	if err := poolRunner.SendCreditToPool(ctx, pool, credit); err != nil {
		return nil, err
	}

	// Store credit
	k.AddCredit(ctx, &credit)

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypeCredit,
			sdk.NewAttribute(types.AttributeKeyPoolID, strconv.FormatUint(credit.PoolId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreditID, strconv.FormatUint(credit.CreditId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreditor, credit.CreditorAddress),
			sdk.NewAttribute(types.AttributeKeyWinner, credit.WinnerAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, credit.Amount.String()),
			sdk.NewAttribute(types.AttributeKeySponsor, strconv.FormatBool(msg.IsSponsor)),
		),
	})

	if err := k.TransferCreditToRemoteZone(ctx, credit.GetPoolId(), credit.GetCreditId()); err != nil {
		return nil, err
	}
	return &types.MsgCreditResponse{CreditId: credit.CreditId}, nil
}

func (k msgServer) CreditRetry(goCtx context.Context, msg *types.MsgCreditRetry) (*types.MsgCreditRetryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Ensure msg received is valid
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	credit, err := k.GetPoolCredit(ctx, msg.PoolId, msg.CreditId)
	if err != nil {
		return nil, err
	}

	creditorAddr, err := sdk.AccAddressFromBech32(msg.GetCreditorAddress())
	if err != nil {
		return nil, types.ErrInvalidCreditorAddress
	}

	// Verify that the creditorAddr msg is the same as the entity creditor
	if creditorAddr.String() != credit.CreditorAddress {
		return nil, types.ErrInvalidCreditorAddress
	}

	newState := types.CreditState_Unspecified
	if credit.State == types.CreditState_IbcTransfer && ctx.BlockTime().After(credit.UpdatedAt.Add(types.IBCForceRetryNanos*time.Nanosecond)) {
		// Handle IBC stucked operation for more than 3 days (no timeout received)
		newState = types.CreditState_IbcTransfer
		if err := k.TransferCreditToRemoteZone(ctx, credit.PoolId, credit.CreditId); err != nil {
			return nil, err
		}
	} else if credit.ErrorState == types.CreditState_IbcTransfer {
		newState = types.CreditState_IbcTransfer
		k.UpdateCreditStatus(ctx, credit.PoolId, credit.CreditId, newState, false)
		if err := k.TransferCreditToRemoteZone(ctx, credit.PoolId, credit.CreditId); err != nil {
			return nil, err
		}
	} else if credit.ErrorState == types.CreditState_IcaDelegate {
		newState = types.CreditState_IcaDelegate
		k.UpdateCreditStatus(ctx, credit.PoolId, credit.CreditId, newState, false)
		if err := k.DelegateCreditOnRemoteZone(ctx, credit.PoolId, credit.CreditId); err != nil {
			return nil, err
		}
	} else {
		return nil, errorsmod.Wrapf(
			types.ErrInvalidCreditState,
			"state is %s instead of %s",
			credit.State.String(), types.CreditState_Failure.String(),
		)
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypeCreditRetry,
			sdk.NewAttribute(types.AttributeKeyState, newState.String()),
			sdk.NewAttribute(types.AttributeKeyPoolID, strconv.FormatUint(credit.PoolId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreditID, strconv.FormatUint(credit.CreditId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreditor, credit.CreditorAddress),
			sdk.NewAttribute(types.AttributeKeyWinner, credit.WinnerAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, credit.Amount.String()),
		),
	})

	return &types.MsgCreditRetryResponse{}, nil
}

func (k msgServer) CreditEdit(goCtx context.Context, msg *types.MsgCreditEdit) (*types.MsgCreditEditResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	var isSponsor bool

	// Ensure msg received is valid
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	credit, err := k.GetPoolCredit(ctx, msg.PoolId, msg.CreditId)
	if err != nil {
		return nil, err
	}

	creditorAddr, err := sdk.AccAddressFromBech32(msg.GetCreditorAddress())
	if err != nil {
		return nil, types.ErrInvalidCreditorAddress
	}

	// Verify that the creditorAddr msg is the same as the entity creditor
	if creditorAddr.String() != credit.CreditorAddress {
		return nil, types.ErrInvalidCreditorAddress
	}

	winnerAddress, err := sdk.AccAddressFromBech32(credit.WinnerAddress)
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
		isSponsor = credit.IsSponsor
	}

	if !creditorAddr.Equals(winnerAddress) && isSponsor {
		return nil, types.ErrInvalidSponsorWinnerCombo
	}

	// Edit credit
	if err := k.EditCredit(ctx, msg.PoolId, credit.CreditId, winnerAddress, isSponsor); err != nil {
		return nil, err
	}

	// Emit event
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		),
		sdk.NewEvent(
			types.EventTypeCreditEdit,
			sdk.NewAttribute(types.AttributeKeyPoolID, strconv.FormatUint(credit.PoolId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreditID, strconv.FormatUint(credit.CreditId, 10)),
			sdk.NewAttribute(types.AttributeKeyCreditor, creditorAddr.String()),
			sdk.NewAttribute(types.AttributeKeyWinner, winnerAddress.String()),
			sdk.NewAttribute(types.AttributeKeySponsor, strconv.FormatBool(isSponsor)),
		),
	})

	return &types.MsgCreditEditResponse{}, nil
}
