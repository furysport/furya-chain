package keeper_test

import (
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	apptesting "github.com/furysport/furya-chain/v2/app/testing"
	millionstypes "github.com/furysport/furya-chain/v2/x/millions/types"
)

// TestPayment_IDsGeneration tests that the paymentID is properly incremented for each new payment
func (suite *KeeperTestSuite) TestPayment_IDsGeneration() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)
	// Add 10 payments
	for i := 0; i < 10; i++ {
		poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
		paymentID := app.MillionsKeeper.GetNextPaymentIdAndIncrement(ctx)
		app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{PoolId: poolID}))

		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           poolID,
			PaymentId:        paymentID,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
		// Test that paymentID is incremented
		suite.Require().Equal(uint64(i+1), paymentID)
		// Test that we never override an existing entity
		panicF := func() {
			app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
				PoolId:           poolID,
				PaymentId:        paymentID,
				PaymentorAddress: suite.addrs[0].String(),
				WinnerAddress:    suite.addrs[0].String(),
				State:            millionstypes.PaymentState_IbcTransfer,
				Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
			})
		}
		suite.Require().Panics(panicF)
	}
}

// TestPayment_AddPayment tests the logic of adding a payment to the store
func (suite *KeeperTestSuite) TestPayment_AddPayment() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)

	paymentsBefore := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(paymentsBefore, 0)
	payments := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 0)

	// Track poolTVLBefore
	var poolTvlBefore math.Int
	// Track poolPaymentorsCount
	var poolPaymentorCountBefore uint64
	// Add 5 payments
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

		// Test poolPaymentorCount
		poolPaymentorCountBefore = pool.PaymentorsCount
		suite.Require().Equal(uint64(0), poolPaymentorCountBefore)

		// Create a new payment and add it to the state
		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           pool.PoolId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}

	// - Test that payments with unknown IDs have a new paymentIDs reassigned
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	for i, payment := range payments {
		suite.Require().NotEqual(uint64(0), payment.PaymentId)
		suite.Require().Equal(uint64(i+1), payment.PaymentId)
	}

	// - Test the validate basics
	// -- Test one successful payment with all correct validate basics
	panicF := func() {
		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentorAddress: suite.addrs[1].String(),
			WinnerAddress:    suite.addrs[1].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().NotPanics(panicF)
	// -- Test that the payment validation panics with an invalid pool ID
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           uint64(0),
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the payment validation panics with an invalid paymentID
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		payments[7].PaymentId = 0
	}
	suite.Require().Panics(panicF)
	// -- Test that the payment validation panics with an invalid state
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_Unspecified,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the payment validation panics with an invalid PaymentorAddress
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentorAddress: "",
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the payment validation panics with an invalid WinnerAddress
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    "",
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)

	// - Test that we never override an existing entity (double payment)
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentId:        payments[len(payments)-1].PaymentId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)

	// - Test that the pool payments and account payments are correctly updated
	paymentsAccount := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Equal(len(paymentsBefore)+5, len(paymentsAccount))
	paymentsAccount = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[1])
	suite.Require().Equal(len(paymentsBefore)+1, len(paymentsAccount))
	payments = app.MillionsKeeper.ListPayments(ctx)
	suite.Require().Equal(len(paymentsBefore)+6, len(payments))

	pools := app.MillionsKeeper.ListPools(ctx)

	for i, pool := range pools {
		// Test that the first 4 pools are updated
		if i <= 3 {
			paymentsAccountPool := app.MillionsKeeper.ListAccountPoolPayments(ctx, suite.addrs[0], pool.PoolId)
			suite.Require().Equal(len(paymentsBefore)+1, len(paymentsAccountPool))

			paymentsPool := app.MillionsKeeper.ListPoolPayments(ctx, pool.PoolId)
			suite.Require().Equal(len(paymentsBefore)+1, len(paymentsPool))
			// - Test that the TVL is added to the pool by the amount of the payment
			suite.Require().Equal(poolTvlBefore.Int64()+1_000_000, pool.TvlAmount.Int64())
			// - Test that the payment count is correctly incremented if it's a new paymentor
			suite.Require().Equal(poolPaymentorCountBefore+1, pool.PaymentorsCount)

			// Test GetPoolPayment
			poolPayment, err := app.MillionsKeeper.GetPoolPayment(ctx, pool.PoolId, payments[i].PaymentId)
			suite.Require().NoError(err)
			suite.Require().Equal(payments[i].Amount.Amount.Int64(), poolPayment.Amount.Amount.Int64())
			suite.Require().Equal(payments[i].PoolId, poolPayment.PoolId)

			// Test that the last pool is updated accordingly
		} else if i == len(pools)-1 {
			paymentsPool := app.MillionsKeeper.ListPoolPayments(ctx, pool.PoolId)
			suite.Require().Equal(len(paymentsBefore)+2, len(paymentsPool))
			paymentsAccountPool := app.MillionsKeeper.ListAccountPoolPayments(ctx, suite.addrs[1], pool.PoolId)
			suite.Require().Equal(len(paymentsBefore)+1, len(paymentsAccountPool))
			// - Test that the TVL is added to the pool by the amount of the payment for the last pool
			suite.Require().Equal(poolTvlBefore.Int64()+2_000_000, pool.TvlAmount.Int64())
			// - Test that the payment count is correctly incremented if it's a new paymentor for the last pool
			suite.Require().Equal(poolPaymentorCountBefore+2, pool.PaymentorsCount)

			// Test GetPoolPayment
			poolPayment, err := app.MillionsKeeper.GetPoolPayment(ctx, pool.PoolId, payments[i].PaymentId)
			suite.Require().NoError(err)
			suite.Require().Equal(payments[i].Amount.Amount.Int64(), poolPayment.Amount.Amount.Int64())
			suite.Require().Equal(payments[i].PoolId, poolPayment.PoolId)
		}
	}
}

