package v2

import (
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	feeburntypes "github.com/furysport/furya-chain/v2/x/feeburn/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

// CreateUpgradeHandler that migrates the chain from v3.0.2 to v4.1.2
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	clientKeeper clientkeeper.Keeper,
	paramsKeeper paramskeeper.Keeper,
	consensusParamsKeeper consensuskeeper.Keeper,
	icacontrollerKeeper icacontrollerkeeper.Keeper,
	accountKeeper authkeeper.AccountKeeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// READ: https://github.com/cosmos/cosmos-sdk/blob/v0.47.4/UPGRADING.md#xconsensus
		baseAppLegacySS := paramsKeeper.Subspace(baseapp.Paramspace).
			WithKeyTable(paramstypes.ConsensusParamsKeyTable())
		baseapp.MigrateParams(ctx, baseAppLegacySS, &consensusParamsKeeper)

		// READ: https://github.com/cosmos/ibc-go/blob/v7.2.0/docs/migrations/v7-to-v7_1.md#chains
		params := clientKeeper.GetParams(ctx)
		params.AllowedClients = append(params.AllowedClients, ibcexported.Localhost)
		clientKeeper.SetParams(ctx, params)

		// READ: https://github.com/terra-money/core/issues/166
		icacontrollerKeeper.SetParams(ctx, icacontrollertypes.DefaultParams())

		// Burning module permissions
		moduleAccI := accountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
		moduleAcc := moduleAccI.(*authtypes.ModuleAccount)
		moduleAcc.Permissions = []string{authtypes.Burner}
		accountKeeper.SetModuleAccount(ctx, moduleAcc)

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
