package v200

import (
	"reflect"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/furysport/furya-chain/v2/app/keepers"
	minttypes "github.com/furysport/furya-chain/v2/x/mint/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	tokenfactorytypes "github.com/furysport/furya-chain/v2/x/tokenfactory/types"
)

//nolint:all
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("start to run module migrations...")
		
		// Get the actual params
		icaHostParams := app.ICAHostKeeper.GetParams(ctx)

		// Patch the parameters - Enable the controller and patch the allowed messages for host
		icaControllerParams := icacontrollertypes.Params{
			ControllerEnabled: true,
		}
		icaHostParams.AllowMessages = append(icaHostParams.AllowMessages,
			// Change: added dfract and millions messages
			sdk.MsgTypeURL(&dfracttypes.MsgDeposit{}),
			sdk.MsgTypeURL(&millionstypes.MsgDeposit{}),
			sdk.MsgTypeURL(&millionstypes.MsgDepositRetry{}),
			sdk.MsgTypeURL(&millionstypes.MsgClaimPrize{}),
			sdk.MsgTypeURL(&millionstypes.MsgWithdrawDeposit{}),
			sdk.MsgTypeURL(&millionstypes.MsgWithdrawDepositRetry{}),
			sdk.MsgTypeURL(&millionstypes.MsgDrawRetry{}),
		)

		// Apply patched parameters
		app.ICAHostKeeper.SetParams(ctx, icaHostParams)
		app.ICAControllerKeeper.SetParams(ctx, icaControllerParams)

		// Set the ICA Callbacks, ICQueries and Millions modules versions so InitGenesis is not run
		icacallbacksVersion := app.mm.Modules[icacallbackstypes.ModuleName].ConsensusVersion()
		icqueriesVersion := app.mm.Modules[icqueriestypes.ModuleName].ConsensusVersion()
		millionsVersion := app.mm.Modules[millionstypes.ModuleName].ConsensusVersion()

		// Apply initial millions state
		genState := millionstypes.DefaultGenesisState()
		app.MillionsKeeper.SetParams(ctx, genState.Params)
		app.MillionsKeeper.SetNextPoolID(ctx, genState.NextPoolId)
		app.MillionsKeeper.SetNextDepositID(ctx, genState.NextDepositId)
		app.MillionsKeeper.SetNextPrizeID(ctx, genState.NextPrizeId)
		app.MillionsKeeper.SetNextWithdrawalID(ctx, genState.NextWithdrawalId)

		app.Logger().Info("v2.0.1 upgrade applied: Millions module enabled and ICA configuration updated.")

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