// TestPayment_RemovePayment tests the logic of removing a payment from the store
func (suite *KeeperTestSuite) TestPayment_RemovePayment() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)

	paymentsBefore := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(paymentsBefore, 0)
	payments := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 0)

	// Track poolTVLBefore
	var poolTvlBefore math.Int
	// Track poolPaymentorsCount
	var poolPaymentorCountBefore uint64

	// Add 5 payments
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

		// Test poolPaymentorCountBefore
		poolPaymentorCountBefore = pool.PaymentorsCount
		suite.Require().Equal(uint64(0), poolPaymentorCountBefore)

		// Create a new payment and add it to the state
		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           pool.PoolId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}

	payments = app.MillionsKeeper.ListPayments(ctx)
	// - Test the validate basics
	// -- Test one successful payment with all correct validate basics
	panicF := func() {
		app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentId:        payments[len(payments)-1].PoolId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().NotPanics(panicF)
	// -- Test that the payment validation panics with an invalid pool ID
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
			PoolId:           uint64(0),
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the payment validation panics with an invalid paymentID
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		payments[7].PaymentId = 0
	}
	suite.Require().Panics(panicF)
	// -- Test that the payment validation panics with an invalid state
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_Unspecified,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the payment validation panics with an invalid PaymentorAddress
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentorAddress: "",
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the payment validation panics with an invalid WinnerAddress
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    "",
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)

	// - Test that the store is delete for pool payments and account payments
	// List all pools
	pools := app.MillionsKeeper.ListPools(ctx)

	for i, pool := range pools {
		if i <= 3 {
			paymentsAccountPool := app.MillionsKeeper.ListAccountPoolPayments(ctx, suite.addrs[0], pool.PoolId)
			suite.Require().Equal(len(paymentsBefore)+1, len(paymentsAccountPool))

			paymentsPool := app.MillionsKeeper.ListPoolPayments(ctx, pool.PoolId)
			suite.Require().Equal(len(paymentsBefore)+1, len(paymentsPool))
			// - Test that the TVL is added to the pool by the amount of the payment
			suite.Require().Equal(poolTvlBefore.Int64()+1_000_000, pool.TvlAmount.Int64())
			// - Test that the payment count is correctly incremented if it's a new paymentor
			suite.Require().Equal(poolPaymentorCountBefore+1, pool.PaymentorsCount)
			// Check that payment was removed from last pool
		} else if i == len(pools)-1 {
			paymentsPool := app.MillionsKeeper.ListPoolPayments(ctx, pool.PoolId)
			suite.Require().Equal(len(paymentsBefore), len(paymentsPool))
			paymentsAccountPool := app.MillionsKeeper.ListAccountPoolPayments(ctx, suite.addrs[0], pool.PoolId)
			suite.Require().Equal(len(paymentsBefore), len(paymentsAccountPool))
			// - Test that the TVL is added to the pool by the amount of the payment for the last pool
			suite.Require().Equal(poolTvlBefore.Int64(), pool.TvlAmount.Int64())
			// - Test that the payment count is correctly incremented if it's a new paymentor for the last pool
			suite.Require().Equal(poolPaymentorCountBefore, pool.PaymentorsCount)

		}
	}

	for _, payment := range payments {
		// Get the pool before remove payment
		poolBefore, err := app.MillionsKeeper.GetPool(ctx, payment.PoolId)
		suite.Require().NoError(err)
		// Get the Account payment before remove payment
		addr := sdk.MustAccAddressFromBech32(payment.PaymentorAddress)
		payments = app.MillionsKeeper.ListAccountPayments(ctx, addr)
		// Get the Account pool payment before remove payment
		paymentsAccountPoolBefore := app.MillionsKeeper.ListAccountPoolPayments(ctx, addr, poolBefore.PoolId)
		// Get the pool payment before remove payment
		paymentsPoolBefore := app.MillionsKeeper.ListPoolPayments(ctx, poolBefore.PoolId)

		// Test GetPoolPayment
		poolPayment, err := app.MillionsKeeper.GetPoolPayment(ctx, payment.PoolId, payment.PaymentId)
		suite.Require().NoError(err)
		suite.Require().Equal(payment.Amount.Amount.Int64(), poolPayment.Amount.Amount.Int64())
		suite.Require().Equal(payment.PoolId, poolPayment.PoolId)

		// Remove the payment
		app.MillionsKeeper.RemovePayment(ctx, &payment)

		// Decrement by 1 for each loop
		paymentsAccount := app.MillionsKeeper.ListAccountPayments(ctx, addr)
		suite.Require().Equal(len(payments)-1, len(paymentsAccount))

		// Decrement by 1 for each loop
		paymentsAccountPool := app.MillionsKeeper.ListAccountPoolPayments(ctx, addr, payment.PoolId)
		suite.Require().Equal(len(paymentsAccountPoolBefore)-1, len(paymentsAccountPool))

		// Decrement by 1 for each loop
		paymentsPool := app.MillionsKeeper.ListPoolPayments(ctx, payment.PoolId)
		suite.Require().Equal(len(paymentsPoolBefore)-1, len(paymentsPool))
		// Get the pool
		pool, err := app.MillionsKeeper.GetPool(ctx, payment.PoolId)
		suite.Require().NoError(err)
		// Test that the pool tvl is deducted by the payment amount
		suite.Require().Equal(poolBefore.TvlAmount.Int64()-payment.Amount.Amount.Int64(), pool.TvlAmount.Int64())
		// - Test that the pool count is correctly decremented for each removed payment
		poolBefore.PaymentorsCount--
		suite.Require().Equal(poolBefore.PaymentorsCount, pool.PaymentorsCount)
	}

	paymentsAll := app.MillionsKeeper.ListPayments(ctx)
	suite.Require().Equal(len(paymentsBefore), len(paymentsAll))
}

