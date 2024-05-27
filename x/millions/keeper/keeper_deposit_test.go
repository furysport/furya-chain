package keeper_test

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	apptesting "github.com/furysport/furya-chain/v2/app/testing"
	millionstypes "github.com/furysport/furya-chain/v2/x/millions/types"
)

// TestCredit_IDsGeneration tests that the creditID is properly incremented for each new credit
func (suite *KeeperTestSuite) TestCredit_IDsGeneration() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)
	// Add 10 credits
	for i := 0; i < 10; i++ {
		poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
		creditID := app.MillionsKeeper.GetNextCreditIdAndIncrement(ctx)
		app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{PoolId: poolID}))

		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           poolID,
			CreditId:        creditID,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
		// Test that creditID is incremented
		suite.Require().Equal(uint64(i+1), creditID)
		// Test that we never override an existing entity
		panicF := func() {
			app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
				PoolId:           poolID,
				CreditId:        creditID,
				CreditorAddress: suite.addrs[0].String(),
				WinnerAddress:    suite.addrs[0].String(),
				State:            millionstypes.CreditState_IbcTransfer,
				Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
			})
		}
		suite.Require().Panics(panicF)
	}
}

// TestCredit_AddCredit tests the logic of adding a credit to the store
func (suite *KeeperTestSuite) TestCredit_AddCredit() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)

	creditsBefore := app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Len(creditsBefore, 0)
	credits := app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Len(credits, 0)

	// Track poolTVLBefore
	var poolTvlBefore math.Int
	// Track poolCreditorsCount
	var poolCreditorCountBefore uint64
	// Add 5 credits
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
			AvailablePrizePool: sdk.NewCoin(app.StakingKeeper.BondDenom(ctx), math.NewInt(1000)),
		}))
		// Retrieve the pool from the state
		pool, err := app.MillionsKeeper.GetPool(ctx, poolID)
		suite.Require().NoError(err)

		// Test poolTVLBefore
		poolTvlBefore = pool.TvlAmount
		suite.Require().Equal(int64(0), poolTvlBefore.Int64())

		// Test poolCreditorCount
		poolCreditorCountBefore = pool.CreditorsCount
		suite.Require().Equal(uint64(0), poolCreditorCountBefore)

		// Create a new credit and add it to the state
		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           pool.PoolId,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}

	// - Test that credits with unknown IDs have a new creditIDs reassigned
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	for i, credit := range credits {
		suite.Require().NotEqual(uint64(0), credit.CreditId)
		suite.Require().Equal(uint64(i+1), credit.CreditId)
	}

	// - Test the validate basics
	// -- Test one successful credit with all correct validate basics
	panicF := func() {
		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           credits[len(credits)-1].PoolId,
			CreditorAddress: suite.addrs[1].String(),
			WinnerAddress:    suite.addrs[1].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().NotPanics(panicF)
	// -- Test that the credit validation panics with an invalid pool ID
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           uint64(0),
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the credit validation panics with an invalid creditID
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	panicF = func() {
		credits[7].CreditId = 0
	}
	suite.Require().Panics(panicF)
	// -- Test that the credit validation panics with an invalid state
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           credits[len(credits)-1].PoolId,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_Unspecified,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the credit validation panics with an invalid CreditorAddress
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           credits[len(credits)-1].PoolId,
			CreditorAddress: "",
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the credit validation panics with an invalid WinnerAddress
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           credits[len(credits)-1].PoolId,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    "",
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)

	// - Test that we never override an existing entity (double credit)
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           credits[len(credits)-1].PoolId,
			CreditId:        credits[len(credits)-1].CreditId,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)

	// - Test that the pool credits and account credits are correctly updated
	creditsAccount := app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Equal(len(creditsBefore)+5, len(creditsAccount))
	creditsAccount = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[1])
	suite.Require().Equal(len(creditsBefore)+1, len(creditsAccount))
	credits = app.MillionsKeeper.ListCredits(ctx)
	suite.Require().Equal(len(creditsBefore)+6, len(credits))

	pools := app.MillionsKeeper.ListPools(ctx)

	for i, pool := range pools {
		// Test that the first 4 pools are updated
		if i <= 3 {
			creditsAccountPool := app.MillionsKeeper.ListAccountPoolCredits(ctx, suite.addrs[0], pool.PoolId)
			suite.Require().Equal(len(creditsBefore)+1, len(creditsAccountPool))

			creditsPool := app.MillionsKeeper.ListPoolCredits(ctx, pool.PoolId)
			suite.Require().Equal(len(creditsBefore)+1, len(creditsPool))
			// - Test that the TVL is added to the pool by the amount of the credit
			suite.Require().Equal(poolTvlBefore.Int64()+1_000_000, pool.TvlAmount.Int64())
			// - Test that the credit count is correctly incremented if it's a new creditor
			suite.Require().Equal(poolCreditorCountBefore+1, pool.CreditorsCount)

			// Test GetPoolCredit
			poolCredit, err := app.MillionsKeeper.GetPoolCredit(ctx, pool.PoolId, credits[i].CreditId)
			suite.Require().NoError(err)
			suite.Require().Equal(credits[i].Amount.Amount.Int64(), poolCredit.Amount.Amount.Int64())
			suite.Require().Equal(credits[i].PoolId, poolCredit.PoolId)

			// Test that the last pool is updated accordingly
		} else if i == len(pools)-1 {
			creditsPool := app.MillionsKeeper.ListPoolCredits(ctx, pool.PoolId)
			suite.Require().Equal(len(creditsBefore)+2, len(creditsPool))
			creditsAccountPool := app.MillionsKeeper.ListAccountPoolCredits(ctx, suite.addrs[1], pool.PoolId)
			suite.Require().Equal(len(creditsBefore)+1, len(creditsAccountPool))
			// - Test that the TVL is added to the pool by the amount of the credit for the last pool
			suite.Require().Equal(poolTvlBefore.Int64()+2_000_000, pool.TvlAmount.Int64())
			// - Test that the credit count is correctly incremented if it's a new creditor for the last pool
			suite.Require().Equal(poolCreditorCountBefore+2, pool.CreditorsCount)

			// Test GetPoolCredit
			poolCredit, err := app.MillionsKeeper.GetPoolCredit(ctx, pool.PoolId, credits[i].CreditId)
			suite.Require().NoError(err)
			suite.Require().Equal(credits[i].Amount.Amount.Int64(), poolCredit.Amount.Amount.Int64())
			suite.Require().Equal(credits[i].PoolId, poolCredit.PoolId)
		}
	}
}

