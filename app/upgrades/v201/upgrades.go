package v201

import (
	"github.com/furysport/furya-chain/v2/app/keepers"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	millionstypes "github.com/furysport/furya-chain/v2/x/millions/types"
	iccontrollertypes "github.com/furysport/furya-chain/v2/x/icacontroller/types"
)

// CreateUpgradeHandler returns an UpgradeHandler for the v2.0.1 upgrade
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx types.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("start to run module migrations...")

		// Get the actual params
		icaHostParams := keepers.ICAHostKeeper.GetParams(ctx)

		// Patch the parameters - Enable the controller and patch the allowed messages for host
		icaControllerParams := iccontrollertypes.Params{
			ControllerEnabled: true,
		}
		icaHostParams.AllowMessages = append(icaHostParams.AllowMessages,
			// Change: added dfract and millions messages
			types.MsgTypeURL(&dfracttypes.MsgDeposit{}),
			types.MsgTypeURL(&millionstypes.MsgDeposit{}),
			types.MsgTypeURL(&millionstypes.MsgDepositRetry{}),
			types.MsgTypeURL(&millionstypes.MsgClaimPrize{}),
			types.MsgTypeURL(&millionstypes.MsgWithdrawDeposit{}),
			types.MsgTypeURL(&millionstypes.MsgWithdrawDepositRetry{}),
			types.MsgTypeURL(&millionstypes.MsgDrawRetry{}),
		)

		// Apply patched parameters
		keepers.ICAHostKeeper.SetParams(ctx, icaHostParams)
		keepers.ICAControllerKeeper.SetParams(ctx, icaControllerParams)

		// Apply initial millions state
		genState := millionstypes.DefaultGenesisState()
		keepers.MillionsKeeper.SetParams(ctx, genState.Params)
		keepers.MillionsKeeper.SetNextPoolID(ctx, genState.NextPoolId)
		keepers.MillionsKeeper.SetNextDepositID(ctx, genState.NextDepositId)
		keepers.MillionsKeeper.SetNextPrizeID(ctx, genState.NextPrizeId)
		keepers.MillionsKeeper.SetNextWithdrawalID(ctx, genState.NextWithdrawalId)

		keepers.Logger().Info("v2.0.1 upgrade applied: Millions module enabled and ICA configuration updated.")

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
