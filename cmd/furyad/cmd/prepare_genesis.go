package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	appparams "github.com/FURYA/furya-chain/app/params"
	"github.com/FURYA/furya-chain/x/airdrop/types"
	airdroptypes "github.com/FURYA/furya-chain/x/airdrop/types"
	minttypes "github.com/FURYA/furya-chain/x/mint/types"
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
	airdropGenState.Params.Owner = "furya19ftk3lkfupgtnh38d7enc8c6jp7aljj3l66vae" // POP's address
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

	addrStrategicReserve, err := sdk.AccAddressFromBech32("furya1kcuty7d5mc0rasw6mpmn4nhk99me55ch57puyn")
	if err != nil {
		return nil, nil, err
	}
	genAccounts = append(genAccounts, authtypes.NewBaseAccount(addrStrategicReserve, nil, 0, 0))

	// send 10 FURY to genesis validators
	genesisValidators := []string{
		"furya1uechzauku6mhj2je8jmyrkq6d0ydm3g4d5jsdd", // metahuahua
		"furya1t7cyvydpp4lklprksnrjy2y3xzv3q2l075vlz3", // activenodes
		"furya1utr8j9685hfxyza3wnu8pa9lpu8360knf6rnu6", // alxvoy
		"furya17vx59a9897ltpyw6dwr7jvcjk8wyxhc023q5vs", // aurie
		"furya19pg5t5adjese84q5azjuv46rtz7jt2yqwz9rh2", // berty
		"furya16czx9ukfcelsxmjpyt90fprjx5qjw5an6q0pz5", // chillvalidation
		"furya1c22uwrtvadcp2a8rjn2l00kmuuqdcu2tup859x", // crosnest
		"furya1jfw63tylcc4gscayv68prsu275q6te4wkqzy0u", // dibugnodes
		"furya1a7taydvzhkd5vrndlykqtj7nsk2erdp24wkquq", // ericet
		"furya138664l4407d7hfwe8a82q25fk4vht53jdhs392", // forbole
		"furya15fh88a6kx6n6cgx35cxf2edzyjwq3rwyd85ya8", // freemint
		"furya1dyduggaqthztgm8tnk59flkeu3l3qvpzlyz6tu", // gatadao
		"furya15n624eajd04jjhnlvza2fvft3lmf69aeqltmrm", // interblockchainservices
		"furya1gspqsgxm8s4e9uza78met67x2eted2cd4gvx82", // goldenratiostaking
		"furya1534tslwra4hrvt8k8tdwh5aghmc74hvtjl2ka6", // gopher
		"furya1ttgzvn4lwkqe33drcvjxrefu8j9u5x9qpgx4z6", // hashquark
		"furya16dzaxgnq9zlac7yl3ar3zp4y2zgr9fm0aamdv2", // highstakes
		"furya1lgy98shrs4uyrqnmgh38su3gm08uh3sr63hqvt", // ibrahimarslan
		"furya1xpyql3vw67h8l99n3sswy5ev94ntwt9c5x9ju8", // icosmosdao
		"furya1shtyw4f5pdhvx7gsrsknwrryy9ulqvvylgnnys", // kalianetwork
		"furya184ln03hkpt75uhrrr26f66kvcqvf4yn4mt9xw5", // kjnodes
		"furya1rly8ah6hffkt28hy3ka8ue2h32mqknyxmuc5x5", // landeros
		"furya140l6y2gp3gxvay6qtn70re7z2s0gn57zrvgjnn", // lavenderfive
		"furya1kunzrdg6u8gql4faj33lstghhqdtp59e848g5t", // lesnik_utsa
		"furya1e6ajryqxefpxuhjg2y9wk4y2dzq48uz49dsmmz", // maethstro
		"furya18wjuryzyuwpg5f0wukgjey3za28s4fm9ka7md4", // munris
		"furya1p5z27dj7zrxue8pe5t0m39q8mmgavdclrc0lwv", // n0ok
		"furya18t2j2kc08su2l2dafcanq43yxj9akpwpjmkmlq", // nodejumper
		"furya1nrgahzmlr4nrnumlu0ud99qslsdvay8a6469k9", // noderunners
		"furya1phzay7cf4ayk9dsvt0q5nlc8qehlwlpxasd66u", // nodesblocks
		"furya1w3wse8cx2al5947ke0hnd2tgphjt43dydmpnnx", // nodesguru
		"furya16mzm5w3ys2va5mv00g0qnafnev4erc5knuysp8", // nxtpop
		"furya1nuh2h60wlvzvk58xll3d8gz2wpqjt6gw22y0kd", // nysa_network
		"furya18hgz56rlcpvc2y6l97n0gz248nmy86h3jayfrh", // onblocnode
		"furya1azdfljp04ptlazs95e5gscweavmaszw52gz794", // oni
		"furya18je2ph09a7flemkkzmvenz2eeyw5pdgegsye2f", // orbitalapes
		"furya1gp957czryfgyvxwn3tfnyy2f0t9g2p4p2ckclx", // polkachu
		"furya1fyyl63zqylda0qrkqdzeyag28eyh9swrmaur3j", // rhino
		"furya1qy38xmcrnht0kt5c5fryvl8llrpdwer64cfjgn", // romanv
		"furya1267l9z6yeua438mct5ee2mnm53yn3n9wjpqj3u", // samourai-world
		"furya167xwmhtrn7n8ftexu6luhh4luvhpy4357a2y4l", // silknodes
		"furya1xu736l4vt6l2pg9k2yk66fq7zq6y4aj5wgz6s0", // stakelab
		"furya1cm3hmw63a9wugawf9jn2jv0savynkgu9wufhxe", // stakingcabin
		"furya1sqk72uwf6tg867ssuu7whxfu9pfcyrpegaj9kh", // stavr
		"furya1lrq8sl2jq7246yjplutv5lul8ykrhqcrgg0dvn", // stingray
		"furya1gtz5v838vf7ucnn0jnqr3crs5099g9p2yjdxtr", // furya-core-1
		"furya1vxmq5epj83z8en5h0zul624nrmfxzmhkk0hy9a", // furya-core-2
		"furya1x6vfjy754fvzrlug2kxsp6s54yfj753s6cv7nx", // web34ever
		"furya1dfnzup7nppxvlpwnmzjnuet0tn4t9cnw6p2tmj", // wetez
		"furya1tjh6wpj6d9kpkfrcyglksevkhhtk9gm7sa3e2n", // whispernode
	}

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
