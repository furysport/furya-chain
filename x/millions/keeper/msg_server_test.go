package keeper_test

import (
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gogotypes "github.com/cosmos/gogoproto/types"

	apptesting "github.com/furysport/furya-chain/v2/app/testing"
	epochstypes "github.com/furysport/furya-chain/v2/x/epochs/types"

	millionskeeper "github.com/furysport/furya-chain/v2/x/millions/keeper"
	millionstypes "github.com/furysport/furya-chain/v2/x/millions/types"
)

// TestMsgServer_GenerateSeed run seed generation related tests
func (suite *KeeperTestSuite) TestMsgServer_GenerateSeed() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx.WithBlockHeight(12).WithBlockTime(time.Now().UTC())
	msgServer := millionskeeper.NewMsgServerImpl(*app.MillionsKeeper)

	// Generate a basic seed
	response, err := msgServer.GenerateSeed(sdk.WrapSDKContext(ctx), &millionstypes.MsgGenerateSeed{})
	suite.Require().NoError(err)
	suite.Require().NotNil(response.GetSeed())

	// Asking for a second one should always return the same (at same height)
	secondResponse, err := msgServer.GenerateSeed(sdk.WrapSDKContext(ctx), &millionstypes.MsgGenerateSeed{})
	suite.Require().NoError(err)
	suite.Require().NotNil(secondResponse.GetSeed())
	suite.Require().Equal(response.GetSeed(), secondResponse.GetSeed())

	// Asking for a third one should be different (at different height)
	ctx = suite.ctx.WithBlockHeight(13).WithBlockTime(time.Now().UTC())
	thirdResponse, err := msgServer.GenerateSeed(sdk.WrapSDKContext(ctx), &millionstypes.MsgGenerateSeed{})
	suite.Require().NoError(err)
	suite.Require().NotNil(thirdResponse.GetSeed())
	suite.Require().NotEqual(thirdResponse.GetSeed(), response.GetSeed())
	suite.Require().NotEqual(thirdResponse.GetSeed(), secondResponse.GetSeed())
}

