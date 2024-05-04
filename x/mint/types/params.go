package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"cosmossdk.io/math"
	yaml "gopkg.in/yaml.v2"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys.
var (
	KeyMintDenom                            = []byte("MintDenom")
	KeyGenesisBlockProvisions               = []byte("GenesisBlockProvisions")
	KeyReductionPeriodInBlocks              = []byte("ReductionPeriodInBlocks")
	KeyReductionFactor                      = []byte("ReductionFactor")
	KeyPoolAllocationRatio                  = []byte("PoolAllocationRatio")
	KeyDeveloperRewardsReceiver             = []byte("DeveloperRewardsReceiver")
	KeyMintingRewardsDistributionStartBlock = []byte("MintingRewardsDistributionStartBlock")
	KeyUsageIncentiveAddress                = []byte("UsageIncentiveAddress")
	KeyGrantsProgramAddress                 = []byte("GrantsProgramAddress")
	KeyTeamReserveAddress                   = []byte("TeamReserveAddress")
	KeyBlocksPerYear                        = []byte("BlocksPerYear")
	KeyTotalBurntAmount                     = []byte("TotalBurntAmount")
)

// ParamTable for minting module.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams returns new mint module parameters initialized to the given values.
func NewParams(
	mintDenom string, genesisBlockProvisions sdk.Dec,
	ReductionFactor sdk.Dec, reductionPeriodInBlocks int64, distrProportions DistributionProportions,
	weightedDevRewardsReceivers []MonthlyVestingAddress, MintingRewardsDistributionStartBlock int64,
	blocksPerYear uint64, totalBurntAmount sdk.Coins,
) Params {
	return Params{
		MintDenom:                            mintDenom,
		GenesisBlockProvisions:               genesisBlockProvisions,
		ReductionPeriodInBlocks:              reductionPeriodInBlocks,
		ReductionFactor:                      ReductionFactor,
		DistributionProportions:              distrProportions,
		WeightedDeveloperRewardsReceivers:    weightedDevRewardsReceivers,
		MintingRewardsDistributionStartBlock: MintingRewardsDistributionStartBlock,
		BlocksPerYear:                        blocksPerYear,
		TotalBurntAmount:                     totalBurntAmount,
	}
}

func addressTable() map[string]string {
	addressJSON := `{
		"adress1": "furya13ehhh4f854wv5fatdp42qqa8k7n3seknkcjnp0",
		"adress2": "furya1usxflj986y7nmw32dgwmsssd32v845zqdknal7",
		"adress3": "furya1wlmqkpz0hkvevu3gghyxudn42gpnu4zgx2w7wt",
		"adress4": "furya1slye0cr6l3czwrdmpflq7ucd42a8egtclcrmhe",
		"adress5": "furya133dalktxc8uj94q07hjekrpucpl49fg783q628",
		"adress6": "furya1670dvuv348eynr9lsmdrhqu3g7vpmzx99qr6dp",
		"adress7": "furya1kzhxmlddc76zmhm48yejqfz0d8uqvgxxyh3zmm",
		"adress8": "furya1efptgq00hngrlgz2gc66g53jslw5eqkn4kexd5",
		"adress9": "furya1kt4pap9rdcm72gcppvm0l2khnku98padg8nrte",
		"adress10": "furya197lfuc8s5amj4lhkmyv5dex7qsdu3m5jh40tp6",
		"adress11": "furya158ur3xdz4clc09tjtz7sgyv475cfmc4jkd4ra5",
		"adress12": "furya1atjp0q3fh0hnya0kpu0zfrm93g2l2jldtsxxsw",
		"adress13": "furya1xzern65ppact6sgh4htllnr8pg9dpnc8vzzv7t",
		"adress14": "furya1slh5gxhljapjrsj2d9nmsnpp9vn3a0fd7yalzz",
		"adress15": "furya1kdhac2qmk2y0zpfmh3hhquxqkw463tunjpgejt",
		"adress16": "furya1nrdcetypk86kr2lxs6ns3rvf2fcafmvqurdnzv",
		"adress17": "furya1xesqr8vjvy34jhu027zd70ypl0nnev5efgn0me",
		"adress18": "furya1fqspuscrlzrtpghqnmqmj8j9dqr9hdmzlatcu6",
		"adress19": "furya1yasjpwlug8355auvle465lp3k3y0yhuuqufxwp",
		"adress20": "furya1fdqgj600jv0ydjas89kgel2gt880j44jt0l7ls",
		"adress21": "furya1pg9t4kxgwhtsp72dgd42x42pxzswaze35fxxlq"
	}`

	var addressMap map[string]string
	err := json.Unmarshal([]byte(addressJSON), &addressMap)
	if err != nil {
		panic(err)
	}
	return addressMap
}

