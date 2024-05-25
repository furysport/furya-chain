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

// TestMsgServer_Payment runs payment related tests
func (suite *KeeperTestSuite) TestMsgServer_Payment() {
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
	_, err := msgServer.Payment(goCtx, &millionstypes.MsgPayment{
		PoolId:           uint64(0),
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		PaymentorAddress: suite.addrs[0].String(),
	})
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, millionstypes.ErrPoolNotFound)

	// Invalid paymentor address
	_, err = msgServer.Payment(goCtx, &millionstypes.MsgPayment{
		PoolId:           poolID,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(1_000_000))),
		PaymentorAddress: "",
	})
	suite.Require().Error(err)
	suite.Require().Equal(err, millionstypes.ErrInvalidPaymentorAddress)

	// Invalid payment denom
	_, err = msgServer.Payment(goCtx, &millionstypes.MsgPayment{
		PoolId:           poolID,
		Amount:           sdk.NewCoin("ufaketoken", sdk.NewInt(1_000_000)),
		PaymentorAddress: suite.addrs[0].String(),
	})
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidPaymentDenom)

	// Invalid amount
	_, err = msgServer.Payment(goCtx, &millionstypes.MsgPayment{
		PoolId:           poolID,
		Amount:           sdk.NewCoin(localPoolDenom, pools[0].MinPaymentAmount.Sub(sdk.NewInt(1))),
		PaymentorAddress: suite.addrs[0].String(),
	})
	suite.Require().Error(err)
	suite.Require().ErrorIs(err, millionstypes.ErrInsufficientPaymentAmount)

	// Invalid winner address
	_, err = msgServer.Payment(goCtx, &millionstypes.MsgPayment{
		PoolId:           poolID,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(1_000_000))),
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    "no-address",
	})
	suite.Require().Error(err)
	suite.Require().Equal(err, millionstypes.ErrInvalidWinnerAddress)

	_, err = msgServer.Payment(goCtx, &millionstypes.MsgPayment{
		PoolId:           poolID,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(123456))),
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
		IsSponsor:        true,
	})
	suite.Require().Error(err)
	suite.Require().Equal(err, millionstypes.ErrInvalidSponsorWinnerCombo)

	// Register real pool and apply real payments
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
		app.MillionsKeeper.GetParams(ctx).MinPaymentAmount,
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
	// Create pool and payments
	// Operate basic checks to verify if payment is created
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
	_, err = msgServer.Payment(goCtx, &millionstypes.MsgPayment{
		PoolId:           pool.GetPoolId(),
		Amount:           sdk.NewCoin(pool.Denom, sdk.NewInt(int64(1_000_000))),
		PaymentorAddress: suite.addrs[0].String(),
	})
	suite.Require().NoError(err)
	balanceNow := app.BankKeeper.GetBalance(ctx, suite.addrs[0], localPoolDenom)
	suite.Require().Equal(balanceBefore.Amount.Int64()-1_000_000, balanceNow.Amount.Int64())

	_, err = msgServer.Payment(goCtx, &millionstypes.MsgPayment{
		PoolId:           pool.GetPoolId(),
		Amount:           sdk.NewCoin(pool.Denom, sdk.NewInt(int64(2_000_000))),
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
	})
	suite.Require().NoError(err)
	balanceNow = app.BankKeeper.GetBalance(ctx, suite.addrs[0], localPoolDenom)
	suite.Require().Equal(balanceBefore.Amount.Int64()-3_000_000, balanceNow.Amount.Int64())

	balanceBefore = app.BankKeeper.GetBalance(ctx, suite.addrs[2], localPoolDenom)
	_, err = msgServer.Payment(goCtx, &millionstypes.MsgPayment{
		PoolId:           pool.GetPoolId(),
		Amount:           sdk.NewCoin(pool.Denom, sdk.NewInt(int64(5_000_000))),
		PaymentorAddress: suite.addrs[2].String(),
		IsSponsor:        true,
	})
	suite.Require().NoError(err)
	balanceNow = app.BankKeeper.GetBalance(ctx, suite.addrs[2], localPoolDenom)
	suite.Require().Equal(balanceBefore.Amount.Int64()-5_000_000, balanceNow.Amount.Int64())

	// Compare pool tvl and paymentors count state
	suite.Require().Equal(sdk.NewInt(0), pool.TvlAmount)
	suite.Require().Equal(uint64(0), pool.PaymentorsCount)
	pool, err = app.MillionsKeeper.GetPool(ctx, pool.GetPoolId())
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewInt(8_000_000), pool.TvlAmount)
	suite.Require().Equal(uint64(2), pool.PaymentorsCount)
	suite.Require().Equal(sdk.NewInt(int64(5_000_000)), pool.SponsorshipAmount)

	// Test payment for a pool that is not in ready state
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

	_, err = msgServer.Payment(goCtx, &millionstypes.MsgPayment{
		PoolId:           pool.PoolId,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(1_000_000))),
		PaymentorAddress: suite.addrs[0].String(),
	})
	// Payment should be allowed for paused pools
	suite.Require().NoError(err)
	pool, err = app.MillionsKeeper.GetPool(ctx, 4)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(1_000_000), pool.TvlAmount.Int64())
}