// TestMsgServer_DrawRetry runs draw retry related tests
func (suite *KeeperTestSuite) TestMsgServer_DrawRetry() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	goCtx := sdk.WrapSDKContext(ctx)
	msgServer := millionskeeper.NewMsgServerImpl(*app.MillionsKeeper)

	// Add 5 pools
	for i := 0; i < 5; i++ {
		poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
		drawDelta1 := 1 * time.Hour
		app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
			PoolId: poolID,
			PrizeStrategy: millionstypes.PrizeStrategy{
				PrizeBatches: []millionstypes.PrizeBatch{
					{PoolPercent: 100, Quantity: 1, DrawProbability: floatToDec(0.00)},
				},
			},
			DrawSchedule: millionstypes.DrawSchedule{
				InitialDrawAt: ctx.BlockTime().Add(drawDelta1),
				DrawDelta:     drawDelta1,
			},
			AvailablePrizePool: sdk.NewCoin(localPoolDenom, math.NewInt(1000)),
		}))
	}

	// Retrieve the pool from the state
	pools := app.MillionsKeeper.ListPools(ctx)

	// Create a new draw
	draw1 := millionstypes.Draw{
		PoolId:                 pools[0].PoolId,
		DrawId:                 uint64(10),
		State:                  millionstypes.DrawState_Failure,
		ErrorState:             millionstypes.DrawState_IcaWithdrawRewards,
		PrizePoolRemainsAmount: sdk.NewInt(1_000_000),
		PrizePool:              sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	}
	app.MillionsKeeper.SetPoolDraw(ctx, draw1)

	// Test GetPoolDraw
	draw, err := app.MillionsKeeper.GetPoolDraw(ctx, draw1.PoolId, draw1.DrawId)
	suite.Require().NoError(err)
	// - Test GetPoolDraw with wrong poolID
	_, err = app.MillionsKeeper.GetPoolDraw(ctx, uint64(0), draw.DrawId)
	suite.Require().ErrorIs(err, millionstypes.ErrPoolDrawNotFound)
	// - Test GetPoolDraw with wrong drawID
	_, err = app.MillionsKeeper.GetPoolDraw(ctx, draw.PoolId, uint64(0))
	suite.Require().ErrorIs(err, millionstypes.ErrPoolDrawNotFound)

	// Test ValidateDrawRetryBasic
	// - Test unknown poolID
	_, err = msgServer.DrawRetry(goCtx, &millionstypes.MsgDrawRetry{
		PoolId: uint64(0),
		DrawId: draw.DrawId,
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)
	// - Test unknown drawID
	_, err = msgServer.DrawRetry(goCtx, &millionstypes.MsgDrawRetry{
		PoolId: draw.PoolId,
		DrawId: uint64(0),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)

	// - Test unknown drawID
	_, err = msgServer.DrawRetry(goCtx, &millionstypes.MsgDrawRetry{
		PoolId: uint64(10),
		DrawId: draw.DrawId,
	})
	suite.Require().ErrorIs(err, millionstypes.ErrPoolNotFound)

	draw, err = app.MillionsKeeper.GetPoolDraw(ctx, draw.PoolId, draw.DrawId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.DrawState_Failure, draw.State)
	suite.Require().Equal(millionstypes.DrawState_IcaWithdrawRewards, draw.ErrorState)

	pools = app.MillionsKeeper.ListPools(ctx)

	// Create a new draw to test DrawState_IcaWithdrawRewards ErrorState
	draw2 := millionstypes.Draw{
		PoolId:                 pools[1].PoolId,
		DrawId:                 uint64(20),
		State:                  millionstypes.DrawState_Failure,
		ErrorState:             millionstypes.DrawState_IcaWithdrawRewards,
		PrizePoolRemainsAmount: sdk.NewInt(1_000_000),
		PrizePool:              sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	}
	app.MillionsKeeper.SetPoolDraw(ctx, draw2)

	draw2, err = app.MillionsKeeper.GetPoolDraw(ctx, draw2.PoolId, draw2.DrawId)
	suite.Require().NoError(err)

	_, err = msgServer.DrawRetry(goCtx, &millionstypes.MsgDrawRetry{
		PoolId: draw2.PoolId,
		DrawId: draw2.DrawId,
	})
	suite.Require().NoError(err)

	draw2, err = app.MillionsKeeper.GetPoolDraw(ctx, draw2.PoolId, draw2.DrawId)
	suite.Require().NoError(err)

	// Test that the msg retry is success
	suite.Require().Equal(millionstypes.DrawState_Success, draw2.State)
	suite.Require().Equal(millionstypes.DrawState_Unspecified, draw2.ErrorState)
	suite.Require().Equal(ctx.BlockHeight(), draw2.UpdatedAtHeight)
	suite.Require().Equal(ctx.BlockTime(), draw2.UpdatedAt)

	pools = app.MillionsKeeper.ListPools(ctx)

	// Create a new draw to test DrawState_IbcTransfer ErrorState
	draw3 := millionstypes.Draw{
		PoolId:                 pools[2].PoolId,
		DrawId:                 uint64(30),
		State:                  millionstypes.DrawState_Failure,
		ErrorState:             millionstypes.DrawState_IbcTransfer,
		PrizePoolRemainsAmount: sdk.NewInt(1_000_000),
		PrizePool:              sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	}
	app.MillionsKeeper.SetPoolDraw(ctx, draw3)

	draw3, err = app.MillionsKeeper.GetPoolDraw(ctx, draw3.PoolId, draw3.DrawId)
	suite.Require().NoError(err)

	_, err = msgServer.DrawRetry(goCtx, &millionstypes.MsgDrawRetry{
		PoolId: draw3.PoolId,
		DrawId: draw3.DrawId,
	})
	suite.Require().NoError(err)

	draw3, err = app.MillionsKeeper.GetPoolDraw(ctx, draw3.PoolId, draw3.DrawId)
	suite.Require().NoError(err)

	// Test that the msg retry is success
	suite.Require().Equal(millionstypes.DrawState_Success, draw3.State)
	suite.Require().Equal(millionstypes.DrawState_Unspecified, draw3.ErrorState)
	suite.Require().Equal(ctx.BlockHeight(), draw3.UpdatedAtHeight)
	suite.Require().Equal(ctx.BlockTime(), draw3.UpdatedAt)

	// Create a new draw to test DrawState_Drawing ErrorState
	draw4 := millionstypes.Draw{
		PoolId:                 pools[3].PoolId,
		DrawId:                 uint64(40),
		State:                  millionstypes.DrawState_Failure,
		ErrorState:             millionstypes.DrawState_Drawing,
		PrizePoolRemainsAmount: sdk.NewInt(1_000_000),
		PrizePool:              sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	}
	app.MillionsKeeper.SetPoolDraw(ctx, draw4)

	draw4, err = app.MillionsKeeper.GetPoolDraw(ctx, draw4.PoolId, draw4.DrawId)
	suite.Require().NoError(err)

	_, err = msgServer.DrawRetry(goCtx, &millionstypes.MsgDrawRetry{
		PoolId: draw4.PoolId,
		DrawId: draw4.DrawId,
	})
	suite.Require().NoError(err)

	draw4, err = app.MillionsKeeper.GetPoolDraw(ctx, draw4.PoolId, draw4.DrawId)
	suite.Require().NoError(err)

	suite.Require().Equal(millionstypes.DrawState_Success, draw4.State)
	suite.Require().Equal(millionstypes.DrawState_Unspecified, draw4.ErrorState)
	suite.Require().Equal(ctx.BlockHeight(), draw4.UpdatedAtHeight)
	suite.Require().Equal(ctx.BlockTime(), draw4.UpdatedAt)

	// Create a new draw to test DrawState_Unspecified ErrorState
	draw5 := millionstypes.Draw{
		PoolId:                 pools[4].PoolId,
		DrawId:                 uint64(50),
		State:                  millionstypes.DrawState_Failure,
		ErrorState:             millionstypes.DrawState_Unspecified,
		PrizePoolRemainsAmount: sdk.NewInt(1_000_000),
		PrizePool:              sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	}
	app.MillionsKeeper.SetPoolDraw(ctx, draw5)

	draw5, err = app.MillionsKeeper.GetPoolDraw(ctx, draw5.PoolId, draw5.DrawId)
	suite.Require().NoError(err)

	_, err = msgServer.DrawRetry(goCtx, &millionstypes.MsgDrawRetry{
		PoolId: draw5.PoolId,
		DrawId: draw5.DrawId,
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidDrawState)
}

// TestMsgServer_Credit runs credit related tests
func (suite *KeeperTestSuite) TestMsgServer_Credit() {
	app := suite.app
	ctx := suite.ctx
	goCtx := sdk.WrapSDKContext(ctx)
	msgServer := millionskeeper.NewMsgServerImpl(*app.MillionsKeeper)

	// Create pool
	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
	drawDelta1 := 1 * time.Hour
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId: poolID,
		PrizeStrategy: millionstypes.PrizeStrategy{
			PrizeBatches: []millionstypes.PrizeBatch{
				{PoolPercent: 100, Quantity: 1, DrawProbability: floatToDec(0.00)},
			},
		},
		DrawSchedule: millionstypes.DrawSchedule{
			InitialDrawAt: ctx.BlockTime().Add(drawDelta1),
			DrawDelta:     drawDelta1,
		},
		AvailablePrizePool: sdk.NewCoin(localPoolDenom, math.NewInt(1000)),
	}))

	pools := app.MillionsKeeper.ListPools(ctx)

	// Invalid pool ID
	_, err := msgServer.Credit(goCtx, &millionstypes.MsgCredit{
		PoolId:           uint64(0),
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, millionstypes.ErrPoolNotFound)

	// Invalid creditor address
	_, err = msgServer.Credit(goCtx, &millionstypes.MsgCredit{
		PoolId:           poolID,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(1_000_000))),
		CreditorAddress: "",
	})
	suite.Require().Error(err)
	suite.Require().Equal(err, millionstypes.ErrInvalidCreditorAddress)

	// Invalid credit denom
	_, err = msgServer.Credit(goCtx, &millionstypes.MsgCredit{
		PoolId:           poolID,
		Amount:           sdk.NewCoin("ufaketoken", sdk.NewInt(1_000_000)),
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidCreditDenom)

	// Invalid amount
	_, err = msgServer.Credit(goCtx, &millionstypes.MsgCredit{
		PoolId:           poolID,
		Amount:           sdk.NewCoin(localPoolDenom, pools[0].MinCreditAmount.Sub(sdk.NewInt(1))),
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, millionstypes.ErrInsufficientCreditAmount)

	// Invalid winner address
	_, err = msgServer.Credit(goCtx, &millionstypes.MsgCredit{
		PoolId:           poolID,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(1_000_000))),
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    "no-address",
	})
	suite.Require().Error(err)
	suite.Require().Equal(err, millionstypes.ErrInvalidWinnerAddress)

	_, err = msgServer.Credit(goCtx, &millionstypes.MsgCredit{
		PoolId:           poolID,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(123456))),
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
		IsSponsor:        true,
	})
	suite.Require().Error(err)
	suite.Require().Equal(err, millionstypes.ErrInvalidSponsorWinnerCombo)

	// Register real pool and apply real credits
	poolID, err = app.MillionsKeeper.RegisterPool(ctx,
		millionstypes.PoolType_Staking,
		"ufury",
		"ufury",
		testChainID,
		"",
		"",
		[]string{suite.valAddrs[0].String()},
		"furya",
		"furyavaloper",
		app.MillionsKeeper.GetParams(ctx).MinCreditAmount,
		time.Duration(millionstypes.DefaultUnbondingDuration),
		sdk.NewInt(millionstypes.DefaultMaxUnbondingEntries),
		millionstypes.DrawSchedule{DrawDelta: 24 * time.Hour, InitialDrawAt: ctx.BlockTime().Add(24 * time.Hour)},
		millionstypes.PrizeStrategy{PrizeBatches: []millionstypes.PrizeBatch{{PoolPercent: 100, Quantity: 100, DrawProbability: sdk.NewDec(1)}}},
		[]millionstypes.FeeTaker{
			{Destination: authtypes.FeeCollectorName, Amount: sdk.NewDecWithPrec(millionstypes.DefaultFeeTakerAmount, 2), Type: millionstypes.FeeTakerType_LocalModuleAccount},
		},
	)
	suite.Require().NoError(err)
	_, err = app.MillionsKeeper.GetPool(ctx, poolID)
	suite.Require().NoError(err)
	// Create pool and credits
	// Operate basic checks to verify if credit is created
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId: app.MillionsKeeper.GetNextPoolID(ctx),
		PrizeStrategy: millionstypes.PrizeStrategy{
			PrizeBatches: []millionstypes.PrizeBatch{
				{PoolPercent: 100, Quantity: 1, DrawProbability: floatToDec(0.00)},
			},
		},
		DrawSchedule: millionstypes.DrawSchedule{
			InitialDrawAt: ctx.BlockTime().Add(drawDelta1),
			DrawDelta:     drawDelta1,
		},
		AvailablePrizePool: sdk.NewCoin(localPoolDenom, math.NewInt(1000)),
	}))
	pools = app.MillionsKeeper.ListPools(ctx)

	pool, err := app.MillionsKeeper.GetPool(ctx, pools[1].PoolId)
	suite.Require().NoError(err)

	balanceBefore := app.BankKeeper.GetBalance(ctx, suite.addrs[0], localPoolDenom)
	_, err = msgServer.Credit(goCtx, &millionstypes.MsgCredit{
		PoolId:           pool.GetPoolId(),
		Amount:           sdk.NewCoin(pool.Denom, sdk.NewInt(int64(1_000_000))),
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().NoError(err)
	balanceNow := app.BankKeeper.GetBalance(ctx, suite.addrs[0], localPoolDenom)
	suite.Require().Equal(balanceBefore.Amount.Int64()-1_000_000, balanceNow.Amount.Int64())

	_, err = msgServer.Credit(goCtx, &millionstypes.MsgCredit{
		PoolId:           pool.GetPoolId(),
		Amount:           sdk.NewCoin(pool.Denom, sdk.NewInt(int64(2_000_000))),
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
	})
	suite.Require().NoError(err)
	balanceNow = app.BankKeeper.GetBalance(ctx, suite.addrs[0], localPoolDenom)
	suite.Require().Equal(balanceBefore.Amount.Int64()-3_000_000, balanceNow.Amount.Int64())

	balanceBefore = app.BankKeeper.GetBalance(ctx, suite.addrs[2], localPoolDenom)
	_, err = msgServer.Credit(goCtx, &millionstypes.MsgCredit{
		PoolId:           pool.GetPoolId(),
		Amount:           sdk.NewCoin(pool.Denom, sdk.NewInt(int64(5_000_000))),
		CreditorAddress: suite.addrs[2].String(),
		IsSponsor:        true,
	})
	suite.Require().NoError(err)
	balanceNow = app.BankKeeper.GetBalance(ctx, suite.addrs[2], localPoolDenom)
	suite.Require().Equal(balanceBefore.Amount.Int64()-5_000_000, balanceNow.Amount.Int64())

	// Compare pool tvl and creditors count state
	suite.Require().Equal(sdk.NewInt(0), pool.TvlAmount)
	suite.Require().Equal(uint64(0), pool.CreditorsCount)
	pool, err = app.MillionsKeeper.GetPool(ctx, pool.GetPoolId())
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewInt(8_000_000), pool.TvlAmount)
	suite.Require().Equal(uint64(2), pool.CreditorsCount)
	suite.Require().Equal(sdk.NewInt(int64(5_000_000)), pool.SponsorshipAmount)

	// Test credit for a pool that is not in ready state
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId: 4,
		PrizeStrategy: millionstypes.PrizeStrategy{
			PrizeBatches: []millionstypes.PrizeBatch{
				{PoolPercent: 100, Quantity: 1, DrawProbability: floatToDec(0.00)},
			},
		},
		DrawSchedule: millionstypes.DrawSchedule{
			InitialDrawAt: ctx.BlockTime().Add(drawDelta1),
			DrawDelta:     drawDelta1,
		},
		AvailablePrizePool: sdk.NewCoin(localPoolDenom, math.NewInt(1000)),
		State:              millionstypes.PoolState_Paused,
	}))

	pool, err = app.MillionsKeeper.GetPool(ctx, 4)
	suite.Require().NoError(err)

	_, err = msgServer.Credit(goCtx, &millionstypes.MsgCredit{
		PoolId:           pool.PoolId,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(1_000_000))),
		CreditorAddress: suite.addrs[0].String(),
	})
	// Credit should be allowed for paused pools
	suite.Require().NoError(err)
	pool, err = app.MillionsKeeper.GetPool(ctx, 4)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(1_000_000), pool.TvlAmount.Int64())
}