func parseMonthlyVesting() []MonthlyVestingAddress {
	records := [][]string{}
	lines := strings.Split(vestingStr, "\n")
	for _, line := range lines {
		records = append(records, strings.Split(line, ","))
	}

	addressMap := addressTable()
	vAddrs := []MonthlyVestingAddress{}
	for _, addr := range records[0] {
		vAddrs = append(vAddrs, MonthlyVestingAddress{
			Address:        addressMap[addr],
			MonthlyAmounts: []math.Int{},
		})
	}

	for _, line := range records[1:] {
		for index, amountStr := range line {
			amountDec := sdk.MustNewDecFromStr(amountStr)
			amountInt := amountDec.Mul(sdk.NewDec(1000_000)).TruncateInt()
			vAddrs[index].MonthlyAmounts = append(vAddrs[index].MonthlyAmounts, amountInt)
		}
	}

	return vAddrs
}

// DefaultParams returns the default minting module parameters.
func DefaultParams() Params {
	return Params{
		MintDenom:               sdk.DefaultBondDenom,
		GenesisBlockProvisions:  sdk.NewDec(47000000),        //  300 million /  6307200 * 10 ^ 6
		ReductionPeriodInBlocks: 6307200,                     // 1 year - 86400 x 365 / 5
		ReductionFactor:         sdk.NewDecWithPrec(6666, 4), // 0.6666
		DistributionProportions: DistributionProportions{
			GrantsProgram:    sdk.NewDecWithPrec(10, 2), // 10%
			CommunityPool:    sdk.NewDecWithPrec(10, 2), // 10%
			UsageIncentive:   sdk.NewDecWithPrec(25, 2), // 25%
			Staking:          sdk.NewDecWithPrec(40, 2), // 40%
			DeveloperRewards: sdk.NewDecWithPrec(15, 2), // 15%
		},
		WeightedDeveloperRewardsReceivers:    parseMonthlyVesting(),
		UsageIncentiveAddress:                "furya1at6zkjpxleg8nd8u67542fprzgsev6jh5lfzne",
		GrantsProgramAddress:                 "furya1a28lq0usqrma2tn5t7vmdg3jnglh3v3qln4ky0",
		TeamReserveAddress:                   "furya1efcnw3j074urqryseyx4weahr2p5at9lhwcaju",
		MintingRewardsDistributionStartBlock: 0,
		BlocksPerYear:                        5733818,
		TotalBurntAmount:                     sdk.Coins{},
	}
}

// Validate validates mint module parameters. Returns nil if valid,
// error otherwise
func (p Params) Validate() error {
	if err := validateMintDenom(p.MintDenom); err != nil {
		return err
	}
	if err := validateGenesisBlockProvisions(p.GenesisBlockProvisions); err != nil {
		return err
	}
	if err := validateReductionPeriodInBlocks(p.ReductionPeriodInBlocks); err != nil {
		return err
	}
	if err := validateReductionFactor(p.ReductionFactor); err != nil {
		return err
	}
	if err := validateDistributionProportions(p.DistributionProportions); err != nil {
		return err
	}

	if err := validateAddress(p.UsageIncentiveAddress); err != nil {
		return err
	}

	if err := validateAddress(p.GrantsProgramAddress); err != nil {
		return err
	}

	if err := validateAddress(p.TeamReserveAddress); err != nil {
		return err
	}

	if err := validateWeightedDeveloperRewardsReceivers(p.WeightedDeveloperRewardsReceivers); err != nil {
		return err
	}
	if err := validateMintingRewardsDistributionStartBlock(p.MintingRewardsDistributionStartBlock); err != nil {
		return err
	}

	if err := validateBlocksPerYear(p.BlocksPerYear); err != nil {
		return err
	}

	if err := validateTotalBurntAmount(p.TotalBurntAmount); err != nil {
		return err
	}
	return nil
}