// TestCredit_RemoveCredit tests the logic of removing a credit from the store
func (suite *KeeperTestSuite) TestCredit_RemoveCredit() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)

	creditsBefore := app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Len(creditsBefore, 0)
	credits := app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Len(credits, 0)

	// Track poolTVLBefore
	var poolTvlBefore math.Int
	// Track poolCreditorsCount
	var poolCreditorCountBefore uint64

	// Add 5 credits
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
			AvailablePrizePool: sdk.NewCoin(app.StakingKeeper.BondDenom(ctx), math.NewInt(1000)),
		}))
		// Retrieve the pool from the state
		pool, err := app.MillionsKeeper.GetPool(ctx, poolID)
		suite.Require().NoError(err)

		// Test poolTVLBefore
		poolTvlBefore = pool.TvlAmount
		suite.Require().Equal(int64(0), poolTvlBefore.Int64())

		// Test poolCreditorCountBefore
		poolCreditorCountBefore = pool.CreditorsCount
		suite.Require().Equal(uint64(0), poolCreditorCountBefore)

		// Create a new credit and add it to the state
		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           pool.PoolId,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}

	credits = app.MillionsKeeper.ListCredits(ctx)
	// - Test the validate basics
	// -- Test one successful credit with all correct validate basics
	panicF := func() {
		app.MillionsKeeper.RemoveCredit(ctx, &millionstypes.Credit{
			PoolId:           credits[len(credits)-1].PoolId,
			CreditId:        credits[len(credits)-1].PoolId,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().NotPanics(panicF)
	// -- Test that the credit validation panics with an invalid pool ID
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.RemoveCredit(ctx, &millionstypes.Credit{
			PoolId:           uint64(0),
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the credit validation panics with an invalid creditID
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	panicF = func() {
		credits[7].CreditId = 0
	}
	suite.Require().Panics(panicF)
	// -- Test that the credit validation panics with an invalid state
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.RemoveCredit(ctx, &millionstypes.Credit{
			PoolId:           credits[len(credits)-1].PoolId,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_Unspecified,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the credit validation panics with an invalid CreditorAddress
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.RemoveCredit(ctx, &millionstypes.Credit{
			PoolId:           credits[len(credits)-1].PoolId,
			CreditorAddress: "",
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the credit validation panics with an invalid WinnerAddress
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.RemoveCredit(ctx, &millionstypes.Credit{
			PoolId:           credits[len(credits)-1].PoolId,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    "",
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)

	// - Test that the store is delete for pool credits and account credits
	// List all pools
	pools := app.MillionsKeeper.ListPools(ctx)

	for i, pool := range pools {
		if i <= 3 {
			creditsAccountPool := app.MillionsKeeper.ListAccountPoolCredits(ctx, suite.addrs[0], pool.PoolId)
			suite.Require().Equal(len(creditsBefore)+1, len(creditsAccountPool))

			creditsPool := app.MillionsKeeper.ListPoolCredits(ctx, pool.PoolId)
			suite.Require().Equal(len(creditsBefore)+1, len(creditsPool))
			// - Test that the TVL is added to the pool by the amount of the credit
			suite.Require().Equal(poolTvlBefore.Int64()+1_000_000, pool.TvlAmount.Int64())
			// - Test that the credit count is correctly incremented if it's a new creditor
			suite.Require().Equal(poolCreditorCountBefore+1, pool.CreditorsCount)
			// Check that credit was removed from last pool
		} else if i == len(pools)-1 {
			creditsPool := app.MillionsKeeper.ListPoolCredits(ctx, pool.PoolId)
			suite.Require().Equal(len(creditsBefore), len(creditsPool))
			creditsAccountPool := app.MillionsKeeper.ListAccountPoolCredits(ctx, suite.addrs[0], pool.PoolId)
			suite.Require().Equal(len(creditsBefore), len(creditsAccountPool))
			// - Test that the TVL is added to the pool by the amount of the credit for the last pool
			suite.Require().Equal(poolTvlBefore.Int64(), pool.TvlAmount.Int64())
			// - Test that the credit count is correctly incremented if it's a new creditor for the last pool
			suite.Require().Equal(poolCreditorCountBefore, pool.CreditorsCount)

		}
	}

	for _, credit := range credits {
		// Get the pool before remove credit
		poolBefore, err := app.MillionsKeeper.GetPool(ctx, credit.PoolId)
		suite.Require().NoError(err)
		// Get the Account credit before remove credit
		addr := sdk.MustAccAddressFromBech32(credit.CreditorAddress)
		credits = app.MillionsKeeper.ListAccountCredits(ctx, addr)
		// Get the Account pool credit before remove credit
		creditsAccountPoolBefore := app.MillionsKeeper.ListAccountPoolCredits(ctx, addr, poolBefore.PoolId)
		// Get the pool credit before remove credit
		creditsPoolBefore := app.MillionsKeeper.ListPoolCredits(ctx, poolBefore.PoolId)

		// Test GetPoolCredit
		poolCredit, err := app.MillionsKeeper.GetPoolCredit(ctx, credit.PoolId, credit.CreditId)
		suite.Require().NoError(err)
		suite.Require().Equal(credit.Amount.Amount.Int64(), poolCredit.Amount.Amount.Int64())
		suite.Require().Equal(credit.PoolId, poolCredit.PoolId)

		// Remove the credit
		app.MillionsKeeper.RemoveCredit(ctx, &credit)

		// Decrement by 1 for each loop
		creditsAccount := app.MillionsKeeper.ListAccountCredits(ctx, addr)
		suite.Require().Equal(len(credits)-1, len(creditsAccount))

		// Decrement by 1 for each loop
		creditsAccountPool := app.MillionsKeeper.ListAccountPoolCredits(ctx, addr, credit.PoolId)
		suite.Require().Equal(len(creditsAccountPoolBefore)-1, len(creditsAccountPool))

		// Decrement by 1 for each loop
		creditsPool := app.MillionsKeeper.ListPoolCredits(ctx, credit.PoolId)
		suite.Require().Equal(len(creditsPoolBefore)-1, len(creditsPool))
		// Get the pool
		pool, err := app.MillionsKeeper.GetPool(ctx, credit.PoolId)
		suite.Require().NoError(err)
		// Test that the pool tvl is deducted by the credit amount
		suite.Require().Equal(poolBefore.TvlAmount.Int64()-credit.Amount.Amount.Int64(), pool.TvlAmount.Int64())
		// - Test that the pool count is correctly decremented for each removed credit
		poolBefore.CreditorsCount--
		suite.Require().Equal(poolBefore.CreditorsCount, pool.CreditorsCount)
	}

	creditsAll := app.MillionsKeeper.ListCredits(ctx)
	suite.Require().Equal(len(creditsBefore), len(creditsAll))
}

// TestCredit_UpdateCreditStatus tests the logic of updating a credit from the store
func (suite *KeeperTestSuite) TestCredit_UpdateCreditStatus() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)

	// Add 5 credits
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
			AvailablePrizePool: sdk.NewCoin(app.StakingKeeper.BondDenom(ctx), math.NewInt(1000)),
		}))
		// Retrieve the pool from the state
		pool, err := app.MillionsKeeper.GetPool(ctx, poolID)
		suite.Require().NoError(err)

		// Create a new credit and add it to the state
		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           pool.PoolId,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}

	// - Test if the credit exist
	// Retrived credits
	credits := app.MillionsKeeper.ListCredits(ctx)
	// Retrieve the pool from the state
	pools := app.MillionsKeeper.ListPools(ctx)
	for i, pool := range pools {
		app.MillionsKeeper.UpdateCreditStatus(ctx, pool.PoolId, credits[i].CreditId, millionstypes.CreditState_IcaDelegate, false)
		credit, err := app.MillionsKeeper.GetPoolCredit(ctx, pool.PoolId, credits[i].CreditId)
		suite.Require().NoError(err)
		suite.Require().NotNil(credit, "Not nil")
		suite.Require().Equal(millionstypes.CreditState_IcaDelegate, credit.State)
		suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)
	}
	credits = app.MillionsKeeper.ListCredits(ctx)
	// Trigger panic on wrong poolID
	panicF := func() {
		_, err := app.MillionsKeeper.GetPoolCredit(ctx, credits[10].PoolId, uint64(5))
		suite.Require().NoError(err)
	}
	suite.Require().Panics(panicF)
	// Trigger panic on wrong creditID
	panicF = func() {
		_, err := app.MillionsKeeper.GetPoolCredit(ctx, uint64(5), credits[10].PoolId)
		suite.Require().NoError(err)
	}
	suite.Require().Panics(panicF)

	// - Test Error Management
	credits = app.MillionsKeeper.ListCredits(ctx)
	// Retrieve the pool from the state
	pools = app.MillionsKeeper.ListPools(ctx)
	for i, pool := range pools {
		app.MillionsKeeper.UpdateCreditStatus(ctx, pool.PoolId, credits[i].CreditId, millionstypes.CreditState_IcaDelegate, true)
		credit, err := app.MillionsKeeper.GetPoolCredit(ctx, pool.PoolId, credits[i].CreditId)
		suite.Require().NoError(err)
		suite.Require().NotNil(credit, "Not nil")
		suite.Require().Equal(millionstypes.CreditState_Failure, credit.State)
		suite.Require().Equal(millionstypes.CreditState_IcaDelegate, credit.ErrorState)
	}

	// - Test that the account credit was updated
	// - Test that the pool credit was updated
	pools = app.MillionsKeeper.ListPools(ctx)
	for i, pool := range pools {
		creditsAccounts := app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
		suite.Require().Equal(ctx.BlockTime(), creditsAccounts[i].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), creditsAccounts[i].UpdatedAtHeight)

		creditsPoolAccount := app.MillionsKeeper.ListAccountPoolCredits(ctx, suite.addrs[0], pool.PoolId)
		suite.Require().Equal(ctx.BlockTime(), creditsPoolAccount[0].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), creditsPoolAccount[0].UpdatedAtHeight)

		creditsPool := app.MillionsKeeper.ListPoolCredits(ctx, pool.PoolId)
		suite.Require().Equal(ctx.BlockTime(), creditsPool[0].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), creditsPool[0].UpdatedAtHeight)
	}
}

