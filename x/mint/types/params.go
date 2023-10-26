package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
) Params {
	return Params{
		MintDenom:                            mintDenom,
		GenesisBlockProvisions:               genesisBlockProvisions,
		ReductionPeriodInBlocks:              reductionPeriodInBlocks,
		ReductionFactor:                      ReductionFactor,
		DistributionProportions:              distrProportions,
		WeightedDeveloperRewardsReceivers:    weightedDevRewardsReceivers,
		MintingRewardsDistributionStartBlock: MintingRewardsDistributionStartBlock,
	}
}

func addressTable() map[string]string {
	addressJSON := `{
		"adress1": "furya1g5dgnawwefpfkjmkyv7f6ga56wzp2w7l9vzpec",
		"adress2": "furya1e23suxkwrv5zfxt99js43lzmspsy9jfaa2ahfg",
		"adress3": "furya1yf98k5mv8g846d3m9j6t6rygfs6q7zwlp9awek",
		"adress4": "furya173wpa4cpv45klhjv7fjczytcv95eyuttamh0ma",
		"adress5": "furya1628wh06darjv9qutw7c2xlx4drr2nv9an03dk7",
		"adress6": "furya1z03prz76yeg4eyk5gnuwyfhd0dgmeafywv5h0x",
		"adress7": "furya1rwq208ld9wfrw389gjjh7vhmj8d8wpwexklcn0",
		"adress8": "furya1a9jqsecfyxuflrm0d4yl06hhn7p6du0al8rq7f",
		"adress9": "furya18crcv989365qxea04s4q70nl2cz30gewntj287",
		"adress10": "furya1twt8vxqayzgwx2fdfv49gzqfnuly9l6jzfx0ff",
		"adress11": "furya1hfetdttle00juc2wlvhgj2vsyvt7n85q5yc759",
		"adress12": "furya1gr2l7xyeuc33ztqlvyvkcsfysxez8vmcjmlatu",
		"adress13": "furya12y0elhp2hkxgyy5hrarrl8g7qpuyay22tdxsak",
		"adress14": "furya1aztlnw7fr246ya5p069pyl3udr033xc0l4kc0d",
		"adress15": "furya1s34lwjlxssjqw3zu3nyfzc5xk3zks7w0qvre4n",
		"adress16": "furya137relphljq46zqqkys3dlsrl5jtqjyu5lg62ec",
		"adress17": "furya1skyuxjps7gl3zc55r4e9juk7vn7zgcwz2y9sm2",
		"adress18": "furya19fwzytx3uuzyxelknkmwuktspk6p65l45sswe4",
		"adress19": "furya1dxq5usy2r9u9vsu8h55sx88ka70zua4p0gw2yt",
		"adress20": "furya1shegth9agcakr8q9jp2ckkejuyllhs0jdlhv0x",
		"adress21": "furya1np62mw4zuaej94s0y3vx0uyncwj039rtz5lxah",
		"adress22": "furya16j8l86vjkgthd0xake65vr3z6zg6grmnrqt0wc",
		"adress23": "furya17a3r0wdsgyl6kvl4wgj5ry4lqaqea2vefwvnnf",
		"adress24": "furya1aft7w2rc47kw0alcamwmth2ry9p7mw4gqs5n8z",
	}`

	var addressMap map[string]string
	json.Unmarshal([]byte(addressJSON), &addressMap)
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
			MonthlyAmounts: []sdk.Int{},
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
		UsageIncentiveAddress:                "furya1sf0vruy94hpdvcagcx99myhjp8xujevsgrpprg",
		GrantsProgramAddress:                 "furya1r0sxwusya7hkj6mlgspyurpvdzucjvclxgv3dw",
		TeamReserveAddress:                   "furya1eztq5wejayp84r9vegmm5p2jf4cc2e59dvha8c",
		MintingRewardsDistributionStartBlock: 0,
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