// TestPayment_UpdatePaymentStatus tests the logic of updating a payment from the store
func (suite *KeeperTestSuite) TestPayment_UpdatePaymentStatus() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)

	// Add 5 payments
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

		// Create a new payment and add it to the state
		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           pool.PoolId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}

	// - Test if the payment exist
	// Retrived payments
	payments := app.MillionsKeeper.ListPayments(ctx)
	// Retrieve the pool from the state
	pools := app.MillionsKeeper.ListPools(ctx)
	for i, pool := range pools {
		app.MillionsKeeper.UpdatePaymentStatus(ctx, pool.PoolId, payments[i].PaymentId, millionstypes.PaymentState_IcaDelegate, false)
		payment, err := app.MillionsKeeper.GetPoolPayment(ctx, pool.PoolId, payments[i].PaymentId)
		suite.Require().NoError(err)
		suite.Require().NotNil(payment, "Not nil")
		suite.Require().Equal(millionstypes.PaymentState_IcaDelegate, payment.State)
		suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)
	}
	payments = app.MillionsKeeper.ListPayments(ctx)
	// Trigger panic on wrong poolID
	panicF := func() {
		_, err := app.MillionsKeeper.GetPoolPayment(ctx, payments[10].PoolId, uint64(5))
		suite.Require().NoError(err)
	}
	suite.Require().Panics(panicF)
	// Trigger panic on wrong paymentID
	panicF = func() {
		_, err := app.MillionsKeeper.GetPoolPayment(ctx, uint64(5), payments[10].PoolId)
		suite.Require().NoError(err)
	}
	suite.Require().Panics(panicF)

	// - Test Error Management
	payments = app.MillionsKeeper.ListPayments(ctx)
	// Retrieve the pool from the state
	pools = app.MillionsKeeper.ListPools(ctx)
	for i, pool := range pools {
		app.MillionsKeeper.UpdatePaymentStatus(ctx, pool.PoolId, payments[i].PaymentId, millionstypes.PaymentState_IcaDelegate, true)
		payment, err := app.MillionsKeeper.GetPoolPayment(ctx, pool.PoolId, payments[i].PaymentId)
		suite.Require().NoError(err)
		suite.Require().NotNil(payment, "Not nil")
		suite.Require().Equal(millionstypes.PaymentState_Failure, payment.State)
		suite.Require().Equal(millionstypes.PaymentState_IcaDelegate, payment.ErrorState)
	}

	// - Test that the account payment was updated
	// - Test that the pool payment was updated
	pools = app.MillionsKeeper.ListPools(ctx)
	for i, pool := range pools {
		paymentsAccounts := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
		suite.Require().Equal(ctx.BlockTime(), paymentsAccounts[i].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), paymentsAccounts[i].UpdatedAtHeight)

		paymentsPoolAccount := app.MillionsKeeper.ListAccountPoolPayments(ctx, suite.addrs[0], pool.PoolId)
		suite.Require().Equal(ctx.BlockTime(), paymentsPoolAccount[0].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), paymentsPoolAccount[0].UpdatedAtHeight)

		paymentsPool := app.MillionsKeeper.ListPoolPayments(ctx, pool.PoolId)
		suite.Require().Equal(ctx.BlockTime(), paymentsPool[0].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), paymentsPool[0].UpdatedAtHeight)
	}
}

