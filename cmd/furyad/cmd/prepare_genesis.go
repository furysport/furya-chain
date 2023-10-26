package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	appparams "github.com/furysport/fury-chain/app/params"
	"github.com/furysport/fury-chain/x/airdrop/types"
	airdroptypes "github.com/furysport/fury-chain/x/airdrop/types"
	minttypes "github.com/furysport/fury-chain/x/mint/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	icatypes "github.com/cosmos/ibc-go/v3/modules/apps/27-interchain-accounts/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	liquiditytypes "github.com/gravity-devs/liquidity/x/liquidity/types"
	"github.com/spf13/cobra"
	tmtypes "github.com/tendermint/tendermint/types"
)

// PrepareGenesisCmd returns prepare-genesis cobra Command.
func PrepareGenesisCmd(defaultNodeHome string, mbm module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "prepare-genesis [chain_id] [airdrop_file_path]",
		Short: "Prepare a genesis file with initial setup",
		Long: `Prepare a genesis file with initial setup.
Example:
	furyad prepare-genesis furya-1 cosmos_aidrop.csv crew3_airdrop.csv evmos_orbital_ape.csv
	- Check input genesis:
		file is at ~/.furyad/config/genesis.json
`,
		Args: cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			depCdc := clientCtx.Codec
			cdc := depCdc
			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			// read genesis file
			genFile := config.GenesisFile()
			appState, genDoc, err := genutiltypes.GenesisStateFromGenFile(genFile)
			if err != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}

			// get genesis params
			chainID := args[0]

			// run Prepare Genesis
			appState, genDoc, err = PrepareGenesis(clientCtx, appState, genDoc, chainID, args[1], args[2], args[3])
			if err != nil {
				return err
			}

			// validate genesis state
			if err = mbm.ValidateGenesis(cdc, clientCtx.TxConfig, appState); err != nil {
				return fmt.Errorf("error validating genesis file: %s", err.Error())
			}

			// save genesis
			appStateJSON, err := json.Marshal(appState)
			if err != nil {
				return fmt.Errorf("failed to marshal application genesis state: %w", err)
			}

			genDoc.AppState = appStateJSON
			err = genutil.ExportGenesisFile(genDoc, genFile)
			return err
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func parseCosmosAirdropAmount(path string) ([]airdroptypes.AirdropAllocation, sdk.Coin) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}

	totalAmount := sdk.ZeroInt()
	allocations := []airdroptypes.AirdropAllocation{}
	for _, line := range records[1:] {
		cosmosAddr, amountStr := line[0], line[1]
		amountDec := sdk.MustNewDecFromStr(amountStr)
		amountInt := amountDec.Mul(sdk.NewDec(1000_000)).TruncateInt()

		allocations = append(allocations, airdroptypes.AirdropAllocation{
			Chain:         "cosmos",
			Address:       cosmosAddr,
			Amount:        sdk.NewCoin(appparams.BaseCoinUnit, amountInt),
			ClaimedAmount: sdk.NewInt64Coin(appparams.BaseCoinUnit, 0),
		})
		totalAmount = totalAmount.Add(amountInt)
	}

	return allocations, sdk.NewCoin(appparams.BaseCoinUnit, totalAmount)
}

func parseEvmosOrbitalApeAirdropAmount(path string) ([]airdroptypes.AirdropAllocation, sdk.Coin) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}

	totalAmount := sdk.ZeroInt()
	allocations := []airdroptypes.AirdropAllocation{}
	for _, line := range records[1:] {
		evmAddr, amountStr := line[0], line[1]
		amountDec := sdk.MustNewDecFromStr(amountStr)
		amountInt := amountDec.Mul(sdk.NewDec(1000_000)).TruncateInt()

		allocations = append(allocations, airdroptypes.AirdropAllocation{
			Chain:         "evm",
			Address:       evmAddr,
			Amount:        sdk.NewCoin(appparams.BaseCoinUnit, amountInt),
			ClaimedAmount: sdk.NewInt64Coin(appparams.BaseCoinUnit, 0),
		})
		totalAmount = totalAmount.Add(amountInt)
	}

	return allocations, sdk.NewCoin(appparams.BaseCoinUnit, totalAmount)
}