// TestCredit_EditCredit tests the logic of editing a credit's winnerAddr and sponsor
func (suite *KeeperTestSuite) TestCredit_EditCredit() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)

	// Add 5 credits
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
			AvailablePrizePool: sdk.NewCoin(app.StakingKeeper.BondDenom(ctx), math.NewInt(1000)),
		}))
		// Retrieve the pool from the state
		pool, err := app.MillionsKeeper.GetPool(ctx, poolID)
		suite.Require().NoError(err)

		// Create a new credit and add it to the state
		app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
			PoolId:           pool.PoolId,
			CreditorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.CreditState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}

	// Test Error Management
	// - Trigger err on wrong poolID
	_, err := app.MillionsKeeper.GetPoolCredit(ctx, uint64(10), uint64(5))
	suite.Require().Error(err)
	// - Trigger err on wrong creditID
	_, err = app.MillionsKeeper.GetPoolCredit(ctx, uint64(5), uint64(10))
	suite.Require().Error(err)

	credits := app.MillionsKeeper.ListCredits(ctx)
	// Retrieve the pool from the state
	pools := app.MillionsKeeper.ListPools(ctx)
	for i, pool := range pools {
		err = app.MillionsKeeper.EditCredit(ctx, pool.PoolId, credits[i].CreditId, suite.addrs[1], true)
		suite.Require().NoError(err)
		credit, err := app.MillionsKeeper.GetPoolCredit(ctx, pool.PoolId, credits[i].CreditId)
		suite.Require().NoError(err)
		pool, err := app.MillionsKeeper.GetPool(ctx, credit.PoolId)
		suite.Require().NoError(err)
		suite.Require().NotNil(credit, "Not nil")
		suite.Require().Equal(suite.addrs[1].String(), credit.WinnerAddress)
		suite.Require().Equal(true, credit.IsSponsor)
		suite.Require().Equal(int64(1_000_000), pool.SponsorshipAmount.Int64())
	}

	// pool SponsorshipAmount should be sub in case user does not want to sponsor anymore
	pool, err := app.MillionsKeeper.GetPool(ctx, credits[0].PoolId)
	suite.Require().NoError(err)
	err = app.MillionsKeeper.EditCredit(ctx, pool.PoolId, credits[0].CreditId, suite.addrs[1], false)
	suite.Require().NoError(err)
	pool, err = app.MillionsKeeper.GetPool(ctx, credits[0].PoolId)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(0), pool.SponsorshipAmount.Int64())
}