// TestPayment_EditPayment tests the logic of editing a payment's winnerAddr and sponsor
func (suite *KeeperTestSuite) TestPayment_EditPayment() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Set the denom
	denom := app.StakingKeeper.BondDenom(ctx)

	// Add 5 payments
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

		// Create a new payment and add it to the state
		app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
			PoolId:           pool.PoolId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_IbcTransfer,
			Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
		})
	}

	// Test Error Management
	// - Trigger err on wrong poolID
	_, err := app.MillionsKeeper.GetPoolPayment(ctx, uint64(10), uint64(5))
	suite.Require().Error(err)
	// - Trigger err on wrong paymentID
	_, err = app.MillionsKeeper.GetPoolPayment(ctx, uint64(5), uint64(10))
	suite.Require().Error(err)

	payments := app.MillionsKeeper.ListPayments(ctx)
	// Retrieve the pool from the state
	pools := app.MillionsKeeper.ListPools(ctx)
	for i, pool := range pools {
		err = app.MillionsKeeper.EditPayment(ctx, pool.PoolId, payments[i].PaymentId, suite.addrs[1], true)
		suite.Require().NoError(err)
		payment, err := app.MillionsKeeper.GetPoolPayment(ctx, pool.PoolId, payments[i].PaymentId)
		suite.Require().NoError(err)
		pool, err := app.MillionsKeeper.GetPool(ctx, payment.PoolId)
		suite.Require().NoError(err)
		suite.Require().NotNil(payment, "Not nil")
		suite.Require().Equal(suite.addrs[1].String(), payment.WinnerAddress)
		suite.Require().Equal(true, payment.IsSponsor)
		suite.Require().Equal(int64(1_000_000), pool.SponsorshipAmount.Int64())
	}

	// pool SponsorshipAmount should be sub in case user does not want to sponsor anymore
	pool, err := app.MillionsKeeper.GetPool(ctx, payments[0].PoolId)
	suite.Require().NoError(err)
	err = app.MillionsKeeper.EditPayment(ctx, pool.PoolId, payments[0].PaymentId, suite.addrs[1], false)
	suite.Require().NoError(err)
	pool, err = app.MillionsKeeper.GetPool(ctx, payments[0].PoolId)
	suite.Require().NoError(err)
	suite.Require().Equal(int64(0), pool.SponsorshipAmount.Int64())
}