func combineAirdropAllocations(allocations1, allocations2 []airdroptypes.AirdropAllocation) []types.AirdropAllocation {
	usedAllocation := map[string]bool{}
	allocationIndex := map[string]int{}

	for index, allo := range allocations1 {
		usedAllocation[allo.Address] = true
		allocationIndex[allo.Address] = index
	}

	allocations := allocations1
	for _, allo := range allocations2 {
		if usedAllocation[allo.Address] {
			allocations[allocationIndex[allo.Address]].Amount = allocations[allocationIndex[allo.Address]].Amount.Add(allo.Amount)
		} else {
			allocations = append(allocations, allo)
		}
	}
	return allocations
}

func PrepareGenesis(clientCtx client.Context, appState map[string]json.RawMessage, genDoc *tmtypes.GenesisDoc, chainID, cosmosAirdropPath, crew3AirdropPath, evmosOrbitalApePath string) (map[string]json.RawMessage, *tmtypes.GenesisDoc, error) {
	depCdc := clientCtx.Codec
	cdc := depCdc

	// chain params genesis
	genDoc.ChainID = chainID
	genDoc.GenesisTime = time.Unix(1664802000, 0) // Mon Oct 03 2022 13:00:00 GMT+0000
	genDoc.ConsensusParams = tmtypes.DefaultConsensusParams()
	genDoc.ConsensusParams.Block.MaxBytes = 21 * 1024 * 1024
	genDoc.ConsensusParams.Block.MaxGas = 300_000_000

	// mint module genesis
	mintGenState := minttypes.DefaultGenesisState()
	mintGenState.Params = minttypes.DefaultParams()
	mintGenState.Params.MintDenom = appparams.BaseCoinUnit
	mintGenState.Params.MintingRewardsDistributionStartBlock = 51840 // 3 days after launch - 86400s x 3 / 5s
	mintGenStateBz, err := cdc.MarshalJSON(mintGenState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal mint genesis state: %w", err)
	}
	appState[minttypes.ModuleName] = mintGenStateBz

	// airdrop module genesis
	airdropGenState := airdroptypes.DefaultGenesis()
	airdropGenState.Params = airdroptypes.DefaultParams()
	airdropGenState.Params.Owner = "furya16w6chfrrg930cqcfewdzse6szgjk657764dll7" // POP's address
	cosmosAllocations, totalCosmosAirdropAllocation := parseCosmosAirdropAmount(cosmosAirdropPath)
	crew3Allocations, totalCrew3AirdropAllocation := parseCosmosAirdropAmount(crew3AirdropPath)
	cosmosAllocations = combineAirdropAllocations(cosmosAllocations, crew3Allocations)
	totalCosmosAirdropAllocation = totalCosmosAirdropAllocation.Add(totalCrew3AirdropAllocation)
	evmosOrbitalApeAllocations, totalEvmosAirdropAllocataion := parseEvmosOrbitalApeAirdropAmount(evmosOrbitalApePath)
	allocations := append(cosmosAllocations, evmosOrbitalApeAllocations...)
	airdropGenState.Allocations = allocations
	airdropGenStateBz, err := cdc.MarshalJSON(airdropGenState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal airdrop genesis state: %w", err)
	}
	appState[airdroptypes.ModuleName] = airdropGenStateBz

	// bank module genesis
	bankGenState := banktypes.DefaultGenesisState()
	bankGenState.Params = banktypes.DefaultParams()

	bankGenState.Supply = sdk.NewCoins(sdk.NewInt64Coin(appparams.BaseCoinUnit, 200_000_000_000_000)) // 200M FURY

	airdropCoins := sdk.Coins{totalCosmosAirdropAllocation.Add(totalEvmosAirdropAllocataion)}
	communityPoolCoins := sdk.NewCoins(sdk.NewInt64Coin(appparams.BaseCoinUnit, 50_000_000_000_000)) // 50M FURY

	seenBalances := make(map[string]bool)

	moduleAccs := []string{
		authtypes.FeeCollectorName,
		distrtypes.ModuleName,
		icatypes.ModuleName,
		minttypes.ModuleName,
		stakingtypes.BondedPoolName,
		stakingtypes.NotBondedPoolName,
		govtypes.ModuleName,
		liquiditytypes.ModuleName,
		ibctransfertypes.ModuleName,
		airdroptypes.ModuleName,
	}

	for _, module := range moduleAccs {
		moduleAddr := authtypes.NewModuleAddress(module)
		seenBalances[moduleAddr.String()] = true
	}

	// airdrop balance
	airdropModuleAddr := authtypes.NewModuleAddress(airdroptypes.ModuleName)
	bankGenState.Balances = append(bankGenState.Balances, banktypes.Balance{
		Address: airdropModuleAddr.String(),
		Coins:   airdropCoins,
	})

	// distribution balances for community pool
	distrModuleAddr := authtypes.NewModuleAddress(distributiontypes.ModuleName)
	bankGenState.Balances = append(bankGenState.Balances, banktypes.Balance{
		Address: distrModuleAddr.String(),
		Coins:   communityPoolCoins,
	})

	genAccounts := []authtypes.GenesisAccount{}

	addrStrategicReserve, err := sdk.AccAddressFromBech32("furya16w6chfrrg930cqcfewdzse6szgjk657764dll7")
	if err != nil {
		return nil, nil, err
	}
	genAccounts = append(genAccounts, authtypes.NewBaseAccount(addrStrategicReserve, nil, 0, 0))

	// send 10 FURY to genesis validators
	genesisValidators := []string{}

	totalValidatorInitialCoins := sdk.NewCoins()
	validatorInitialCoins := sdk.NewCoins(sdk.NewInt64Coin(appparams.BaseCoinUnit, 10_000_000)) // 10 FURY
	for _, address := range genesisValidators {
		if seenBalances[address] {
			continue
		}

		bankGenState.Balances = append(bankGenState.Balances, banktypes.Balance{
			Address: address,
			Coins:   validatorInitialCoins, // 0.1 FURY
		})
		addr, err := sdk.AccAddressFromBech32(address)
		if err != nil {
			return nil, nil, err
		}
		totalValidatorInitialCoins = totalValidatorInitialCoins.Add(validatorInitialCoins...)
		genAccounts = append(genAccounts, authtypes.NewBaseAccount(addr, nil, 0, 0))

		seenBalances[address] = true
	}

	// send 0.1 FURY to bech32 converted cosmos airdrop addresses
	totalAirdropGasCoins := sdk.NewCoins()
	airdropGasCoins := sdk.NewCoins(sdk.NewInt64Coin(appparams.BaseCoinUnit, 100_000))
	for _, allocation := range cosmosAllocations {
		_, bz, err := bech32.DecodeAndConvert(allocation.Address)
		if err != nil {
			return nil, nil, err
		}

		bech32Addr, err := bech32.ConvertAndEncode(appparams.Bech32PrefixAccAddr, bz)
		if err != nil {
			return nil, nil, err
		}

		if seenBalances[bech32Addr] {
			continue
		}

		bankGenState.Balances = append(bankGenState.Balances, banktypes.Balance{
			Address: bech32Addr,
			Coins:   airdropGasCoins, // 0.1 FURY
		})
		totalAirdropGasCoins = totalAirdropGasCoins.Add(airdropGasCoins...)
		genAccounts = append(genAccounts, authtypes.NewBaseAccount(sdk.AccAddress(bz), nil, 0, 0))

		seenBalances[bech32Addr] = true
	}

	// strategic reserve = 200M - 50M - airdropCoins
	bankGenState.Balances = append(bankGenState.Balances, banktypes.Balance{
		Address: addrStrategicReserve.String(),
		Coins:   bankGenState.Supply.Sub(airdropCoins).Sub(communityPoolCoins).Sub(totalAirdropGasCoins).Sub(totalValidatorInitialCoins),
	})

	bankGenStateBz, err := cdc.MarshalJSON(bankGenState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal bank genesis state: %w", err)
	}
	appState[banktypes.ModuleName] = bankGenStateBz

	// account module genesis
	authGenState := authtypes.GetGenesisStateFromAppState(depCdc, appState)
	authGenState.Params = authtypes.DefaultParams()

	accounts, err := authtypes.PackAccounts(genAccounts)
	if err != nil {
		panic(err)
	}

	authGenState.Accounts = append(authGenState.Accounts, accounts...)
	authGenStateBz, err := cdc.MarshalJSON(&authGenState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal staking genesis state: %w", err)
	}
	appState[authtypes.ModuleName] = authGenStateBz

	// staking module genesis
	stakingGenState := stakingtypes.GetGenesisStateFromAppState(depCdc, appState)
	stakingGenState.Params = stakingtypes.DefaultParams()
	stakingGenState.Params.BondDenom = appparams.BaseCoinUnit
	stakingGenStateBz, err := cdc.MarshalJSON(stakingGenState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal staking genesis state: %w", err)
	}
	appState[stakingtypes.ModuleName] = stakingGenStateBz

	// distribution module genesis
	distributionGenState := distributiontypes.DefaultGenesisState()
	distributionGenState.Params = distributiontypes.DefaultParams()
	distributionGenState.Params.BaseProposerReward = sdk.ZeroDec()
	distributionGenState.Params.BonusProposerReward = sdk.ZeroDec()
	distributionGenState.FeePool.CommunityPool = sdk.NewDecCoinsFromCoins(communityPoolCoins...)
	distributionGenStateBz, err := cdc.MarshalJSON(distributionGenState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal distribution genesis state: %w", err)
	}
	appState[distributiontypes.ModuleName] = distributionGenStateBz

	// gov module genesis
	govGenState := govtypes.DefaultGenesisState()
	defaultGovParams := govtypes.DefaultParams()
	govGenState.DepositParams = defaultGovParams.DepositParams
	govGenState.DepositParams.MinDeposit = sdk.Coins{sdk.NewInt64Coin(appparams.BaseCoinUnit, 500_000_000)} // 500 FURY
	govGenState.TallyParams = defaultGovParams.TallyParams
	govGenState.VotingParams = defaultGovParams.VotingParams
	govGenState.VotingParams.VotingPeriod = time.Hour * 24 * 2 // 2 days
	govGenStateBz, err := cdc.MarshalJSON(govGenState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal gov genesis state: %w", err)
	}
	appState[govtypes.ModuleName] = govGenStateBz

	// slashing module genesis
	slashingGenState := slashingtypes.DefaultGenesisState()
	slashingGenState.Params = slashingtypes.DefaultParams()
	slashingGenState.Params.SignedBlocksWindow = 10000
	slashingGenStateBz, err := cdc.MarshalJSON(slashingGenState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal slashing genesis state: %w", err)
	}
	appState[slashingtypes.ModuleName] = slashingGenStateBz

	// crisis module genesis
	crisisGenState := crisistypes.DefaultGenesisState()
	crisisGenState.ConstantFee = sdk.NewInt64Coin(appparams.BaseCoinUnit, 1000_000) // 1 FURY
	crisisGenStateBz, err := cdc.MarshalJSON(crisisGenState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal crisis genesis state: %w", err)
	}
	appState[crisistypes.ModuleName] = crisisGenStateBz

	// ica module genesis
	icaGenState := icatypes.DefaultGenesis()
	icaGenState.HostGenesisState.Params.AllowMessages = []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.bank.v1beta1.MsgMultiSend",
		"/cosmos.distribution.v1beta1.MsgSetWithdrawAddress",
		"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission",
		"/cosmos.distribution.v1beta1.MsgFundCommunityPool",
		"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward",
		"/cosmos.feegrant.v1beta1.MsgGrantAllowance",
		"/cosmos.feegrant.v1beta1.MsgRevokeAllowance",
		"/cosmos.gov.v1beta1.MsgVoteWeighted",
		"/cosmos.gov.v1beta1.MsgSubmitProposal",
		"/cosmos.gov.v1beta1.MsgDeposit",
		"/cosmos.gov.v1beta1.MsgVote",
		"/cosmos.staking.v1beta1.MsgEditValidator",
		"/cosmos.staking.v1beta1.MsgDelegate",
		"/cosmos.staking.v1beta1.MsgUndelegate",
		"/cosmos.staking.v1beta1.MsgBeginRedelegate",
		"/cosmos.staking.v1beta1.MsgCreateValidator",
		"/ibc.applications.transfer.v1.MsgTransfer",
	}
	icaGenStateBz, err := cdc.MarshalJSON(icaGenState)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal crisis genesis state: %w", err)
	}
	appState[icatypes.ModuleName] = icaGenStateBz

	// return appState and genDoc
	return appState, genDoc, nil
}