// TestMsgServer_CreditRetry tests the retry of a failed credit
func (suite *KeeperTestSuite) TestMsgServer_CreditRetry() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	goCtx := sdk.WrapSDKContext(ctx)
	msgServer := millionskeeper.NewMsgServerImpl(*app.MillionsKeeper)

	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)
	// Initialize the pool
	poolID, err := app.MillionsKeeper.RegisterPool(ctx,
		millionstypes.PoolType_Staking,
		denom,
		denom,
		testChainID,
		"",
		"",
		[]string{suite.valAddrs[0].String()},
		"furya",
		"furyavaloper",
		app.MillionsKeeper.GetParams(ctx).MinCreditAmount,
		time.Duration(millionstypes.DefaultUnbondingDuration),
		sdk.NewInt(millionstypes.DefaultMaxUnbondingEntries),
		millionstypes.DrawSchedule{DrawDelta: 24 * time.Hour, InitialDrawAt: ctx.BlockTime().Add(24 * time.Hour)},
		millionstypes.PrizeStrategy{PrizeBatches: []millionstypes.PrizeBatch{{PoolPercent: 100, Quantity: 100, DrawProbability: sdk.NewDec(1)}}},
		[]millionstypes.FeeTaker{
			{Destination: authtypes.FeeCollectorName, Amount: sdk.NewDecWithPrec(millionstypes.DefaultFeeTakerAmount, 2), Type: millionstypes.FeeTakerType_LocalModuleAccount},
		},
	)
	suite.Require().NoError(err)
	_, err = app.MillionsKeeper.GetPool(ctx, poolID)
	suite.Require().NoError(err)
	// Add 3 pools
	for i := 0; i < 3; i++ {
		poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
		drawDelta1 := 1 * time.Hour
		app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
			PoolId: poolID,
			PrizeStrategy: millionstypes.PrizeStrategy{
				PrizeBatches: []millionstypes.PrizeBatch{
					{PoolPercent: 100, Quantity: 1, DrawProbability: floatToDec(0.00)},
				},
			},
			DrawSchedule: millionstypes.DrawSchedule{
				InitialDrawAt: ctx.BlockTime().Add(drawDelta1),
				DrawDelta:     drawDelta1,
			},
			AvailablePrizePool: sdk.NewCoin(localPoolDenom, math.NewInt(1000)),
		}))
	}

	// Retrieve the pool from the state
	pools := app.MillionsKeeper.ListPools(ctx)

	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pools[0].PoolId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.CreditState_Failure,
		ErrorState:       millionstypes.CreditState_IbcTransfer,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[0].IcaCreditAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	credits := app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	// - Test GetPoolDraw with wrong poolID
	_, err = app.MillionsKeeper.GetPoolCredit(ctx, uint64(0), credits[0].CreditId)
	suite.Require().ErrorIs(err, millionstypes.ErrCreditNotFound)
	// - Test GetPoolDraw with wrong drawID
	_, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, uint64(0))
	suite.Require().ErrorIs(err, millionstypes.ErrCreditNotFound)

	// Test validate Basics
	// - Test unknown poolID
	_, err = msgServer.CreditRetry(goCtx, &millionstypes.MsgCreditRetry{
		PoolId:           uint64(0),
		CreditId:        credits[0].CreditId,
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)
	// - Test unknown creditID
	_, err = msgServer.CreditRetry(goCtx, &millionstypes.MsgCreditRetry{
		PoolId:           credits[0].PoolId,
		CreditId:        uint64(0),
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)

	// Should throw error if creditor Addr msg is different than entity msg
	_, err = msgServer.CreditRetry(goCtx, &millionstypes.MsgCreditRetry{
		PoolId:           credits[0].PoolId,
		CreditId:        credits[0].CreditId,
		CreditorAddress: "different-address",
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidCreditorAddress)

	// Test CreditState_IbcTransfer ErrorState
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	_, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)

	_, err = msgServer.CreditRetry(goCtx, &millionstypes.MsgCreditRetry{
		PoolId:           credits[0].PoolId,
		CreditId:        credits[0].CreditId,
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().NoError(err)
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Equal(millionstypes.CreditState_Success, credits[0].State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credits[0].ErrorState)

	// Test CreditState_IcaDelegate ErrorState
	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pools[1].PoolId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.CreditState_Failure,
		ErrorState:       millionstypes.CreditState_IcaDelegate,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[1].IcaCreditAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	_, err = msgServer.CreditRetry(goCtx, &millionstypes.MsgCreditRetry{
		PoolId:           credits[1].PoolId,
		CreditId:        credits[1].CreditId,
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().NoError(err)
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Equal(millionstypes.CreditState_Success, credits[1].State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credits[1].ErrorState)

	// Test CreditState_Unspecified ErrorState
	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pools[2].PoolId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.CreditState_Failure,
		ErrorState:       millionstypes.CreditState_Unspecified,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], suite.moduleAddrs[0], sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])

	_, err = msgServer.CreditRetry(goCtx, &millionstypes.MsgCreditRetry{
		PoolId:           credits[2].PoolId,
		CreditId:        credits[2].CreditId,
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidCreditState)
}

// TestMsgServer_CreditEdit tests the edition of a winnerAddr and sponsor
func (suite *KeeperTestSuite) TestMsgServer_CreditEdit() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	goCtx := sdk.WrapSDKContext(ctx)
	msgServer := millionskeeper.NewMsgServerImpl(*app.MillionsKeeper)

	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
	drawDelta1 := 1 * time.Hour
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId: poolID,
		PrizeStrategy: millionstypes.PrizeStrategy{
			PrizeBatches: []millionstypes.PrizeBatch{
				{PoolPercent: 100, Quantity: 1, DrawProbability: floatToDec(0.00)},
			},
		},
		DrawSchedule: millionstypes.DrawSchedule{
			InitialDrawAt: ctx.BlockTime().Add(drawDelta1),
			DrawDelta:     drawDelta1,
		},
		AvailablePrizePool: sdk.NewCoin(localPoolDenom, math.NewInt(1000)),
	}))

	// Retrieve the pool from the state
	pools := app.MillionsKeeper.ListPools(ctx)

	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pools[0].PoolId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.CreditState_Success,
		ErrorState:       millionstypes.CreditState_Unspecified,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[0].IcaCreditAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	credit, err := app.MillionsKeeper.GetPoolCredit(ctx, 1, 1)
	suite.Require().NoError(err)
	// - Test GetPoolDraw with wrong poolID
	_, err = app.MillionsKeeper.GetPoolCredit(ctx, uint64(0), credit.CreditId)
	suite.Require().ErrorIs(err, millionstypes.ErrCreditNotFound)
	// - Test GetPoolDraw with wrong drawID
	_, err = app.MillionsKeeper.GetPoolCredit(ctx, credit.PoolId, uint64(0))
	suite.Require().ErrorIs(err, millionstypes.ErrCreditNotFound)

	// Test validate Basics
	// - Test unknown poolID
	_, err = msgServer.CreditEdit(goCtx, &millionstypes.MsgCreditEdit{
		PoolId:           uint64(0),
		CreditId:        credit.CreditId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)
	// - Test unknown creditID
	_, err = msgServer.CreditEdit(goCtx, &millionstypes.MsgCreditEdit{
		PoolId:           credit.PoolId,
		CreditId:        uint64(0),
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)

	// Should throw error if creditor Addr msg is different than entity msg
	_, err = msgServer.CreditEdit(goCtx, &millionstypes.MsgCreditEdit{
		PoolId:           credit.PoolId,
		CreditId:        credit.CreditId,
		CreditorAddress: "different-address",
		WinnerAddress:    suite.addrs[0].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidCreditorAddress)

	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, 1, 1)
	suite.Require().NoError(err)

	// Should throw error for an invalid winner address
	_, err = msgServer.CreditEdit(goCtx, &millionstypes.MsgCreditEdit{
		PoolId:           credit.PoolId,
		CreditId:        credit.CreditId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    "fake-address",
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().Error(err)

	// Should update winnerAddress valid param
	_, err = msgServer.CreditEdit(goCtx, &millionstypes.MsgCreditEdit{
		PoolId:           credit.PoolId,
		CreditId:        credit.CreditId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().NoError(err)
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, 1, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(suite.addrs[1].String(), credit.WinnerAddress)

	// Sponsor can be nil and default to initial credit isSponsor
	_, err = msgServer.CreditEdit(goCtx, &millionstypes.MsgCreditEdit{
		PoolId:           credit.PoolId,
		CreditId:        credit.CreditId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
		IsSponsor:        nil,
	})
	suite.Require().NoError(err)
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, 1, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(false, credit.IsSponsor)
	pool, err := app.MillionsKeeper.GetPool(ctx, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(0), pool.SponsorshipAmount.Int64())

	// Should update sponsor for credit and pool SponsorshipAmount for valid sponsor param
	_, err = msgServer.CreditEdit(goCtx, &millionstypes.MsgCreditEdit{
		PoolId:           credit.PoolId,
		CreditId:        credit.CreditId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: true,
		},
	})
	suite.Require().NoError(err)
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, 1, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(true, credit.IsSponsor)
	pool, err = app.MillionsKeeper.GetPool(ctx, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(1_000_000), pool.SponsorshipAmount.Int64())

	pool, err = app.MillionsKeeper.GetPool(ctx, 1)
	suite.Require().NoError(err)

	// A sponsor cannot designate another winner address
	_, err = msgServer.CreditEdit(goCtx, &millionstypes.MsgCreditEdit{
		PoolId:           credit.PoolId,
		CreditId:        credit.CreditId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
		IsSponsor:        nil,
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidSponsorWinnerCombo)

	// Sponsor can be nil and default to initial credit isSponsor -> In this case true
	_, err = msgServer.CreditEdit(goCtx, &millionstypes.MsgCreditEdit{
		PoolId:           credit.PoolId,
		CreditId:        credit.CreditId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		IsSponsor:        nil,
	})
	suite.Require().NoError(err)
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, 1, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(true, credit.IsSponsor)
	pool, err = app.MillionsKeeper.GetPool(ctx, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(1_000_000), pool.SponsorshipAmount.Int64())

	// Should sub sponsor amount if user is not willing to sponsor anymore
	_, err = msgServer.CreditEdit(goCtx, &millionstypes.MsgCreditEdit{
		PoolId:           credit.PoolId,
		CreditId:        credit.CreditId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().NoError(err)
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, 1, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(false, credit.IsSponsor)
	pool, err = app.MillionsKeeper.GetPool(ctx, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(0), pool.SponsorshipAmount.Int64())
}

// TestMsgServer_WithdrawCredit runs withdrawal credit related tests
func (suite *KeeperTestSuite) TestMsgServer_WithdrawCredit() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	goCtx := sdk.WrapSDKContext(ctx)
	msgServer := millionskeeper.NewMsgServerImpl(*app.MillionsKeeper)
	uatomAddresses := apptesting.AddTestAddrsWithDenom(app, ctx, 7, sdk.NewInt(1_000_0000_000), "uatom")

	// Initialize the pool
	poolID, err := app.MillionsKeeper.RegisterPool(ctx,
		millionstypes.PoolType_Staking,
		"ufury",
		"ufury",
		testChainID,
		"",
		"",
		[]string{suite.valAddrs[0].String()},
		"furya",
		"furyavaloper",
		app.MillionsKeeper.GetParams(ctx).MinCreditAmount,
		time.Duration(millionstypes.DefaultUnbondingDuration),
		sdk.NewInt(millionstypes.DefaultMaxUnbondingEntries),
		millionstypes.DrawSchedule{DrawDelta: 24 * time.Hour, InitialDrawAt: ctx.BlockTime().Add(24 * time.Hour)},
		millionstypes.PrizeStrategy{PrizeBatches: []millionstypes.PrizeBatch{{PoolPercent: 100, Quantity: 100, DrawProbability: sdk.NewDec(1)}}},
		[]millionstypes.FeeTaker{
			{Destination: authtypes.FeeCollectorName, Amount: sdk.NewDecWithPrec(millionstypes.DefaultFeeTakerAmount, 2), Type: millionstypes.FeeTakerType_LocalModuleAccount},
		},
	)
	suite.Require().NoError(err)
	_, err = app.MillionsKeeper.GetPool(ctx, poolID)
	suite.Require().NoError(err)
	poolID = app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
	drawDelta1 := 1 * time.Hour
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId: poolID,
		PrizeStrategy: millionstypes.PrizeStrategy{
			PrizeBatches: []millionstypes.PrizeBatch{
				{PoolPercent: 100, Quantity: 1, DrawProbability: floatToDec(0.00)},
			},
		},
		DrawSchedule: millionstypes.DrawSchedule{
			InitialDrawAt: ctx.BlockTime().Add(drawDelta1),
			DrawDelta:     drawDelta1,
		},
		AvailablePrizePool: sdk.NewCoin(localPoolDenom, math.NewInt(1000)),
	}))
	pools := app.MillionsKeeper.ListPools(ctx)
	pool, err := app.MillionsKeeper.GetPool(ctx, pools[0].PoolId)
	suite.Require().NoError(err)

	// Create 2 successful credits
	for i := 0; i < 2; i++ {
		// Create second credit
		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           pool.PoolId,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
		credits := app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
		err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[0].IcaCreditAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
		suite.Require().NoError(err)
		// Trigger the Transfer credit to native chain
		err = app.MillionsKeeper.TransferCreditToRemoteZone(ctx, credits[i].PoolId, credits[i].CreditId)
		suite.Require().NoError(err)
	}
	// Create credit with failure state
	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pool.PoolId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.CreditState_Failure,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	// List credits
	credits := app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Len(credits, 3)

	// Create invalid Withdraw request
	// - Wrong creditorAddress
	_, err = msgServer.WithdrawCredit(goCtx, &millionstypes.MsgWithdrawCredit{
		PoolId:           credits[0].PoolId,
		CreditId:        credits[0].CreditId,
		CreditorAddress: "",
		ToAddress:        suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidCreditorAddress)

	// - Wrong to_address
	_, err = msgServer.WithdrawCredit(goCtx, &millionstypes.MsgWithdrawCredit{
		PoolId:           credits[0].PoolId,
		CreditId:        credits[0].CreditId,
		CreditorAddress: suite.addrs[0].String(),
		ToAddress:        "",
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidDestinationAddress)

	// - Wrong credit_id
	_, err = msgServer.WithdrawCredit(goCtx, &millionstypes.MsgWithdrawCredit{
		PoolId:           pool.PoolId,
		CreditId:        uint64(0),
		CreditorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrCreditNotFound)

	// - Wrong state
	_, err = msgServer.WithdrawCredit(goCtx, &millionstypes.MsgWithdrawCredit{
		PoolId:           credits[2].PoolId,
		CreditId:        credits[2].CreditId,
		CreditorAddress: credits[2].CreditorAddress,
		ToAddress:        suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidCreditState)

	// - Wront creditor and withdraw address
	_, err = msgServer.WithdrawCredit(goCtx, &millionstypes.MsgWithdrawCredit{
		PoolId:           pool.PoolId,
		CreditId:        credits[1].CreditId,
		CreditorAddress: suite.addrs[3].String(),
		ToAddress:        suite.addrs[3].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidWithdrawalCreditorAddress)

	// Add a successful withdrawal should remove a credit
	_, err = msgServer.WithdrawCredit(goCtx, &millionstypes.MsgWithdrawCredit{
		PoolId:           credits[0].PoolId,
		CreditId:        credits[0].CreditId,
		CreditorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
	})
	suite.Require().NoError(err)
	// Make sure we have the correct number of withdrawals
	withdrawals := app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 1)
	// State should be unbonding
	suite.Require().Equal(millionstypes.WithdrawalState_Pending, withdrawals[0].State)
	suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawals[0].ErrorState)

	// Two credits only should remain
	credits = app.MillionsKeeper.ListCredits(ctx)
	suite.Require().Len(credits, 2)

	// Add a successful withdrawal should remove a credit
	_, err = msgServer.WithdrawCredit(goCtx, &millionstypes.MsgWithdrawCredit{
		PoolId:           credits[0].PoolId,
		CreditId:        credits[0].CreditId,
		CreditorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
	})
	suite.Require().NoError(err)

	// Make sure we have the correct number of withdrawals
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 2)
	// State should be unbonding
	suite.Require().Equal(millionstypes.WithdrawalState_Pending, withdrawals[1].State)
	suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawals[1].ErrorState)

	// One credit only should remain
	credits = app.MillionsKeeper.ListCredits(ctx)
	suite.Require().Len(credits, 1)

	// Initialize a remote pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:              3,
		Bech32PrefixValAddr: remoteBech32PrefixValAddr,
		Bech32PrefixAccAddr: remoteBech32PrefixAccAddr,
		ChainId:             remoteChainId,
		Denom:               remotePoolDenom,
		NativeDenom:         remotePoolDenom,
		ConnectionId:        remoteConnectionId,
		TransferChannelId:   remoteTransferChannelId,
		Validators: []millionstypes.PoolValidator{
			{
				OperatorAddress: cosmosPoolValidator,
				BondedAmount:    sdk.NewInt(1_000_000),
				IsEnabled:       true,
			},
		},
		IcaCreditAddress:   cosmosIcaCreditAddress,
		IcaPrizepoolAddress: cosmosIcaPrizePoolAddress,
		PrizeStrategy: millionstypes.PrizeStrategy{
			PrizeBatches: []millionstypes.PrizeBatch{
				{PoolPercent: 100, Quantity: 1, DrawProbability: floatToDec(0.00)},
			},
		},
		DrawSchedule: millionstypes.DrawSchedule{
			InitialDrawAt: ctx.BlockTime().Add(drawDelta1),
			DrawDelta:     drawDelta1,
		},
		AvailablePrizePool: sdk.NewCoin(remotePoolDenom, math.NewInt(1000)),
	}))
	pools = app.MillionsKeeper.ListPools(ctx)
	pool, err = app.MillionsKeeper.GetPool(ctx, pools[2].PoolId)
	suite.Require().NoError(err)

	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pool.PoolId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.CreditState_Success,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
	})
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	err = app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[2].LocalAddress), sdk.Coins{sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	pools = app.MillionsKeeper.ListPools(ctx)
	pool, err = app.MillionsKeeper.GetPool(ctx, pools[2].PoolId)
	suite.Require().NoError(err)

	_, err = msgServer.WithdrawCredit(goCtx, &millionstypes.MsgWithdrawCredit{
		PoolId:           credits[1].PoolId,
		CreditId:        credits[1].CreditId,
		CreditorAddress: suite.addrs[0].String(),
		ToAddress:        "osmo11qggwr7vze9x7ustru9dctf6krhk2285lkf89g7",
	})
	// Error is triggered as toAddress does not match pool bech prefix and neither furya AccAddressFromBech32
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidDestinationAddress)

	_, err = msgServer.WithdrawCredit(goCtx, &millionstypes.MsgWithdrawCredit{
		PoolId:           credits[1].PoolId,
		CreditId:        credits[1].CreditId,
		CreditorAddress: suite.addrs[0].String(),
		ToAddress:        cosmosAccAddress,
	})
	suite.Require().NoError(err)

	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Equal(millionstypes.WithdrawalState_Pending, withdrawals[len(withdrawals)-1].State)
}