// TestPayment_TransferPayment tests the full flow from the transfer till the delegation point
func (suite *KeeperTestSuite) TestPayment_TransferPayment() {
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
		AvailablePrizePool: sdk.NewCoin("uatom", math.NewInt(1000)),
	}))

	pools := app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 1)

	// - Test the full payment process from the transfer till the delegation point
	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[0].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[0].Validators[0].BondedAmount)

	// Create a new payment and add it to the state
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[0].PoolId,
		PaymentorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.PaymentState_IbcTransfer,
		Amount:           sdk.NewCoin("uatom", sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[0].LocalAddress), sdk.Coins{sdk.NewCoin(pools[0].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	payments := app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])

	// Test that the pool is not local
	suite.Require().Equal(false, pools[0].IsLocalZone(ctx))
	payment, err := app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().NotNil(payment, "Not nil")
	suite.Require().Equal(millionstypes.PaymentState_IbcTransfer, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)

	payments = app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	suite.Require().Len(payments, 1)

	// Simulate failed ackResponse AckResponseStatus_FAILURE
	err = app.MillionsKeeper.OnTransferPaymentToRemoteZoneCompleted(ctx, payments[0].GetPoolId(), payments[0].GetPaymentId(), true)
	suite.Require().NoError(err)

	// List payments to get latest status
	payments = app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	suite.Require().Len(payments, 1)
	// Failed Transfer should set the errorState to PaymentState_IbcTransfer and state to PaymentState_Failure
	suite.Require().Equal(millionstypes.PaymentState_Failure, payments[0].State)
	suite.Require().Equal(millionstypes.PaymentState_IbcTransfer, payments[0].ErrorState)

	// Update status to simulate that the state is initially on PaymentState_IbcTransfer
	app.MillionsKeeper.UpdatePaymentStatus(ctx, payments[0].PoolId, payments[0].PaymentId, millionstypes.PaymentState_IbcTransfer, false)

	// Simulate succesful ackResponse AckResponseStatus_SUCCESS
	err = app.MillionsKeeper.OnTransferPaymentToRemoteZoneCompleted(ctx, payments[0].GetPoolId(), payments[0].GetPaymentId(), false)
	suite.Require().NoError(err)

	// List payments to get latest state
	payments = app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	suite.Require().Len(payments, 1)
	// The payment will reached the sequence, err := k.BroadcastICAMessages an fail on OnDelegatePaymentOnRemoteZoneCompleted as we don't BroadcastICAMessages in our test
	suite.Require().Equal(millionstypes.PaymentState_Failure, payments[0].State)
	suite.Require().Equal(millionstypes.PaymentState_IcaDelegate, payments[0].ErrorState)

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

	// Create a new payment and add it to the state
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[1].PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_IbcTransfer,
		Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), sdk.Coins{sdk.NewCoin(pools[1].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 1)

	// Test that the pool is local
	suite.Require().Equal(true, pools[1].IsLocalZone(ctx))
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().NotNil(payment, "Not nil")
	suite.Require().Equal(millionstypes.PaymentState_IbcTransfer, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)

	// Trigger the Transfer payment to native chain
	err = app.MillionsKeeper.TransferPaymentToRemoteZone(ctx, payment.PoolId, payment.PaymentId)
	suite.Require().NoError(err)
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.PaymentState_Success, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)
}

