package types

import (
	time "time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MinAcceptablePaymentAmount        = 1_000
	MinAcceptableDrawDelta            = 1 * time.Hour
	MinAcceptablePaymentDrawDelta     = 1 * time.Minute
	MinAcceptablePrizeExpirationDelta = 1 * time.Hour

	defaultMinPaymentAmount        = 100_000
	defaultMaxPrizeStrategyBatches = 10
	defaultMaxPrizeBatchQuantity   = 1000
	defaultMinDrawScheduleDelta    = 1 * time.Hour
	defaultMaxDrawScheduleDelta    = 366 * 24 * time.Hour // 366 days
	defaultPrizeExpirationDelta    = 30 * 24 * time.Hour  // 30 days
	defaultMinPaymentDrawDelta     = 5 * time.Minute
)

func DefaultParams() Params {
	return Params{
		MinPaymentAmount:        sdk.NewInt(defaultMinPaymentAmount),
		MaxPrizeStrategyBatches: defaultMaxPrizeStrategyBatches,
		MaxPrizeBatchQuantity:   defaultMaxPrizeBatchQuantity,
		MinDrawScheduleDelta:    defaultMinDrawScheduleDelta,
		MaxDrawScheduleDelta:    defaultMaxDrawScheduleDelta,
		PrizeExpirationDelta:    defaultPrizeExpirationDelta,
		MinPaymentDrawDelta:     defaultMinPaymentDrawDelta,
	}
}

func (p *Params) ValidateBasics() error {
	if p.MinPaymentAmount.IsNil() {
		return errorsmod.Wrapf(ErrInvalidParams, "min payment amount required")
	} else if p.MinPaymentAmount.LT(sdk.NewInt(MinAcceptablePaymentAmount)) {
		return errorsmod.Wrapf(ErrInvalidParams, "min payment amount must be gte %d, got: %d", MinAcceptablePaymentAmount, p.MinPaymentAmount.Int64())
	}
	if p.MaxPrizeStrategyBatches <= 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "max prize strategy batches must be gt 0, got: %d", p.MaxPrizeStrategyBatches)
	}
	if p.MaxPrizeBatchQuantity <= 0 {
		return errorsmod.Wrapf(ErrInvalidParams, "max prize batch quantity must be gt 0, got: %d", p.MaxPrizeBatchQuantity)
	}
	if p.MinDrawScheduleDelta < MinAcceptableDrawDelta {
		return errorsmod.Wrapf(ErrInvalidParams, "min draw schedule delta must be gte %s, got: %s", MinAcceptableDrawDelta.String(), p.MinDrawScheduleDelta.String())
	}
	if p.MaxDrawScheduleDelta < MinAcceptableDrawDelta {
		return errorsmod.Wrapf(ErrInvalidParams, "max draw schedule delta must be gte %s, got %s", MinAcceptableDrawDelta.String(), p.MaxDrawScheduleDelta.String())
	}
	if p.MaxDrawScheduleDelta < p.MinDrawScheduleDelta {
		return errorsmod.Wrapf(ErrInvalidParams, "max draw schedule delta must be gte min draw schedule delta")
	}
	if p.PrizeExpirationDelta < MinAcceptablePrizeExpirationDelta {
		return errorsmod.Wrapf(ErrInvalidParams, "prize clawback delta must be gte %s, got: %s", MinAcceptablePrizeExpirationDelta.String(), p.PrizeExpirationDelta.String())
	}
	if p.MinPaymentDrawDelta < MinAcceptablePaymentDrawDelta {
		return errorsmod.Wrapf(ErrInvalidParams, "min payment draw delta must be gte %s, got: %s", MinAcceptablePaymentDrawDelta.String(), p.MinPaymentDrawDelta.String())
	}
	return nil
}