// TestMsgServer_WithdrawCreditRetry runs withdrawal retry related tests
func (suite *KeeperTestSuite) TestMsgServer_WithdrawCreditRetry() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	goCtx := sdk.WrapSDKContext(ctx)
	msgServer := millionskeeper.NewMsgServerImpl(*app.MillionsKeeper)
	drawDelta1 := 1 * time.Hour
	var now = time.Now().UTC()
	frequency := uint64(4)
	// Assuming first epoch is 1, and nextEpochUnbonding is the 4th one
	for epoch := int64(1); epoch <= 4; epoch++ {
		epochInfo, err := TriggerEpochUpdate(suite)
		suite.Require().NoError(err)
		suite.Require().Equal(epoch, epochInfo.CurrentEpoch)

		_, err = TriggerEpochTrackerUpdate(suite, epochInfo)
		suite.Require().NoError(err)
	}

	pool := newValidPool(suite, millionstypes.Pool{
		PoolId:      uint64(1),
		Denom:       localPoolDenom,
		NativeDenom: localPoolDenom,
		Validators: []millionstypes.PoolValidator{{
			OperatorAddress: suite.valAddrs[0].String(),
			BondedAmount:    sdk.NewInt(1_000_000),
			IsEnabled:       true,
		}},
		PrizeStrategy: millionstypes.PrizeStrategy{
			PrizeBatches: []millionstypes.PrizeBatch{
				{PoolPercent: 100, Quantity: 1, DrawProbability: floatToDec(0.00)},
			},
		},
		DrawSchedule: millionstypes.DrawSchedule{
			InitialDrawAt: ctx.BlockTime().Add(drawDelta1),
			DrawDelta:     drawDelta1,
		},
		AvailablePrizePool: sdk.NewCoin(localPoolDenom, math.NewInt(1000)),
	})
	app.MillionsKeeper.AddPool(ctx, pool)

	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pool.PoolId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.CreditState_IbcTransfer,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pool.IcaCreditAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	credits := app.MillionsKeeper.ListCredits(ctx)
	// Simulate transfer credit and delegate to native chain
	err = app.MillionsKeeper.TransferCreditToRemoteZone(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	credits = app.MillionsKeeper.ListCredits(ctx)
	err = app.MillionsKeeper.DelegateCreditOnRemoteZone(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().Error(err)

	credits = app.MillionsKeeper.ListCredits(ctx)

	app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
		PoolId:           credits[0].PoolId,
		CreditId:        credits[0].CreditId,
		State:            millionstypes.WithdrawalState_Pending,
		ErrorState:       millionstypes.WithdrawalState_Unspecified,
		CreditorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(1_000_000))),
		UnbondingEndsAt:  &time.Time{},
	})

	credit, err := app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	// Remove credit to respect the flow
	app.MillionsKeeper.RemoveCredit(ctx, &credit)

	withdrawals := app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 1)

	err = app.MillionsKeeper.AddEpochUnbonding(ctx, withdrawals[0], false)
	suite.Require().NoError(err)

	// Test ValidateWithdrawCreditRetryBasic
	// Test withdrawal retry with invalid poolID
	// - Test unknown poolID
	_, err = msgServer.WithdrawCreditRetry(goCtx, &millionstypes.MsgWithdrawCreditRetry{
		PoolId:           uint64(0),
		WithdrawalId:     withdrawals[0].WithdrawalId,
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)

	// - Test unknown withdrawalID
	_, err = msgServer.WithdrawCreditRetry(goCtx, &millionstypes.MsgWithdrawCreditRetry{
		PoolId:           withdrawals[0].PoolId,
		WithdrawalId:     uint64(0),
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)

	// Test GetPoolWithdrawal
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 1)

	// Test GetPoolWithdrawal with wrong poolID
	_, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, uint64(0), withdrawals[0].WithdrawalId)
	suite.Require().ErrorIs(err, millionstypes.ErrWithdrawalNotFound)

	// Test GetPoolWithdrawal with wrong withdrawalID
	_, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, uint64(0))
	suite.Require().ErrorIs(err, millionstypes.ErrWithdrawalNotFound)

	// Test if creditorAddr msg is the same as pool withdrawal creditor entity
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	_, err = msgServer.WithdrawCreditRetry(goCtx, &millionstypes.MsgWithdrawCreditRetry{
		PoolId:           withdrawals[0].PoolId,
		WithdrawalId:     withdrawals[0].PoolId,
		CreditorAddress: "different-address",
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidCreditorAddress)

	// Get the millions internal module tracker
	epochTracker, err := app.MillionsKeeper.GetEpochTracker(ctx, epochstypes.DAY_EPOCH, millionstypes.WithdrawalTrackerType)
	suite.Require().NoError(err)

	// Get epoch unbonding
	currentEpochUnbonding, err := app.MillionsKeeper.GetEpochPoolUnbonding(ctx, epochTracker.EpochNumber+frequency, withdrawals[0].PoolId)
	suite.Require().NoError(err)

	err = app.MillionsKeeper.UndelegateWithdrawalsOnRemoteZone(ctx, currentEpochUnbonding)
	suite.Require().NoError(err)

	// move 21 days unbonding onwards and force unbonding
	ctx = ctx.WithBlockTime(now.Add(21 * 24 * time.Hour))
	goCtx = ctx
	_, err = app.StakingKeeper.CompleteUnbonding(ctx, sdk.MustAccAddressFromBech32(pool.IcaCreditAddress), suite.valAddrs[0])
	suite.Require().NoError(err)

	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 1)

	// Should have the correct state
	suite.Require().Equal(millionstypes.WithdrawalState_IcaUnbonding, withdrawals[0].State)
	suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawals[0].ErrorState)

	// Update to test the ibc transfer
	app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId, millionstypes.WithdrawalState_IbcTransfer, withdrawals[0].UnbondingEndsAt, true)

	// Test WithdrawalState_IbcTransfer ErrorState
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	_, err = msgServer.WithdrawCreditRetry(goCtx, &millionstypes.MsgWithdrawCreditRetry{
		PoolId:           withdrawals[0].PoolId,
		WithdrawalId:     withdrawals[0].WithdrawalId,
		CreditorAddress: suite.addrs[0].String(),
	})
	suite.Require().NoError(err)
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	// Withdrawals should be removed upon completion
	suite.Require().Len(withdrawals, 0)
}