// TestPayment_DelegatePayment tests the delegate payment from the transfer point till the final delegation
func (suite *KeeperTestSuite) TestPayment_DelegatePayment() {
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
		AvailablePrizePool: sdk.NewCoin("uatom", math.NewInt(1000)),
	}))

	pools := app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 1)

	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[0].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[0].Validators[0].BondedAmount)

	// Create a new payment and add it to the state
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[0].PoolId,
		PaymentorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.PaymentState_IcaDelegate,
		Amount:           sdk.NewCoin("uatom", sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[0].LocalAddress), sdk.Coins{sdk.NewCoin(pools[0].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	payments := app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	suite.Require().Len(payments, 1)

	// Test that the pool is not local
	suite.Require().Equal(false, pools[0].IsLocalZone(ctx))
	payment, err := app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().NotNil(payment, "Not nil")
	suite.Require().Equal(millionstypes.PaymentState_IcaDelegate, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)

	// Simulate a failed DelegatePaymentOnRemoteZone
	err = app.MillionsKeeper.DelegatePaymentOnRemoteZone(ctx, payment.PoolId, payment.PaymentId)
	suite.Require().NoError(err)

	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	// The payment will reached the sequence, err := k.BroadcastICAMessages and fail on OnDelegatePaymentOnRemoteZoneCompleted as we don't BroadcastICAMessages in our test
	suite.Require().Equal(millionstypes.PaymentState_Failure, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_IcaDelegate, payment.ErrorState)

	// Update status to simulate that the state is initially on PaymentState_IcaDelegate
	app.MillionsKeeper.UpdatePaymentStatus(ctx, payment.PoolId, payment.PaymentId, millionstypes.PaymentState_IcaDelegate, false)

	splits := []*millionstypes.SplitDelegation{{ValidatorAddress: cosmosPoolValidator, Amount: sdk.NewInt(int64(1_000_000))}}

	// Now trigger a successfull Delegation to simulate icacallbackstypes.AckResponseStatus_SUCCESS
	err = app.MillionsKeeper.OnDelegatePaymentOnRemoteZoneCompleted(ctx, payment.PoolId, payment.PaymentId, splits, false)
	suite.Require().NoError(err)
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payment.PoolId, payment.PaymentId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.PaymentState_Success, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)

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

	// Create a new payment and add it to the state
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[1].PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_IcaDelegate,
		Amount:           sdk.NewCoin(app.StakingKeeper.BondDenom(ctx), sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), sdk.Coins{sdk.NewCoin(pools[1].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 1)

	pools = app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 2)

	// Test that the pool is local
	suite.Require().Equal(true, pools[1].IsLocalZone(ctx))
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().NotNil(payment, "Not nil")
	suite.Require().Equal(millionstypes.PaymentState_IcaDelegate, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)

	// As it's local pool trigger a DelegatePaymentOnRemoteZone for the new payment
	err = app.MillionsKeeper.DelegatePaymentOnRemoteZone(ctx, payment.PoolId, payment.PaymentId)
	suite.Require().NoError(err)

	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.PaymentState_Success, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)
}