// TestCredit_TransferCredit tests the full flow from the transfer till the delegation point
func (suite *KeeperTestSuite) TestCredit_TransferCredit() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
	drawDelta1 := 1 * time.Hour
	uatomAddresses := apptesting.AddTestAddrsWithDenom(app, ctx, 7, sdk.NewInt(1_000_0000_000), "uatom")

	// Remote pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:              poolID,
		Bech32PrefixValAddr: "cosmosvaloper",
		ChainId:             "cosmos",
		Denom:               "uatom",
		NativeDenom:         "uatom",
		ConnectionId:        "connection-id",
		TransferChannelId:   "transferChannel-id",
		Validators: []millionstypes.PoolValidator{{
			OperatorAddress: cosmosPoolValidator,
			BondedAmount:    sdk.NewInt(1_000_000),
			IsEnabled:       true,
		}},
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
		AvailablePrizePool: sdk.NewCoin("uatom", math.NewInt(1000)),
	}))

	pools := app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 1)

	// - Test the full credit process from the transfer till the delegation point
	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[0].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[0].Validators[0].BondedAmount)

	// Create a new credit and add it to the state
	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pools[0].PoolId,
		CreditorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.CreditState_IbcTransfer,
		Amount:           sdk.NewCoin("uatom", sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[0].LocalAddress), sdk.Coins{sdk.NewCoin(pools[0].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	credits := app.MillionsKeeper.ListAccountCredits(ctx, uatomAddresses[0])

	// Test that the pool is not local
	suite.Require().Equal(false, pools[0].IsLocalZone(ctx))
	credit, err := app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().NotNil(credit, "Not nil")
	suite.Require().Equal(millionstypes.CreditState_IbcTransfer, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)

	credits = app.MillionsKeeper.ListAccountCredits(ctx, uatomAddresses[0])
	suite.Require().Len(credits, 1)

	// Simulate failed ackResponse AckResponseStatus_FAILURE
	err = app.MillionsKeeper.OnTransferCreditToRemoteZoneCompleted(ctx, credits[0].GetPoolId(), credits[0].GetCreditId(), true)
	suite.Require().NoError(err)

	// List credits to get latest status
	credits = app.MillionsKeeper.ListAccountCredits(ctx, uatomAddresses[0])
	suite.Require().Len(credits, 1)
	// Failed Transfer should set the errorState to CreditState_IbcTransfer and state to CreditState_Failure
	suite.Require().Equal(millionstypes.CreditState_Failure, credits[0].State)
	suite.Require().Equal(millionstypes.CreditState_IbcTransfer, credits[0].ErrorState)

	// Update status to simulate that the state is initially on CreditState_IbcTransfer
	app.MillionsKeeper.UpdateCreditStatus(ctx, credits[0].PoolId, credits[0].CreditId, millionstypes.CreditState_IbcTransfer, false)

	// Simulate succesful ackResponse AckResponseStatus_SUCCESS
	err = app.MillionsKeeper.OnTransferCreditToRemoteZoneCompleted(ctx, credits[0].GetPoolId(), credits[0].GetCreditId(), false)
	suite.Require().NoError(err)

	// List credits to get latest state
	credits = app.MillionsKeeper.ListAccountCredits(ctx, uatomAddresses[0])
	suite.Require().Len(credits, 1)
	// The credit will reached the sequence, err := k.BroadcastICAMessages an fail on OnDelegateCreditOnRemoteZoneCompleted as we don't BroadcastICAMessages in our test
	suite.Require().Equal(millionstypes.CreditState_Failure, credits[0].State)
	suite.Require().Equal(millionstypes.CreditState_IcaDelegate, credits[0].ErrorState)

	// Test local pool
	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)
	// local pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:      app.MillionsKeeper.GetNextPoolID(ctx),
		Denom:       "ufury",
		NativeDenom: "ufury",
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
		AvailablePrizePool: sdk.NewCoin(app.StakingKeeper.BondDenom(ctx), math.NewInt(1000)),
	}))
	pools = app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 2)

	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[1].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[1].Validators[0].BondedAmount)

	// Create a new credit and add it to the state
	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pools[1].PoolId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.CreditState_IbcTransfer,
		Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[1].IcaCreditAddress), sdk.Coins{sdk.NewCoin(pools[1].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Len(credits, 1)

	// Test that the pool is local
	suite.Require().Equal(true, pools[1].IsLocalZone(ctx))
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().NotNil(credit, "Not nil")
	suite.Require().Equal(millionstypes.CreditState_IbcTransfer, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)

	// Trigger the Transfer credit to native chain
	err = app.MillionsKeeper.TransferCreditToRemoteZone(ctx, credit.PoolId, credit.CreditId)
	suite.Require().NoError(err)
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.CreditState_Success, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)
}