// TestMsgServer_PaymentRetry tests the retry of a failed payment
func (suite *KeeperTestSuite) TestMsgServer_PaymentRetry() {
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
		app.MillionsKeeper.GetParams(ctx).MinPaymentAmount,
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

	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[0].PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Failure,
		ErrorState:       millionstypes.PaymentState_IbcTransfer,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[0].IcaPaymentAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	payments := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	// - Test GetPoolDraw with wrong poolID
	_, err = app.MillionsKeeper.GetPoolPayment(ctx, uint64(0), payments[0].PaymentId)
	suite.Require().ErrorIs(err, millionstypes.ErrPaymentNotFound)
	// - Test GetPoolDraw with wrong drawID
	_, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, uint64(0))
	suite.Require().ErrorIs(err, millionstypes.ErrPaymentNotFound)

	// Test validate Basics
	// - Test unknown poolID
	_, err = msgServer.PaymentRetry(goCtx, &millionstypes.MsgPaymentRetry{
		PoolId:           uint64(0),
		PaymentId:        payments[0].PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)
	// - Test unknown paymentID
	_, err = msgServer.PaymentRetry(goCtx, &millionstypes.MsgPaymentRetry{
		PoolId:           payments[0].PoolId,
		PaymentId:        uint64(0),
		PaymentorAddress: suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)

	// Should throw error if paymentor Addr msg is different than entity msg
	_, err = msgServer.PaymentRetry(goCtx, &millionstypes.MsgPaymentRetry{
		PoolId:           payments[0].PoolId,
		PaymentId:        payments[0].PaymentId,
		PaymentorAddress: "different-address",
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidPaymentorAddress)

	// Test PaymentState_IbcTransfer ErrorState
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	_, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)

	_, err = msgServer.PaymentRetry(goCtx, &millionstypes.MsgPaymentRetry{
		PoolId:           payments[0].PoolId,
		PaymentId:        payments[0].PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
	})
	suite.Require().NoError(err)
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Equal(millionstypes.PaymentState_Success, payments[0].State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payments[0].ErrorState)

	// Test PaymentState_IcaDelegate ErrorState
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[1].PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Failure,
		ErrorState:       millionstypes.PaymentState_IcaDelegate,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	_, err = msgServer.PaymentRetry(goCtx, &millionstypes.MsgPaymentRetry{
		PoolId:           payments[1].PoolId,
		PaymentId:        payments[1].PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
	})
	suite.Require().NoError(err)
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Equal(millionstypes.PaymentState_Success, payments[1].State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payments[1].ErrorState)

	// Test PaymentState_Unspecified ErrorState
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[2].PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Failure,
		ErrorState:       millionstypes.PaymentState_Unspecified,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], suite.moduleAddrs[0], sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])

	_, err = msgServer.PaymentRetry(goCtx, &millionstypes.MsgPaymentRetry{
		PoolId:           payments[2].PoolId,
		PaymentId:        payments[2].PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidPaymentState)
}