// TestPayment_BalancePayment tests the balance in case of success or failed payment
func (suite *KeeperTestSuite) TestPayment_BalancePayment() {
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

	// Create a new payment and add it to the state
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[0].PoolId,
		PaymentorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.PaymentState_IbcTransfer,
		Amount:           sdk.NewCoin("uatom", sdk.NewInt(1_000_000)),
	})

	err := app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[0].LocalAddress), sdk.Coins{sdk.NewCoin(pools[0].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	// Balance should have decreased by one payment
	balance := app.BankKeeper.GetBalance(ctx, uatomAddresses[0], "uatom")
	suite.Require().Equal(balanceBefore.Amount.Int64()-1_000_000, balance.Amount.Int64())
	// Module Acc should have increased by one payment
	balanceModuleAcc := app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[0].LocalAddress), "uatom")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64()+1_000_000, balanceModuleAcc.Amount.Int64())

	payments := app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	suite.Require().Len(payments, 1)

	app.MillionsKeeper.UpdatePaymentStatus(ctx, payments[0].PoolId, payments[0].PaymentId, millionstypes.PaymentState_IcaDelegate, false)

	suite.Require().Equal(false, pools[0].IsLocalZone(ctx))
	payment, err := app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().NotNil(payment, "Not nil")
	suite.Require().Equal(millionstypes.PaymentState_IcaDelegate, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)

	splits := []*millionstypes.SplitDelegation{{ValidatorAddress: cosmosPoolValidator, Amount: sdk.NewInt(int64(1_000_000))}}

	// Now trigger a successfull Delegation to simulate icacallbackstypes.AckResponseStatus_SUCCESS
	err = app.MillionsKeeper.OnDelegatePaymentOnRemoteZoneCompleted(ctx, payment.PoolId, payment.PaymentId, splits, false)
	suite.Require().NoError(err)

	// For remote pools we don't reach the success as k.BroadcastICAMessages fails on OnDelegatePaymentOnRemoteZoneCompleted

	// Balance should remain unchanged
	balance = app.BankKeeper.GetBalance(ctx, uatomAddresses[0], "uatom")
	suite.Require().Equal(balanceBefore.Amount.Int64()-1_000_000, balance.Amount.Int64())
	// Module Account should have the payment amount
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
	balanceModuleAccBefore = app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), "ufury")
	suite.Require().Equal(int64(0), balanceModuleAccBefore.Amount.Int64())

	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[1].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[1].Validators[0].BondedAmount)

	// Create a new payment and add it to the state
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[1].PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_IbcTransfer,
		Amount:           sdk.NewCoin(denom, sdk.NewInt(2_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), sdk.Coins{sdk.NewCoin(pools[1].Denom, sdk.NewInt(2_000_000))})
	suite.Require().NoError(err)
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 1)

	// Balance should have decreased by two payments
	balance = app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(balanceBefore.Amount.Int64()-2_000_000, balance.Amount.Int64())
	// Module Acc should have increased by two payments
	balanceModuleAcc = app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), "ufury")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64()+2_000_000, balanceModuleAcc.Amount.Int64())

	// There should be no delegation shares yet
	delegationAmount = app.StakingKeeper.GetDelegatorDelegations(ctx, suite.moduleAddrs[0], 10)
	suite.Require().Len(delegationAmount, 0)

	suite.Require().Equal(true, pools[1].IsLocalZone(ctx))
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().NotNil(payment, "Not nil")
	suite.Require().Equal(millionstypes.PaymentState_IbcTransfer, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)

	err = app.MillionsKeeper.OnTransferPaymentToRemoteZoneCompleted(ctx, payment.PoolId, payment.PaymentId, true)
	suite.Require().NoError(err)
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.PaymentState_Failure, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_IbcTransfer, payment.ErrorState)

	// Balance should be unchanged
	balance = app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(balanceBefore.Amount.Int64()-2_000_000, balance.Amount.Int64())
	// Module Acc should have still the paymented amount
	balanceModuleAcc = app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), "ufury")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64()+2_000_000, balanceModuleAcc.Amount.Int64())

	// There should be no delegation shares yet
	delegationAmount = app.StakingKeeper.GetDelegatorDelegations(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), 10)
	suite.Require().Len(delegationAmount, 0)

	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	// Update the status to trigger successful PaymentToNativeChain
	app.MillionsKeeper.UpdatePaymentStatus(ctx, payment.PoolId, payment.PaymentId, millionstypes.PaymentState_IbcTransfer, false)

	err = app.MillionsKeeper.TransferPaymentToRemoteZone(ctx, payment.PoolId, payment.PaymentId)
	suite.Require().NoError(err)
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.PaymentState_Success, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)

	// Balance should be unchanged
	balance = app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(balanceBefore.Amount.Int64()-2_000_000, balance.Amount.Int64())
	// Module Acc should have delegated the payment
	balanceModuleAcc = app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), "ufury")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64(), balanceModuleAcc.Amount.Int64())

	// The delegation amount should reflect the shares delegated - client precision
	delegationAmount = app.StakingKeeper.GetDelegatorDelegations(ctx, sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), 10)
	paymentToShare := sdk.NewDecFromInt(sdk.NewInt(payment.Amount.Amount.Int64() / 1_000_000))
	suite.Require().Equal(paymentToShare, delegationAmount[0].Shares)
}

