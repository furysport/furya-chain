package v200

import (
	"reflect"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/furysport/furya-chain/app/keepers"
	minttypes "github.com/furysport/furya-chain/x/mint/types"
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
	tokenfactorytypes "github.com/furysport/furya-chain/x/tokenfactory/types"
)

//nolint:all
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		ctx.Logger().Info("start to run module migrations...")

		for _, subspace := range keepers.ParamsKeeper.GetSubspaces() {
			subspace := subspace

			var keyTable paramstypes.KeyTable
			switch subspace.Name() {
			case authtypes.ModuleName:
				keyTable = authtypes.ParamKeyTable()
			case banktypes.ModuleName:
				keyTable = banktypes.ParamKeyTable()
			case stakingtypes.ModuleName:
				keyTable = stakingtypes.ParamKeyTable()
			case slashingtypes.ModuleName:
				keyTable = slashingtypes.ParamKeyTable()
			case crisistypes.ModuleName:
				keyTable = crisistypes.ParamKeyTable()
			case govtypes.ModuleName:
				keyTable = govv1.ParamKeyTable()
			case distrtypes.ModuleName:
				keyTable = distrtypes.ParamKeyTable()
			case wasmtypes.ModuleName:
				keyTable = wasmtypes.ParamKeyTable()
			case minttypes.ModuleName:
				keyTable = minttypes.ParamKeyTable()
			}

			if !subspace.HasKeyTable() {
				subspace.WithKeyTable(keyTable)
			}
		}

		// Mint module params update
		params := minttypes.Params{}
		params.BlocksPerYear = 5733818
		params.TotalBurntAmount = []sdk.Coin{sdk.NewInt64Coin("ufury", 118_547_020000)}
		subspace, ok := keepers.ParamsKeeper.GetSubspace(minttypes.ModuleName)
		if !ok {
			panic("invalid mint module subspace")
		}
		v := reflect.Indirect(reflect.ValueOf(params.BlocksPerYear)).Interface()
		subspace.Set(ctx, minttypes.KeyBlocksPerYear, v)
		v = reflect.Indirect(reflect.ValueOf(params.TotalBurntAmount)).Interface()
		subspace.Set(ctx, minttypes.KeyTotalBurntAmount, v)

		cp := tmtypes.DefaultConsensusParams().ToProto()
		keepers.ConsensusParamsKeeper.Set(ctx, &tmproto.ConsensusParams{
			Block:     cp.Block,
			Validator: cp.Validator,
			Evidence:  cp.Evidence,
			Version:   cp.Version,
		})

		// Migrate Tendermint consensus parameters from x/params module to a deprecated x/consensus module.
		// The old params module is required to still be imported in your app.go in order to handle this migration.
		baseAppLegacySS := keepers.ParamsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable())
		baseapp.MigrateParams(ctx, baseAppLegacySS, &keepers.ConsensusParamsKeeper)

		// Run migrations before applying any other state changes.
		// NOTE: DO NOT PUT ANY STATE CHANGES BEFORE RunMigrations().
		versionMap, err := mm.RunMigrations(ctx, cfg, fromVM)
		if err != nil {
			return nil, err
		}

		// explicitly update the IBC 02-client params, adding the localhost client type
		params := keepers.IBCKeeper.ClientKeeper.GetParams(ctx)
		params.AllowedClients = append(params.AllowedClients, exported.Localhost)
		keepers.IBCKeeper.ClientKeeper.SetParams(ctx, params)

		// set proposal's minimum initial deposit to 20%
		govParams := keepers.GovKeeper.GetParams(ctx)
		govParams.MinInitialDepositRatio = sdk.NewDec(20).Quo(sdk.NewDec(100)).String()
		if err := keepers.GovKeeper.SetParams(ctx, govParams); err != nil {
			return nil, err
		}

		// set validator's minimum commission to 5%
		stakingParams := keepers.StakingKeeper.GetParams(ctx)
		stakingParams.MinCommissionRate = sdk.NewDecWithPrec(5, 2)
		err = keepers.StakingKeeper.SetParams(ctx, stakingParams)
		if err != nil {
			return nil, err
		}

		// set token-factory denom creation fee
		err = keepers.TokenFactoryKeeper.SetParams(ctx, tokenfactorytypes.DefaultParams())
		if err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
