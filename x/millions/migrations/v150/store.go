package v150

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"

	millionskeeper "github.com/furysport/furya-chain/v2/x/millions/keeper"
	millionstypes "github.com/furysport/furya-chain/v2/x/millions/types"
)

func MigratePoolPortIdsToPortOwnerName(ctx sdk.Context, k millionskeeper.Keeper) error {
	k.IteratePools(ctx, func(pool millionstypes.Pool) bool {
		// Migrate the credit port id only if required
		var updatedCreditPortId = pool.GetIcaCreditPortId()
		if strings.HasPrefix(pool.GetIcaCreditPortId(), icatypes.ControllerPortPrefix) {
			updatedCreditPortId = strings.TrimPrefix(pool.GetIcaCreditPortId(), icatypes.ControllerPortPrefix)
		}

		// Migrate the prizepool port id only if required
		var updatedPrizePoolPortId = pool.GetIcaPrizepoolPortId()
		if strings.HasPrefix(pool.GetIcaPrizepoolPortId(), icatypes.ControllerPortPrefix) {
			updatedPrizePoolPortId = strings.TrimPrefix(pool.GetIcaPrizepoolPortId(), icatypes.ControllerPortPrefix)
		}

		// Patch our entity
		if _, err := k.UnsafeUpdatePoolPortIds(ctx, pool.GetPoolId(), updatedCreditPortId, updatedPrizePoolPortId); err != nil {
			panic(err)
		}
		ctx.Logger().Info(fmt.Sprintf("Migrated pool %d port ids | old %s / %s | new %s / %s", pool.GetPoolId(), pool.GetIcaCreditPortId(), pool.GetIcaPrizepoolPortId(), updatedCreditPortId, updatedPrizePoolPortId))
		return false
	})
	return nil
}
