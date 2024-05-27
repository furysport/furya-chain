package types

import (
	time "time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	MinAcceptableCreditAmount        = 1_000
	MinAcceptableDrawDelta            = 1 * time.Hour
	MinAcceptableCreditDrawDelta     = 1 * time.Minute
	MinAcceptablePrizeExpirationDelta = 1 * time.Hour

	defaultMinCreditAmount        = 100_000
	defaultMaxPrizeStrategyBatches = 10
	defaultMaxPrizeBatchQuantity   = 1000
	defaultMinDrawScheduleDelta    = 1 * time.Hour
	defaultMaxDrawScheduleDelta    = 366 * 24 * time.Hour // 366 days
	defaultPrizeExpirationDelta    = 30 * 24 * time.Hour  // 30 days
	defaultMinCreditDrawDelta     = 5 * time.Minute
)

func DefaultParams() Params {
	return Params{
		MinCreditAmount:        sdk.NewInt(defaultMinCreditAmount),
		MaxPrizeStrategyBatches: defaultMaxPrizeStrategyBatches,
		MaxPrizeBatchQuantity:   defaultMaxPrizeBatchQuantity,
		MinDrawScheduleDelta:    defaultMinDrawScheduleDelta,
		MaxDrawScheduleDelta:    defaultMaxDrawScheduleDelta,
		PrizeExpirationDelta:    defaultPrizeExpirationDelta,
		MinCreditDrawDelta:     defaultMinCreditDrawDelta,
	}
}

func (p *Params) ValidateBasics() error {
	if p.MinCreditAmount.IsNil() {
		return errorsmod.Wrapf(ErrInvalidParams, "min credit amount required")
	} else if p.MinCreditAmount.LT(sdk.NewInt(MinAcceptableCreditAmount)) {
		return errorsmod.Wrapf(ErrInvalidParams, "min credit amount must be gte %d, got: %d", MinAcceptableCreditAmount, p.MinCreditAmount.Int64())
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
	if p.MinCreditDrawDelta < MinAcceptableCreditDrawDelta {
		return errorsmod.Wrapf(ErrInvalidParams, "min credit draw delta must be gte %s, got: %s", MinAcceptableCreditDrawDelta.String(), p.MinCreditDrawDelta.String())
	}
	return nil
}
