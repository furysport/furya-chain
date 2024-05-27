package v210

import (
	"github.com/furysport/furya-chain/v2/app/keepers"
// 	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
// 	tmtypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/upgrade/types"
	icacallbackstypes "github.com/furysport/furya-chain/v2/x/icacallbacks/types"
	icqueriestypes "github.com/furysport/furya-chain/v2/x/icqueries/types"
	millionstypes "github.com/furysport/furya-chain/v2/x/millions/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
)

// nolint:all
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers keepers.AppKeepers,
) types.UpgradeHandler {
	return func(ctx sdk.Context, plan types.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("start to run module migrations...")

		icaHostParams := keepers.ICAHostKeeper.GetParams(ctx)

		// Patch the parameters - Enable the controller and patch the allowed messages for host
		icaControllerParams := icacontrollertypes.Params{
			ControllerEnabled: true,
		}
		icaHostParams.AllowMessages = append(icaHostParams.AllowMessages,
			// Change: added millions messages
			sdk.MsgTypeURL(&millionstypes.MsgDeposit{}),
			sdk.MsgTypeURL(&millionstypes.MsgDepositRetry{}),
			sdk.MsgTypeURL(&millionstypes.MsgClaimPrize{}),
			sdk.MsgTypeURL(&millionstypes.MsgWithdrawDeposit{}),
			sdk.MsgTypeURL(&millionstypes.MsgWithdrawDepositRetry{}),
			sdk.MsgTypeURL(&millionstypes.MsgDrawRetry{}),
		)

		keepers.ICAHostKeeper.SetParams(ctx, icaHostParams)
		keepers.ICAControllerKeeper.SetParams(ctx, icaControllerParams)
		
		// Set the ICA Callbacks, ICQueries, and Millions modules versions so InitGenesis is not run
		vm[icacallbackstypes.ModuleName] = mm.GetVersionMap()[icacallbackstypes.ModuleName]
		vm[icqueriestypes.ModuleName] = mm.GetVersionMap()[icqueriestypes.ModuleName]
		vm[millionstypes.ModuleName] = mm.GetVersionMap()[millionstypes.ModuleName]
		
		// Apply initial millions state
		genState := millionstypes.DefaultGenesisState()
		keepers.MillionsKeeper.SetParams(ctx, genState.Params)
		keepers.MillionsKeeper.SetNextPoolID(ctx, genState.NextPoolId)
		keepers.MillionsKeeper.SetNextDepositID(ctx, genState.NextDepositId)
		keepers.MillionsKeeper.SetNextPrizeID(ctx, genState.NextPrizeId)
		keepers.MillionsKeeper.SetNextWithdrawalID(ctx, genState.NextWithdrawalId)


		return mm.RunMigrations(ctx, configurator, vm)
	}
}
