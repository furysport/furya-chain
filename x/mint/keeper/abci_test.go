package keeper_test

import (
	"github.com/furysport/fury-chain/x/mint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/crypto/ed25519"
)

func (suite *KeeperTestSuite) TestEndBlocker() {
	grantsAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())
	usageIncentiveAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())
	dev1Addr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())
	dev2Addr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())
	teamReserveAddr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())

	defaultParams := suite.app.MintKeeper.GetParams(suite.ctx)
	params := types.Params{
		MintDenom:               "ufury",
		GenesisBlockProvisions:  defaultParams.GenesisBlockProvisions,
		ReductionPeriodInBlocks: 4000,
		ReductionFactor:         sdk.NewDecWithPrec(5, 1),
		DistributionProportions: types.DistributionProportions{
			GrantsProgram:    sdk.NewDecWithPrec(2, 1),
			CommunityPool:    sdk.NewDecWithPrec(2, 1),
			UsageIncentive:   sdk.NewDecWithPrec(2, 1),
			Staking:          sdk.NewDecWithPrec(2, 1),
			DeveloperRewards: sdk.NewDecWithPrec(2, 1),
		},
		WeightedDeveloperRewardsReceivers: []types.MonthlyVestingAddress{
			{
				Address:        dev1Addr.String(),
				MonthlyAmounts: []sdk.Int{sdk.NewInt(6000), sdk.NewInt(6000), sdk.NewInt(6000)},
			},
			{
				Address:        dev2Addr.String(),
				MonthlyAmounts: []sdk.Int{sdk.NewInt(4000), sdk.NewInt(4000), sdk.NewInt(4000)},
			},
		},
		UsageIncentiveAddress:                usageIncentiveAddr.String(),
		GrantsProgramAddress:                 grantsAddr.String(),
		TeamReserveAddress:                   teamReserveAddr.String(),
		MintingRewardsDistributionStartBlock: 10,
	}

	suite.SetupTest()
	suite.app.MintKeeper.SetParams(suite.ctx, params)

	newMonthInfo := types.TeamVestingMonthInfo{
		MonthsSinceGenesis:     1,
		MonthStartedBlock:      1,
		OneMonthPeriodInBlocks: 4000,
	}
	suite.app.MintKeeper.SetTeamVestingMonthInfo(suite.ctx, newMonthInfo)

	// check minter information
	minter := suite.app.MintKeeper.GetMinter(suite.ctx)
	suite.Require().Equal(minter.BlockProvisions, defaultParams.GenesisBlockProvisions)

	suite.app.MintKeeper.EndBlocker(suite.ctx)

	// check grants amount is not distributed
	grantsAddrBalance := suite.app.BankKeeper.GetBalance(suite.ctx, grantsAddr, params.MintDenom)
	suite.Require().Equal(grantsAddrBalance, sdk.NewInt64Coin(params.MintDenom, 0))

	// check last reduction block number is kept as 0
	lastReductionBlockNum := suite.app.MintKeeper.GetLastReductionBlockNum(suite.ctx)
	suite.Require().Equal(lastReductionBlockNum, int64(0))

	// check minter information
	minter = suite.app.MintKeeper.GetMinter(suite.ctx)
	suite.Require().Equal(minter.BlockProvisions, defaultParams.GenesisBlockProvisions)

	// check month info update
	monthInfo := suite.app.MintKeeper.GetTeamVestingMonthInfo(suite.ctx)
	suite.Require().Equal(monthInfo.MonthStartedBlock, int64(1))
	suite.Require().Equal(monthInfo.MonthsSinceGenesis, int64(1))

	suite.ctx = suite.ctx.WithBlockHeight(10)
	suite.app.MintKeeper.EndBlocker(suite.ctx)

	// check last reduction block number is updated to 10
	lastReductionBlockNum = suite.app.MintKeeper.GetLastReductionBlockNum(suite.ctx)
	suite.Require().Equal(lastReductionBlockNum, int64(10))

	// check grants amount is not distributed
	grantsAddrBalance = suite.app.BankKeeper.GetBalance(suite.ctx, grantsAddr, params.MintDenom)
	suite.Require().Equal(grantsAddrBalance, sdk.NewInt64Coin(params.MintDenom, 9400000))

	// check minter information
	minter = suite.app.MintKeeper.GetMinter(suite.ctx)
	suite.Require().Equal(minter.BlockProvisions, defaultParams.GenesisBlockProvisions)

	// check month info update
	monthInfo = suite.app.MintKeeper.GetTeamVestingMonthInfo(suite.ctx)
	suite.Require().Equal(monthInfo.MonthStartedBlock, int64(1))
	suite.Require().Equal(monthInfo.MonthsSinceGenesis, int64(1))

	suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + params.ReductionPeriodInBlocks)
	suite.app.MintKeeper.EndBlocker(suite.ctx)

	// check last reduction block number is updated to 10
	lastReductionBlockNum = suite.app.MintKeeper.GetLastReductionBlockNum(suite.ctx)
	suite.Require().Equal(lastReductionBlockNum, int64(10+params.ReductionPeriodInBlocks))

	// check grants amount is not distributed
	grantsAddrBalance = suite.app.BankKeeper.GetBalance(suite.ctx, grantsAddr, params.MintDenom)
	suite.Require().Equal(grantsAddrBalance, sdk.NewInt64Coin(params.MintDenom, 14100000))

	// check minter information
	minter = suite.app.MintKeeper.GetMinter(suite.ctx)
	suite.Require().Equal(minter.BlockProvisions, sdk.NewDec(23500000)) // 23.5

	// check month info update
	monthInfo = suite.app.MintKeeper.GetTeamVestingMonthInfo(suite.ctx)
	suite.Require().Equal(monthInfo.MonthStartedBlock, int64(10+params.ReductionPeriodInBlocks))
	suite.Require().Equal(monthInfo.MonthsSinceGenesis, int64(2))
}