// TestMsgServer_PaymentEdit tests the edition of a winnerAddr and sponsor
func (suite *KeeperTestSuite) TestMsgServer_PaymentEdit() {
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

	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[0].PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Success,
		ErrorState:       millionstypes.PaymentState_Unspecified,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[0].IcaPaymentAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	payment, err := app.MillionsKeeper.GetPoolPayment(ctx, 1, 1)
	suite.Require().NoError(err)
	// - Test GetPoolDraw with wrong poolID
	_, err = app.MillionsKeeper.GetPoolPayment(ctx, uint64(0), payment.PaymentId)
	suite.Require().ErrorIs(err, millionstypes.ErrPaymentNotFound)
	// - Test GetPoolDraw with wrong drawID
	_, err = app.MillionsKeeper.GetPoolPayment(ctx, payment.PoolId, uint64(0))
	suite.Require().ErrorIs(err, millionstypes.ErrPaymentNotFound)

	// Test validate Basics
	// - Test unknown poolID
	_, err = msgServer.PaymentEdit(goCtx, &millionstypes.MsgPaymentEdit{
		PoolId:           uint64(0),
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)
	// - Test unknown paymentID
	_, err = msgServer.PaymentEdit(goCtx, &millionstypes.MsgPaymentEdit{
		PoolId:           payment.PoolId,
		PaymentId:        uint64(0),
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)

	// Should throw error if paymentor Addr msg is different than entity msg
	_, err = msgServer.PaymentEdit(goCtx, &millionstypes.MsgPaymentEdit{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: "different-address",
		WinnerAddress:    suite.addrs[0].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidPaymentorAddress)

	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, 1, 1)
	suite.Require().NoError(err)

	// Should throw error for an invalid winner address
	_, err = msgServer.PaymentEdit(goCtx, &millionstypes.MsgPaymentEdit{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    "fake-address",
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().Error(err)

	// Should update winnerAddress valid param
	_, err = msgServer.PaymentEdit(goCtx, &millionstypes.MsgPaymentEdit{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().NoError(err)
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, 1, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(suite.addrs[1].String(), payment.WinnerAddress)

	// Sponsor can be nil and default to initial payment isSponsor
	_, err = msgServer.PaymentEdit(goCtx, &millionstypes.MsgPaymentEdit{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
		IsSponsor:        nil,
	})
	suite.Require().NoError(err)
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, 1, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(false, payment.IsSponsor)
	pool, err := app.MillionsKeeper.GetPool(ctx, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(0), pool.SponsorshipAmount.Int64())

	// Should update sponsor for payment and pool SponsorshipAmount for valid sponsor param
	_, err = msgServer.PaymentEdit(goCtx, &millionstypes.MsgPaymentEdit{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: true,
		},
	})
	suite.Require().NoError(err)
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, 1, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(true, payment.IsSponsor)
	pool, err = app.MillionsKeeper.GetPool(ctx, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(1_000_000), pool.SponsorshipAmount.Int64())

	pool, err = app.MillionsKeeper.GetPool(ctx, 1)
	suite.Require().NoError(err)

	// A sponsor cannot designate another winner address
	_, err = msgServer.PaymentEdit(goCtx, &millionstypes.MsgPaymentEdit{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
		IsSponsor:        nil,
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidSponsorWinnerCombo)

	// Sponsor can be nil and default to initial payment isSponsor -> In this case true
	_, err = msgServer.PaymentEdit(goCtx, &millionstypes.MsgPaymentEdit{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		IsSponsor:        nil,
	})
	suite.Require().NoError(err)
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, 1, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(true, payment.IsSponsor)
	pool, err = app.MillionsKeeper.GetPool(ctx, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(1_000_000), pool.SponsorshipAmount.Int64())

	// Should sub sponsor amount if user is not willing to sponsor anymore
	_, err = msgServer.PaymentEdit(goCtx, &millionstypes.MsgPaymentEdit{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[1].String(),
		IsSponsor: &gogotypes.BoolValue{
			Value: false,
		},
	})
	suite.Require().NoError(err)
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, 1, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(false, payment.IsSponsor)
	pool, err = app.MillionsKeeper.GetPool(ctx, 1)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(0), pool.SponsorshipAmount.Int64())
}

// TestMsgServer_WithdrawPayment runs withdrawal payment related tests
func (suite *KeeperTestSuite) TestMsgServer_WithdrawPayment() {
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
		app.MillionsKeeper.GetParams(ctx).MinPaymentAmount,
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

	// Create 2 successful payments
	for i := 0; i < 2; i++ {
		// Create second payment
		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           pool.PoolId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
		payments := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
		err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[0].IcaPaymentAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
		suite.Require().NoError(err)
		// Trigger the Transfer payment to native chain
		err = app.MillionsKeeper.TransferPaymentToRemoteZone(ctx, payments[i].PoolId, payments[i].PaymentId)
		suite.Require().NoError(err)
	}
	// Create payment with failure state
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pool.PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Failure,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	// List payments
	payments := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 3)

	// Create invalid Withdraw request
	// - Wrong paymentorAddress
	_, err = msgServer.WithdrawPayment(goCtx, &millionstypes.MsgWithdrawPayment{
		PoolId:           payments[0].PoolId,
		PaymentId:        payments[0].PaymentId,
		PaymentorAddress: "",
		ToAddress:        suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidPaymentorAddress)

	// - Wrong to_address
	_, err = msgServer.WithdrawPayment(goCtx, &millionstypes.MsgWithdrawPayment{
		PoolId:           payments[0].PoolId,
		PaymentId:        payments[0].PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        "",
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidDestinationAddress)

	// - Wrong payment_id
	_, err = msgServer.WithdrawPayment(goCtx, &millionstypes.MsgWithdrawPayment{
		PoolId:           pool.PoolId,
		PaymentId:        uint64(0),
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrPaymentNotFound)

	// - Wrong state
	_, err = msgServer.WithdrawPayment(goCtx, &millionstypes.MsgWithdrawPayment{
		PoolId:           payments[2].PoolId,
		PaymentId:        payments[2].PaymentId,
		PaymentorAddress: payments[2].PaymentorAddress,
		ToAddress:        suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidPaymentState)

	// - Wront paymentor and withdraw address
	_, err = msgServer.WithdrawPayment(goCtx, &millionstypes.MsgWithdrawPayment{
		PoolId:           pool.PoolId,
		PaymentId:        payments[1].PaymentId,
		PaymentorAddress: suite.addrs[3].String(),
		ToAddress:        suite.addrs[3].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidWithdrawalPaymentorAddress)

	// Add a successful withdrawal should remove a payment
	_, err = msgServer.WithdrawPayment(goCtx, &millionstypes.MsgWithdrawPayment{
		PoolId:           payments[0].PoolId,
		PaymentId:        payments[0].PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
	})
	suite.Require().NoError(err)
	// Make sure we have the correct number of withdrawals
	withdrawals := app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 1)
	// State should be unbonding
	suite.Require().Equal(millionstypes.WithdrawalState_Pending, withdrawals[0].State)
	suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawals[0].ErrorState)

	// Two payments only should remain
	payments = app.MillionsKeeper.ListPayments(ctx)
	suite.Require().Len(payments, 2)

	// Add a successful withdrawal should remove a payment
	_, err = msgServer.WithdrawPayment(goCtx, &millionstypes.MsgWithdrawPayment{
		PoolId:           payments[0].PoolId,
		PaymentId:        payments[0].PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
	})
	suite.Require().NoError(err)

	// Make sure we have the correct number of withdrawals
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 2)
	// State should be unbonding
	suite.Require().Equal(millionstypes.WithdrawalState_Pending, withdrawals[1].State)
	suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawals[1].ErrorState)

	// One payment only should remain
	payments = app.MillionsKeeper.ListPayments(ctx)
	suite.Require().Len(payments, 1)

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
		IcaPaymentAddress:   cosmosIcaPaymentAddress,
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

	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pool.PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
	})
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	err = app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[2].LocalAddress), sdk.Coins{sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	pools = app.MillionsKeeper.ListPools(ctx)
	pool, err = app.MillionsKeeper.GetPool(ctx, pools[2].PoolId)
	suite.Require().NoError(err)

	_, err = msgServer.WithdrawPayment(goCtx, &millionstypes.MsgWithdrawPayment{
		PoolId:           payments[1].PoolId,
		PaymentId:        payments[1].PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        "osmo11qggwr7vze9x7ustru9dctf6krhk2285lkf89g7",
	})
	// Error is triggered as toAddress does not match pool bech prefix and neither furya AccAddressFromBech32
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidDestinationAddress)

	_, err = msgServer.WithdrawPayment(goCtx, &millionstypes.MsgWithdrawPayment{
		PoolId:           payments[1].PoolId,
		PaymentId:        payments[1].PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        cosmosAccAddress,
	})
	suite.Require().NoError(err)

	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Equal(millionstypes.WithdrawalState_Pending, withdrawals[len(withdrawals)-1].State)
}

// TestMsgServer_WithdrawPaymentRetry runs withdrawal retry related tests
func (suite *KeeperTestSuite) TestMsgServer_WithdrawPaymentRetry() {
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

	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pool.PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_IbcTransfer,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pool.IcaPaymentAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	payments := app.MillionsKeeper.ListPayments(ctx)
	// Simulate transfer payment and delegate to native chain
	err = app.MillionsKeeper.TransferPaymentToRemoteZone(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	payments = app.MillionsKeeper.ListPayments(ctx)
	err = app.MillionsKeeper.DelegatePaymentOnRemoteZone(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().Error(err)

	payments = app.MillionsKeeper.ListPayments(ctx)

	app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
		PoolId:           payments[0].PoolId,
		PaymentId:        payments[0].PaymentId,
		State:            millionstypes.WithdrawalState_Pending,
		ErrorState:       millionstypes.WithdrawalState_Unspecified,
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(1_000_000))),
		UnbondingEndsAt:  &time.Time{},
	})

	payment, err := app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	// Remove payment to respect the flow
	app.MillionsKeeper.RemovePayment(ctx, &payment)

	withdrawals := app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 1)

	err = app.MillionsKeeper.AddEpochUnbonding(ctx, withdrawals[0], false)
	suite.Require().NoError(err)

	// Test ValidateWithdrawPaymentRetryBasic
	// Test withdrawal retry with invalid poolID
	// - Test unknown poolID
	_, err = msgServer.WithdrawPaymentRetry(goCtx, &millionstypes.MsgWithdrawPaymentRetry{
		PoolId:           uint64(0),
		WithdrawalId:     withdrawals[0].WithdrawalId,
		PaymentorAddress: suite.addrs[0].String(),
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)

	// - Test unknown withdrawalID
	_, err = msgServer.WithdrawPaymentRetry(goCtx, &millionstypes.MsgWithdrawPaymentRetry{
		PoolId:           withdrawals[0].PoolId,
		WithdrawalId:     uint64(0),
		PaymentorAddress: suite.addrs[0].String(),
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

	// Test if paymentorAddr msg is the same as pool withdrawal paymentor entity
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	_, err = msgServer.WithdrawPaymentRetry(goCtx, &millionstypes.MsgWithdrawPaymentRetry{
		PoolId:           withdrawals[0].PoolId,
		WithdrawalId:     withdrawals[0].PoolId,
		PaymentorAddress: "different-address",
	})
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidPaymentorAddress)

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
	_, err = app.StakingKeeper.CompleteUnbonding(ctx, sdk.MustAccAddressFromBech32(pool.IcaPaymentAddress), suite.valAddrs[0])
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
	_, err = msgServer.WithdrawPaymentRetry(goCtx, &millionstypes.MsgWithdrawPaymentRetry{
		PoolId:           withdrawals[0].PoolId,
		WithdrawalId:     withdrawals[0].WithdrawalId,
		PaymentorAddress: suite.addrs[0].String(),
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
