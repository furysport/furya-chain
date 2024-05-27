package v152

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/furysport/furya-chain/v2/x/millions/types"

	millionskeeper "github.com/furysport/furya-chain/v2/x/millions/keeper"
)

func MigrateFailedIcaUndelegationsToEpochUnbonding(ctx sdk.Context, k millionskeeper.Keeper) error {
	// Batch failed ica undelegations
	ctx.Logger().Info("Processing failed ICA undelegations and add them to pending queue...")
	if err := k.AddFailedIcaUndelegationsToEpochUnbonding(ctx); err != nil {
		return err
	}

	// Patch broken credit entities to allow for retry
	// We acquire both entities that are broken. If there is any error, it shouldn't be returned to allow for testnet execution
	// We require to ensure it here, cuz on a sub level, it would panic out in case entity does not exist
	// Both entities are flagged on error state IcaDelegate to allow them to be retried out by their owner
	poolId := uint64(2)
	ctx.Logger().Info("Patching broken credit entities...")
	_, err := k.GetPoolCredit(ctx, poolId, 429)
	if err == nil {
		k.UpdateCreditStatus(ctx, poolId, 429, types.CreditState_IcaDelegate, true)
		ctx.Logger().Info("Credit 429 patched")
	}
	_, err = k.GetPoolCredit(ctx, poolId, 450)
	if err == nil {
		k.UpdateCreditStatus(ctx, poolId, 450, types.CreditState_IcaDelegate, true)
		ctx.Logger().Info("Credit 450 patched")
	}

	return nil
}