// String implements the Stringer interface.
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// Implements params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {

	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyMintDenom, &p.MintDenom, validateMintDenom),
		paramtypes.NewParamSetPair(KeyGenesisBlockProvisions, &p.GenesisBlockProvisions, validateGenesisBlockProvisions),
		paramtypes.NewParamSetPair(KeyReductionPeriodInBlocks, &p.ReductionPeriodInBlocks, validateReductionPeriodInBlocks),
		paramtypes.NewParamSetPair(KeyReductionFactor, &p.ReductionFactor, validateReductionFactor),
		paramtypes.NewParamSetPair(KeyPoolAllocationRatio, &p.DistributionProportions, validateDistributionProportions),
		paramtypes.NewParamSetPair(KeyDeveloperRewardsReceiver, &p.WeightedDeveloperRewardsReceivers, validateWeightedDeveloperRewardsReceivers),
		paramtypes.NewParamSetPair(KeyUsageIncentiveAddress, &p.UsageIncentiveAddress, validateAddress),
		paramtypes.NewParamSetPair(KeyGrantsProgramAddress, &p.GrantsProgramAddress, validateAddress),
		paramtypes.NewParamSetPair(KeyTeamReserveAddress, &p.TeamReserveAddress, validateAddress),
		paramtypes.NewParamSetPair(KeyMintingRewardsDistributionStartBlock, &p.MintingRewardsDistributionStartBlock, validateMintingRewardsDistributionStartBlock),
		paramtypes.NewParamSetPair(KeyBlocksPerYear, &p.BlocksPerYear, validateBlocksPerYear),
		paramtypes.NewParamSetPair(KeyTotalBurntAmount, &p.TotalBurntAmount, validateTotalBurntAmount),
	}
}

func validateMintDenom(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if strings.TrimSpace(v) == "" {
		return errors.New("mint denom cannot be blank")
	}
	if err := sdk.ValidateDenom(v); err != nil {
		return err
	}

	return nil
}

func validateGenesisBlockProvisions(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.LT(sdk.ZeroDec()) {
		return fmt.Errorf("genesis block provision must be non-negative")
	}

	return nil
}

func validateReductionPeriodInBlocks(i interface{}) error {
	v, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v <= 0 {
		return fmt.Errorf("reduction period must be positive: %d", v)
	}

	return nil
}

func validateReductionFactor(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.GT(sdk.NewDec(1)) {
		return fmt.Errorf("reduction factor cannot be greater than 1")
	}

	if v.IsNegative() {
		return fmt.Errorf("reduction factor cannot be negative")
	}

	return nil
}

func validateDistributionProportions(i interface{}) error {
	v, ok := i.(DistributionProportions)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.GrantsProgram.IsNegative() {
		return errors.New("staking distribution ratio should not be negative")
	}

	if v.CommunityPool.IsNegative() {
		return errors.New("staking distribution ratio should not be negative")
	}

	if v.UsageIncentive.IsNegative() {
		return errors.New("community pool distribution ratio should not be negative")
	}

	if v.Staking.IsNegative() {
		return errors.New("staking distribution ratio should not be negative")
	}

	if v.DeveloperRewards.IsNegative() {
		return errors.New("developer rewards distribution ratio should not be negative")
	}

	totalProportions := v.GrantsProgram.Add(v.CommunityPool).Add(v.UsageIncentive).Add(v.Staking).Add(v.DeveloperRewards)

	if !totalProportions.Equal(sdk.NewDec(1)) {
		return errors.New("total distributions ratio should be 1")
	}

	return nil
}

func validateWeightedDeveloperRewardsReceivers(i interface{}) error {
	v, ok := i.([]MonthlyVestingAddress)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	// fund community pool when rewards address is empty
	if len(v) == 0 {
		return nil
	}

	return nil
}

func validateMintingRewardsDistributionStartBlock(i interface{}) error {
	v, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v < 0 {
		return fmt.Errorf("start block must be non-negative")
	}

	return nil
}

func validateAddress(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	_, err := sdk.AccAddressFromBech32(v)

	return err
}

func validateBlocksPerYear(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v <= 0 {
		return fmt.Errorf("number of blocks per year should be positive")
	}

	return nil
}

func validateTotalBurntAmount(i interface{}) error {
	_, ok := i.([]sdk.Coin)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}