func (suite *KeeperTestSuite) TestEndBlocker90MonthsCheckWithDefaultGenesis() {
	monthInfo := suite.app.MintKeeper.GetTeamVestingMonthInfo(suite.ctx)
	monthInfo.OneMonthPeriodInBlocks = 10
	suite.app.MintKeeper.SetTeamVestingMonthInfo(suite.ctx, monthInfo)

	params := suite.app.MintKeeper.GetParams(suite.ctx)
	params.MintingRewardsDistributionStartBlock = 10
	dev1Addr, _ := sdk.AccAddressFromBech32(params.WeightedDeveloperRewardsReceivers[0].Address)
	dev2Addr, _ := sdk.AccAddressFromBech32(params.WeightedDeveloperRewardsReceivers[1].Address)
	suite.app.MintKeeper.SetParams(suite.ctx, params)

	minterInfo := suite.app.MintKeeper.GetMinter(suite.ctx)
	minterInfo.BlockProvisions = minterInfo.BlockProvisions.Mul(sdk.NewDec(525600 / monthInfo.OneMonthPeriodInBlocks))
	suite.app.MintKeeper.SetMinter(suite.ctx, minterInfo)

	for i := 0; i < 90; i++ {
		for j := 0; j < int(monthInfo.OneMonthPeriodInBlocks); j++ {
			suite.NotPanics(func() {
				suite.app.MintKeeper.EndBlocker(suite.ctx)
				suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 1)
			})
		}
	}

	dev1Balance := suite.app.BankKeeper.GetBalance(suite.ctx, dev1Addr, params.MintDenom)
	dev2Balance := suite.app.BankKeeper.GetBalance(suite.ctx, dev2Addr, params.MintDenom)
	suite.Require().Equal(dev1Balance.String(), sdk.NewCoin(params.MintDenom, sdk.NewInt(3000003280000)).String())
	suite.Require().Equal(dev2Balance.String(), sdk.NewCoin(params.MintDenom, sdk.NewInt(6000006530000)).String())
}

func (suite *KeeperTestSuite) TestEndBlocker90MonthsCheck() {
	monthInfo := suite.app.MintKeeper.GetTeamVestingMonthInfo(suite.ctx)
	monthInfo.OneMonthPeriodInBlocks = 1000
	suite.app.MintKeeper.SetTeamVestingMonthInfo(suite.ctx, monthInfo)

	params := suite.app.MintKeeper.GetParams(suite.ctx)
	dev1Addr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())
	dev2Addr := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address().Bytes())
	params.WeightedDeveloperRewardsReceivers = []types.MonthlyVestingAddress{
		{
			Address:        dev1Addr.String(),
			MonthlyAmounts: []sdk.Int{sdk.NewInt(6000), sdk.NewInt(6000), sdk.NewInt(6000)},
		},
		{
			Address:        dev2Addr.String(),
			MonthlyAmounts: []sdk.Int{sdk.NewInt(4000), sdk.NewInt(4000), sdk.NewInt(4000)},
		},
	}
	suite.app.MintKeeper.SetParams(suite.ctx, params)

	for i := 0; i < 90; i++ {
		for j := 0; j < int(monthInfo.OneMonthPeriodInBlocks); j++ {
			suite.NotPanics(func() {
				suite.app.MintKeeper.EndBlocker(suite.ctx)
				suite.ctx = suite.ctx.WithBlockHeight(suite.ctx.BlockHeight() + 1)
			})
		}
	}

	dev1Balance := suite.app.BankKeeper.GetBalance(suite.ctx, dev1Addr, params.MintDenom)
	dev2Balance := suite.app.BankKeeper.GetBalance(suite.ctx, dev2Addr, params.MintDenom)
	suite.Require().Equal(dev1Balance, sdk.NewCoin(params.MintDenom, sdk.NewInt(18000)))
	suite.Require().Equal(dev2Balance, sdk.NewCoin(params.MintDenom, sdk.NewInt(12000)))
}