// TestCredit_DelegateCredit tests the delegate credit from the transfer point till the final delegation
func (suite *KeeperTestSuite) TestCredit_DelegateCredit() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
	drawDelta1 := 1 * time.Hour
	uatomAddresses := apptesting.AddTestAddrsWithDenom(app, ctx, 7, sdk.NewInt(1_000_0000_000), "uatom")

	// Remote pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:              poolID,
		Bech32PrefixValAddr: "cosmosvaloper",
		ChainId:             "cosmos",
		Denom:               "uatom",
		NativeDenom:         "uatom",
		ConnectionId:        "connection-id",
		TransferChannelId:   "transferChannel-id",
		Validators: []millionstypes.PoolValidator{{
			OperatorAddress: cosmosPoolValidator,
			BondedAmount:    sdk.NewInt(1_000_000),
			IsEnabled:       true,
		}},
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
		AvailablePrizePool: sdk.NewCoin("uatom", math.NewInt(1000)),
	}))

	pools := app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 1)

	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[0].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[0].Validators[0].BondedAmount)

	// Create a new credit and add it to the state
	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pools[0].PoolId,
		CreditorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.CreditState_IcaDelegate,
		Amount:           sdk.NewCoin("uatom", sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[0].LocalAddress), sdk.Coins{sdk.NewCoin(pools[0].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	credits := app.MillionsKeeper.ListAccountCredits(ctx, uatomAddresses[0])
	suite.Require().Len(credits, 1)

	// Test that the pool is not local
	suite.Require().Equal(false, pools[0].IsLocalZone(ctx))
	credit, err := app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().NotNil(credit, "Not nil")
	suite.Require().Equal(millionstypes.CreditState_IcaDelegate, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)

	// Simulate a failed DelegateCreditOnRemoteZone
	err = app.MillionsKeeper.DelegateCreditOnRemoteZone(ctx, credit.PoolId, credit.CreditId)
	suite.Require().NoError(err)

	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	// The credit will reached the sequence, err := k.BroadcastICAMessages and fail on OnDelegateCreditOnRemoteZoneCompleted as we don't BroadcastICAMessages in our test
	suite.Require().Equal(millionstypes.CreditState_Failure, credit.State)
	suite.Require().Equal(millionstypes.CreditState_IcaDelegate, credit.ErrorState)

	// Update status to simulate that the state is initially on CreditState_IcaDelegate
	app.MillionsKeeper.UpdateCreditStatus(ctx, credit.PoolId, credit.CreditId, millionstypes.CreditState_IcaDelegate, false)

	splits := []*millionstypes.SplitDelegation{{ValidatorAddress: cosmosPoolValidator, Amount: sdk.NewInt(int64(1_000_000))}}

	// Now trigger a successfull Delegation to simulate icacallbackstypes.AckResponseStatus_SUCCESS
	err = app.MillionsKeeper.OnDelegateCreditOnRemoteZoneCompleted(ctx, credit.PoolId, credit.CreditId, splits, false)
	suite.Require().NoError(err)
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, credit.PoolId, credit.CreditId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.CreditState_Success, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)

	// Test local pool
	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)
	// local pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:      app.MillionsKeeper.GetNextPoolID(ctx),
		Denom:       "ufury",
		NativeDenom: "ufury",
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
		AvailablePrizePool: sdk.NewCoin(denom, math.NewInt(1000)),
	}))
	pools = app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 2)

	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[1].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[1].Validators[0].BondedAmount)

	// Create a new credit and add it to the state
	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pools[1].PoolId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.CreditState_IcaDelegate,
		Amount:           sdk.NewCoin(app.StakingKeeper.BondDenom(ctx), sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[1].IcaCreditAddress), sdk.Coins{sdk.NewCoin(pools[1].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Len(credits, 1)

	pools = app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 2)

	// Test that the pool is local
	suite.Require().Equal(true, pools[1].IsLocalZone(ctx))
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().NotNil(credit, "Not nil")
	suite.Require().Equal(millionstypes.CreditState_IcaDelegate, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)

	// As it's local pool trigger a DelegateCreditOnRemoteZone for the new credit
	err = app.MillionsKeeper.DelegateCreditOnRemoteZone(ctx, credit.PoolId, credit.CreditId)
	suite.Require().NoError(err)

	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.CreditState_Success, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)
}

