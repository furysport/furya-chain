package incentives

import (
	"time"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"

	furyautils "github.com/furysport/furya-chain/v2/utils"

	"github.com/furysport/furya-chain/v2/x/incentives/keeper"
	"github.com/furysport/furya-chain/v2/x/incentives/types"
)

// BeginBlocker distributes block rewards to validators who have signed the
// previous block.
func BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	ids, totalBlockReward := k.ReleaseBlockReward(ctx, req.LastCommitInfo.Votes)

	if !totalBlockReward.IsZero() {
		k.Logger(ctx).Info(
			"released incentives",
			"ids", furyautils.UintArrayToString(ids, ","),
			"amount", totalBlockReward.String(),
		)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeIncentivesReleased,
			sdk.NewAttribute(types.AttributeKeySchedules, furyautils.UintArrayToString(ids, ",")),
			sdk.NewAttribute(sdk.AttributeKeyAmount, totalBlockReward.String()),
		),
	)
}