// TestMsgServer_ClaimPrize runs prize claim related tests
func (suite *KeeperTestSuite) TestMsgServer_ClaimPrize() {
	app := suite.app
	ctx := suite.ctx
	params := app.MillionsKeeper.GetParams(ctx)
	goCtx := sdk.WrapSDKContext(ctx)
	msgServer := millionskeeper.NewMsgServerImpl(*app.MillionsKeeper)

	// Create pool and simulate module account address
	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
	pool := newValidPool(suite, millionstypes.Pool{
		PoolId: poolID,
		PrizeStrategy: millionstypes.PrizeStrategy{
			PrizeBatches: []millionstypes.PrizeBatch{
				{PoolPercent: 50, Quantity: 100, DrawProbability: floatToDec(0.5)},
				{PoolPercent: 50, Quantity: 200, DrawProbability: floatToDec(0.7)},
			},
		},
	})
	app.MillionsKeeper.AddPool(ctx, pool)

	// Create prizes at various stages
	app.MillionsKeeper.AddPrize(ctx, millionstypes.Prize{
		PoolId:        poolID,
		DrawId:        pool.NextDrawId,
		PrizeId:       1,
		Amount:        sdk.NewCoin(pool.Denom, sdk.NewInt(1_000_000)),
		WinnerAddress: suite.addrs[1].String(),
		CreatedAt:     ctx.BlockTime(),
		ExpiresAt:     ctx.BlockTime().Add(params.PrizeExpirationDelta),
		State:         millionstypes.PrizeState_Pending,
	})
	app.MillionsKeeper.AddPrize(ctx, millionstypes.Prize{
		PoolId:        poolID,
		DrawId:        pool.NextDrawId,
		PrizeId:       2,
		Amount:        sdk.NewCoin(pool.Denom, sdk.NewInt(2_000_000)),
		WinnerAddress: suite.addrs[1].String(),
		CreatedAt:     ctx.BlockTime(),
		ExpiresAt:     ctx.BlockTime().Add(params.PrizeExpirationDelta),
		State:         millionstypes.PrizeState_Pending,
	})
	app.MillionsKeeper.AddPrize(ctx, millionstypes.Prize{
		PoolId:        poolID,
		DrawId:        pool.NextDrawId,
		PrizeId:       3,
		Amount:        sdk.NewCoin(pool.Denom, sdk.NewInt(3_000_000)),
		WinnerAddress: suite.addrs[1].String(),
		CreatedAt:     ctx.BlockTime(),
		ExpiresAt:     ctx.BlockTime().Add(params.PrizeExpirationDelta),
		State:         millionstypes.PrizeState_Pending,
	})

	// List Prizes
	prizes := app.MillionsKeeper.ListPrizes(ctx)
	suite.Require().Len(prizes, 3)
	suite.Require().Equal(millionstypes.PrizeState_Pending, prizes[0].State)
	suite.Require().Equal(millionstypes.PrizeState_Pending, prizes[1].State)
	suite.Require().Equal(millionstypes.PrizeState_Pending, prizes[2].State)

	// Send required amounts to pool account
	modAddr, err := sdk.AccAddressFromBech32(pool.GetLocalAddress())
	suite.Require().NoError(err)
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], modAddr, sdk.NewCoins(prizes[0].Amount.Add(prizes[1].Amount).Add(prizes[2].Amount)))
	suite.Require().NoError(err)

	// Test invalid claim requests
	_, err = msgServer.ClaimPrize(goCtx, millionstypes.NewMsgMsgClaimPrize("", prizes[0].PoolId, prizes[0].DrawId, prizes[0].PrizeId))
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidWinnerAddress)
	_, err = msgServer.ClaimPrize(goCtx, millionstypes.NewMsgMsgClaimPrize(suite.addrs[0].String(), prizes[0].PoolId, prizes[0].DrawId, prizes[0].PrizeId))
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidWinnerAddress)

	_, err = msgServer.ClaimPrize(goCtx, millionstypes.NewMsgMsgClaimPrize(suite.addrs[1].String(), prizes[0].PoolId+1, prizes[0].DrawId, prizes[0].PrizeId))
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, millionstypes.ErrPrizeNotFound)
	_, err = msgServer.ClaimPrize(goCtx, millionstypes.NewMsgMsgClaimPrize(suite.addrs[1].String(), prizes[0].PoolId, prizes[0].DrawId+1, prizes[0].PrizeId))
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, millionstypes.ErrPrizeNotFound)
	_, err = msgServer.ClaimPrize(goCtx, millionstypes.NewMsgMsgClaimPrize(suite.addrs[1].String(), prizes[0].PoolId, prizes[0].DrawId, prizes[2].PrizeId+1))
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, millionstypes.ErrPrizeNotFound)

	prizes = app.MillionsKeeper.ListPrizes(ctx)

	for _, prize := range prizes {
		addr := sdk.MustAccAddressFromBech32(prize.WinnerAddress)
		// Get the module balance before claimPrize
		moduleBalanceBefore := app.BankKeeper.GetBalance(ctx, modAddr, pool.Denom)
		// Get the balance before claimPrize
		balanceBefore := app.BankKeeper.GetBalance(ctx, addr, pool.Denom)
		// claimPrize
		_, err = msgServer.ClaimPrize(goCtx, millionstypes.NewMsgMsgClaimPrize(addr.String(), prize.PoolId, prize.DrawId, prize.PrizeId))
		suite.Require().NoError(err)
		// Compare the new balance with the expected balance
		balance := app.BankKeeper.GetBalance(ctx, addr, pool.Denom)
		expectedBalance := balanceBefore.Add(prize.Amount)
		suite.Require().True(expectedBalance.Equal(balance))
		// Compare the new module balance with the expected module balance
		expectedModuleBalance := moduleBalanceBefore.Sub(prize.Amount)
		moduleBalance := app.BankKeeper.GetBalance(ctx, modAddr, pool.Denom)
		suite.Require().True(expectedModuleBalance.Equal(moduleBalance))
	}
	// There should be no more prizes
	prizes = app.MillionsKeeper.ListPrizes(ctx)
	suite.Require().Len(prizes, 0)
}