// TestCredit_BalanceCredit tests the balance in case of success or failed credit
func (suite *KeeperTestSuite) TestCredit_BalanceCredit() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	drawDelta1 := 1 * time.Hour
	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
	uatomAddresses := apptesting.AddTestAddrsWithDenom(app, ctx, 7, sdk.NewInt(1_000_0000_000), "uatom")

	// Remote pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:              poolID,
		Bech32PrefixValAddr: "cosmosvaloper",
		ChainId:             "cosmos",
		Denom:               "uatom",
		NativeDenom:         "uatom",
		ConnectionId:        "connection-id",
		TransferChannelId:   "transferChannel-id",
		Validators: []millionstypes.PoolValidator{{
			OperatorAddress: cosmosPoolValidator,
			BondedAmount:    sdk.NewInt(1_000_000),
			IsEnabled:       true,
		}},
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
		AvailablePrizePool: sdk.NewCoin("uatom", math.NewInt(1000)),
	}))

	pools := app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 1)

	// Initialize balance before
	balanceBefore := app.BankKeeper.GetBalance(ctx, uatomAddresses[0], "uatom")
	suite.Require().Equal(int64(1_000_000_000_0), balanceBefore.Amount.Int64())
	// Initialize balance module account
	balanceModuleAccBefore := app.BankKeeper.GetBalance(ctx, suite.moduleAddrs[7], "uatom")
	suite.Require().Equal(int64(0), balanceModuleAccBefore.Amount.Int64())

	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[0].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[0].Validators[0].BondedAmount)

	// Create a new credit and add it to the state
	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pools[0].PoolId,
		CreditorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.CreditState_IbcTransfer,
		Amount:           sdk.NewCoin("uatom", sdk.NewInt(1_000_000)),
	})

	err := app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[0].LocalAddress), sdk.Coins{sdk.NewCoin(pools[0].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	// Balance should have decreased by one credit
	balance := app.BankKeeper.GetBalance(ctx, uatomAddresses[0], "uatom")
	suite.Require().Equal(balanceBefore.Amount.Int64()-1_000_000, balance.Amount.Int64())
	// Module Acc should have increased by one credit
	balanceModuleAcc := app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[0].LocalAddress), "uatom")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64()+1_000_000, balanceModuleAcc.Amount.Int64())

	credits := app.MillionsKeeper.ListAccountCredits(ctx, uatomAddresses[0])
	suite.Require().Len(credits, 1)

	app.MillionsKeeper.UpdateCreditStatus(ctx, credits[0].PoolId, credits[0].CreditId, millionstypes.CreditState_IcaDelegate, false)

	suite.Require().Equal(false, pools[0].IsLocalZone(ctx))
	credit, err := app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().NotNil(credit, "Not nil")
	suite.Require().Equal(millionstypes.CreditState_IcaDelegate, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)

	splits := []*millionstypes.SplitDelegation{{ValidatorAddress: cosmosPoolValidator, Amount: sdk.NewInt(int64(1_000_000))}}

	// Now trigger a successfull Delegation to simulate icacallbackstypes.AckResponseStatus_SUCCESS
	err = app.MillionsKeeper.OnDelegateCreditOnRemoteZoneCompleted(ctx, credit.PoolId, credit.CreditId, splits, false)
	suite.Require().NoError(err)

	// For remote pools we don't reach the success as k.BroadcastICAMessages fails on OnDelegateCreditOnRemoteZoneCompleted

	// Balance should remain unchanged
	balance = app.BankKeeper.GetBalance(ctx, uatomAddresses[0], "uatom")
	suite.Require().Equal(balanceBefore.Amount.Int64()-1_000_000, balance.Amount.Int64())
	// Module Account should have the credit amount
	balanceModuleAcc = app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[0].LocalAddress), "uatom")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64()+1_000_000, balanceModuleAcc.Amount.Int64())

	// There should no delegation shares
	delegationAmount := app.StakingKeeper.GetDelegatorDelegations(ctx, sdk.MustAccAddressFromBech32(pools[0].LocalAddress), 10)
	suite.Require().Len(delegationAmount, 0)

	// Test local pool
	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)
	// local pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:      app.MillionsKeeper.GetNextPoolID(ctx),
		Denom:       "ufury",
		NativeDenom: "ufury",
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
		AvailablePrizePool: sdk.NewCoin(app.StakingKeeper.BondDenom(ctx), math.NewInt(1000)),
	}))
	pools = app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 2)

	// Initialize balance before
	balanceBefore = app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(int64(1_000_000_000_0), balanceBefore.Amount.Int64())
	// Initialize balance module account
	balanceModuleAccBefore = app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaCreditAddress), "ufury")
	suite.Require().Equal(int64(0), balanceModuleAccBefore.Amount.Int64())

	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[1].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[1].Validators[0].BondedAmount)

	// Create a new credit and add it to the state
	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pools[1].PoolId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.CreditState_IbcTransfer,
		Amount:           sdk.NewCoin(denom, sdk.NewInt(2_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[1].IcaCreditAddress), sdk.Coins{sdk.NewCoin(pools[1].Denom, sdk.NewInt(2_000_000))})
	suite.Require().NoError(err)
	credits = app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Len(credits, 1)

	// Balance should have decreased by two credits
	balance = app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(balanceBefore.Amount.Int64()-2_000_000, balance.Amount.Int64())
	// Module Acc should have increased by two credits
	balanceModuleAcc = app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaCreditAddress), "ufury")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64()+2_000_000, balanceModuleAcc.Amount.Int64())

	// There should be no delegation shares yet
	delegationAmount = app.StakingKeeper.GetDelegatorDelegations(ctx, suite.moduleAddrs[0], 10)
	suite.Require().Len(delegationAmount, 0)

	suite.Require().Equal(true, pools[1].IsLocalZone(ctx))
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().NotNil(credit, "Not nil")
	suite.Require().Equal(millionstypes.CreditState_IbcTransfer, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)

	err = app.MillionsKeeper.OnTransferCreditToRemoteZoneCompleted(ctx, credit.PoolId, credit.CreditId, true)
	suite.Require().NoError(err)
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.CreditState_Failure, credit.State)
	suite.Require().Equal(millionstypes.CreditState_IbcTransfer, credit.ErrorState)

	// Balance should be unchanged
	balance = app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(balanceBefore.Amount.Int64()-2_000_000, balance.Amount.Int64())
	// Module Acc should have still the credited amount
	balanceModuleAcc = app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaCreditAddress), "ufury")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64()+2_000_000, balanceModuleAcc.Amount.Int64())

	// There should be no delegation shares yet
	delegationAmount = app.StakingKeeper.GetDelegatorDelegations(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaCreditAddress), 10)
	suite.Require().Len(delegationAmount, 0)

	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	// Update the status to trigger successful CreditToNativeChain
	app.MillionsKeeper.UpdateCreditStatus(ctx, credit.PoolId, credit.CreditId, millionstypes.CreditState_IbcTransfer, false)

	err = app.MillionsKeeper.TransferCreditToRemoteZone(ctx, credit.PoolId, credit.CreditId)
	suite.Require().NoError(err)
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.CreditState_Success, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)

	// Balance should be unchanged
	balance = app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(balanceBefore.Amount.Int64()-2_000_000, balance.Amount.Int64())
	// Module Acc should have delegated the credit
	balanceModuleAcc = app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaCreditAddress), "ufury")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64(), balanceModuleAcc.Amount.Int64())

	// The delegation amount should reflect the shares delegated - client precision
	delegationAmount = app.StakingKeeper.GetDelegatorDelegations(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaCreditAddress), 10)
	creditToShare := sdk.NewDecFromInt(sdk.NewInt(credit.Amount.Amount.Int64() / 1_000_000))
	suite.Require().Equal(creditToShare, delegationAmount[0].Shares)
}