// TestPayment_FullPaymentProcess tests the complete logic for payments for a local pool
func (suite *KeeperTestSuite) TestPayment_FullPaymentProcess() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	drawDelta1 := 1 * time.Hour
	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)

	// - Test the full payment process for a local pool
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
	balanceModuleAccBefore := app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[0].IcaPaymentAddress), "ufury")
	suite.Require().Equal(int64(0), balanceModuleAccBefore.Amount.Int64())

	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[0].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[0].Validators[0].BondedAmount)

	// Create a new payment and add it to the state
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[0].PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_IbcTransfer,
		Amount:           sdk.NewCoin(denom, sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[0].IcaPaymentAddress), sdk.Coins{sdk.NewCoin(pools[0].Denom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	payments := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 1)

	// Balance should have decreased by one payment
	balance := app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(balanceBefore.Amount.Int64()-1_000_000, balance.Amount.Int64())
	// Module Acc should have increased by one payment
	balanceModuleAcc := app.BankKeeper.GetBalance(ctx, sdk.MustAccAddressFromBech32(pools[0].IcaPaymentAddress), "ufury")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64()+1_000_000, balanceModuleAcc.Amount.Int64())

	// There should no delegation shares
	delegationAmount := app.StakingKeeper.GetDelegatorDelegations(ctx, sdk.MustAccAddressFromBech32(pools[0].IcaPaymentAddress), 10)
	suite.Require().Len(delegationAmount, 0)

	// Test if it's a local pool
	suite.Require().Equal(true, pools[0].IsLocalZone(ctx))
	payment, err := app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().NotNil(payment, "Not nil")
	suite.Require().Equal(millionstypes.PaymentState_IbcTransfer, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)

	err = app.MillionsKeeper.TransferPaymentToRemoteZone(ctx, payment.PoolId, payment.PaymentId)
	suite.Require().NoError(err)
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.PaymentState_Success, payment.State)
	suite.Require().Equal(millionstypes.PaymentState_Unspecified, payment.ErrorState)

	// Balance should be unchanged
	balance = app.BankKeeper.GetBalance(ctx, suite.addrs[0], "ufury")
	suite.Require().Equal(balanceBefore.Amount.Int64()-1_000_000, balance.Amount.Int64())
	// Module Acc should have delegated the payment
	balanceModuleAcc = app.BankKeeper.GetBalance(ctx, suite.moduleAddrs[0], "ufury")
	suite.Require().Equal(balanceModuleAccBefore.Amount.Int64(), balanceModuleAcc.Amount.Int64())

	// The delegation amount should reflect the shares delegated - client precision
	delegationAmount = app.StakingKeeper.GetDelegatorDelegations(ctx, sdk.MustAccAddressFromBech32(pools[0].IcaPaymentAddress), 10)
	paymentToShare := sdk.NewDecFromInt(sdk.NewInt(payment.Amount.Amount.Int64() / 1_000_000))
	suite.Require().Equal(paymentToShare, delegationAmount[0].Shares)
}