// TestCredit_FullCreditProcess tests the complete logic for credits for a local pool
func (suite *KeeperTestSuite) TestCredit_FullCreditProcess() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	drawDelta1 := 1 * time.Hour
	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)

	// - Test the full credit process for a local pool
	denom := app.StakingKeeper.BondDenom(ctx)
	// local pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:      poolID,
		Denom:       "ufury",
		NativeDenom: "ufury",
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
		AvailablePrizePool: sdk.NewCoin(app.StakingKeeper.BondDenom(ctx), math.NewInt(1000)),
	}))

	pools := app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 1)

	// Initialize balance before
	balanceBefore := app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(int64(1_000_000_000_0), balanceBefore.Amount.Int64())
	// Initialize balance module account
	balanceModuleAccBefore := app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[0].IcaCreditAddress), "ufury")
	suite.Require().Equal(int64(0), balanceModuleAccBefore.Amount.Int64())

	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[0].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[0].Validators[0].BondedAmount)

	// Create a new credit and add it to the state
	app.MillionsKeeper.AddCredit(ctx, &millionstypes.Credit{
		PoolId:           pools[0].PoolId,
		CreditorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.CreditState_IbcTransfer,
		Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[0].IcaCreditAddress), sdk.Coins{sdk.NewCoin(pools[0].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	credits := app.MillionsKeeper.ListAccountCredits(ctx, suite.addrs[0])
	suite.Require().Len(credits, 1)

	// Balance should have decreased by one credit
	balance := app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(balanceBefore.Amount.Int64()-1_000_000, balance.Amount.Int64())
	// Module Acc should have increased by one credit
	balanceModuleAcc := app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[0].IcaCreditAddress), "ufury")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64()+1_000_000, balanceModuleAcc.Amount.Int64())

	// There should no delegation shares
	delegationAmount := app.StakingKeeper.GetDelegatorDelegations(ctx, sdk.MustAccAddressFromBech32(pools[0].IcaCreditAddress), 10)
	suite.Require().Len(delegationAmount, 0)

	// Test if it's a local pool
	suite.Require().Equal(true, pools[0].IsLocalZone(ctx))
	credit, err := app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().NotNil(credit, "Not nil")
	suite.Require().Equal(millionstypes.CreditState_IbcTransfer, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)

	err = app.MillionsKeeper.TransferCreditToRemoteZone(ctx, credit.PoolId, credit.CreditId)
	suite.Require().NoError(err)
	credit, err = app.MillionsKeeper.GetPoolCredit(ctx, credits[0].PoolId, credits[0].CreditId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.CreditState_Success, credit.State)
	suite.Require().Equal(millionstypes.CreditState_Unspecified, credit.ErrorState)

	// Balance should be unchanged
	balance = app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(balanceBefore.Amount.Int64()-1_000_000, balance.Amount.Int64())
	// Module Acc should have delegated the credit
	balanceModuleAcc = app.BankKeeper.GetBalance(ctx, suite.moduleAddrs[0], "ufury")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64(), balanceModuleAcc.Amount.Int64())

	// The delegation amount should reflect the shares delegated - client precision
	delegationAmount = app.StakingKeeper.GetDelegatorDelegations(ctx, sdk.MustAccAddressFromBech32(pools[0].IcaCreditAddress), 10)
	creditToShare := sdk.NewDecFromInt(sdk.NewInt(credit.Amount.Amount.Int64() / 1_000_000))
	suite.Require().Equal(creditToShare, delegationAmount[0].Shares)
}
