package keeper_test

import (
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	apptesting "github.com/furysport/furya-chain/v2/app/testing"
	epochstypes "github.com/furysport/furya-chain/v2/x/epochs/types"
	millionstypes "github.com/furysport/furya-chain/v2/x/millions/types"
)

// TestWithdrawal_IDsGeneration tests the withdrawal ID generation
func (suite *KeeperTestSuite) TestWithdrawal_IDsGeneration() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Add 10 payments
	for i := 0; i < 10; i++ {
		poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
		paymentID := app.MillionsKeeper.GetNextPaymentIdAndIncrement(ctx)
		withdrawalID := app.MillionsKeeper.GetNextWithdrawalIdAndIncrement(ctx)
		app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{PoolId: poolID}))

		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           poolID,
			PaymentId:        paymentID,
			PaymentorAddress: suite.addrs[0].String(),
			WithdrawalId:     withdrawalID,
			ToAddress:        suite.addrs[0].String(),
			State:            millionstypes.WithdrawalState_IcaUndelegate,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
		// Test that withdrawalID is incremented
		suite.Require().Equal(uint64(i+1), withdrawalID)

		// Test that we never override an existing entity
		panicF := func() {
			app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
				PoolId:           poolID,
				PaymentId:        paymentID,
				WithdrawalId:     withdrawalID,
				PaymentorAddress: suite.addrs[0].String(),
				ToAddress:        suite.addrs[0].String(),
				State:            millionstypes.WithdrawalState_IcaUndelegate,
				Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
			})
		}
		suite.Require().Panics(panicF)
	}
}

// TestWithdrawal_AddWithdrawal tests the logic of adding a withdrawal to the store
func (suite *KeeperTestSuite) TestWithdrawal_AddWithdrawal() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

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
	withdrawalsBefore := app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
	suite.Require().Len(withdrawalsBefore, 0)
	withdrawals := app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
	suite.Require().Len(withdrawals, 0)

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
			State:            millionstypes.PaymentState_Success,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
	}

	payments := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 5)

	// Add 4 withdrawals (keep 1 payment)
	for i := 0; i < 4; i++ {
		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           payments[i].PoolId,
			PaymentId:        payments[i].PaymentId,
			PaymentorAddress: suite.addrs[0].String(),
			ToAddress:        suite.addrs[0].String(),
			State:            millionstypes.WithdrawalState_IcaUndelegate,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
	}

	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
	suite.Require().Len(withdrawals, 4)

	// Remove the payment associated witht the previous withdrawals to respect flow logic
	for _, withdrawal := range withdrawals {
		app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
			PoolId:           withdrawal.PoolId,
			PaymentId:        withdrawal.PaymentId,
			PaymentorAddress: suite.addrs[0].String(),
			WinnerAddress:    suite.addrs[0].String(),
			State:            millionstypes.PaymentState_Success,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
	}

	// Only 1 payment should remain
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 1)

	// - Test the validate basics
	// -- Test one successful withdrawal with all correct validate basics
	panicF := func() {
		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentId:        payments[len(payments)-1].PaymentId,
			PaymentorAddress: suite.addrs[0].String(),
			ToAddress:        suite.addrs[0].String(),
			State:            millionstypes.WithdrawalState_IcaUndelegate,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().NotPanics(panicF)
	// Simulate the remove payment affter a successful withdrawal
	app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
		PoolId:           payments[len(payments)-1].PoolId,
		PaymentId:        payments[len(payments)-1].PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 5)
	// -- Test that the withdrawal validation panics with an invalid pool ID
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	panicF = func() {
		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           uint64(0),
			PaymentId:        payments[len(payments)-1].PaymentId,
			PaymentorAddress: suite.addrs[0].String(),
			ToAddress:        suite.addrs[0].String(),
			State:            millionstypes.WithdrawalState_IcaUndelegate,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the withdrawal validation panics with an invalid withdrawalID
	panicF = func() {
		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentId:        payments[len(payments)-1].PaymentId,
			WithdrawalId:     uint64(4),
			PaymentorAddress: suite.addrs[0].String(),
			ToAddress:        suite.addrs[0].String(),
			State:            millionstypes.WithdrawalState_Unspecified,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
		withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
		withdrawals[len(withdrawals)-1].WithdrawalId = 0
	}
	suite.Require().Panics(panicF)
	// -- Test that the withdrawal validation panics with an invalid state
	panicF = func() {
		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentId:        payments[len(payments)-1].PaymentId,
			PaymentorAddress: suite.addrs[0].String(),
			ToAddress:        suite.addrs[0].String(),
			State:            millionstypes.WithdrawalState_Unspecified,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the withdrawal validation panics with an invalid PaymentorAddress
	panicF = func() {
		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentId:        payments[len(payments)-1].PaymentId,
			PaymentorAddress: "",
			ToAddress:        suite.addrs[0].String(),
			State:            millionstypes.WithdrawalState_IcaUndelegate,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
	}
	suite.Require().Panics(panicF)
	// -- Test that the withdrawal validation panics with an invalid withdrawal amount
	panicF = func() {
		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           payments[len(payments)-1].PoolId,
			PaymentId:        payments[len(payments)-1].PaymentId,
			PaymentorAddress: suite.addrs[0].String(),
			ToAddress:        suite.addrs[0].String(),
			State:            millionstypes.WithdrawalState_IcaUndelegate,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(0)),
		})
	}
	suite.Require().Panics(panicF)

	// Verify that there is no more payments
	payments = app.MillionsKeeper.ListPayments(ctx)
	suite.Require().Len(payments, 0)

	// Test all withdrawals
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Equal(len(withdrawalsBefore)+5, len(withdrawals))
	// - Test that the account withdrawals are correctly updated
	accountWithdrawals := app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
	suite.Require().Equal(len(withdrawalsBefore)+5, len(accountWithdrawals))

	// - Test that the pool withdrawals are correctly updated
	for _, withdrawal := range withdrawals {
		addr := sdk.MustAccAddressFromBech32(withdrawal.PaymentorAddress)
		// Test ListAccountPoolWithdrawals
		withdrawalsAccountPool := app.MillionsKeeper.ListAccountPoolWithdrawals(ctx, addr, withdrawal.PoolId)
		suite.Require().Equal(len(withdrawalsBefore)+1, len(withdrawalsAccountPool))
		// Test ListPoolWithdrawals
		withdrawalsPool := app.MillionsKeeper.ListPoolWithdrawals(ctx, withdrawal.PoolId)
		suite.Require().Equal(len(withdrawalsBefore)+1, len(withdrawalsPool))
		// Test GetPoolWithdrawal
		poolWithdrawal, err := app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawal.PoolId, withdrawal.WithdrawalId)
		suite.Require().NoError(err)
		suite.Require().Equal(poolWithdrawal.Amount.Amount.Int64(), withdrawal.Amount.Amount.Int64())
		suite.Require().Equal(poolWithdrawal.PoolId, withdrawal.PoolId)
	}
}

// TestWithdrawal_RemoveWithdrawal tests the logic of removing a withdrawal from the store
func (suite *KeeperTestSuite) TestWithdrawal_RemoveWithdrawal() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Initialize the withdrawals
	withdrawalsBefore := app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawalsBefore, 0)
	accountwithdrawalsBefore := app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
	suite.Require().Len(accountwithdrawalsBefore, 0)

	// - Test that withdrawals with wrong poolID, withdrawalID cannot be removed from the pool withdrawal
	for i := 0; i < 5; i++ {
		poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
		paymentID := app.MillionsKeeper.GetNextPaymentIdAndIncrement(ctx)
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

		// Create a new payment and add it to the state
		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           poolID,
			PaymentId:        paymentID,
			PaymentorAddress: suite.addrs[0].String(),
			ToAddress:        suite.addrs[0].String(),
			State:            millionstypes.WithdrawalState_IbcTransfer,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
	}

	// Test RemoveWithdrawal with incorrect poolID
	withdrawal := millionstypes.Withdrawal{
		PoolId:           uint64(0),
		PaymentId:        uint64(1),
		WithdrawalId:     uint64(1),
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
		State:            millionstypes.WithdrawalState_IbcTransfer,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	}
	err := app.MillionsKeeper.RemoveWithdrawal(ctx, withdrawal)
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)

	// Test RemoveWithdrawal with incorrect withdrawalID
	withdrawal = millionstypes.Withdrawal{
		PoolId:           uint64(1),
		PaymentId:        uint64(1),
		WithdrawalId:     uint64(0),
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
		State:            millionstypes.WithdrawalState_IbcTransfer,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	}
	err = app.MillionsKeeper.RemoveWithdrawal(ctx, withdrawal)
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidID)

	// Test RemoveWithdrawal with incorrect PaymentorAddress
	withdrawal = millionstypes.Withdrawal{
		PoolId:           uint64(1),
		PaymentId:        uint64(1),
		WithdrawalId:     uint64(1),
		PaymentorAddress: "no-address",
		ToAddress:        suite.addrs[0].String(),
		State:            millionstypes.WithdrawalState_IbcTransfer,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	}
	err = app.MillionsKeeper.RemoveWithdrawal(ctx, withdrawal)
	suite.Require().ErrorIs(err, millionstypes.ErrInvalidWithdrawalPaymentorAddress)
	// Test RemoveWithdrawal with incorrect amount
	withdrawal = millionstypes.Withdrawal{
		PoolId:           uint64(1),
		PaymentId:        uint64(1),
		WithdrawalId:     uint64(1),
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
		State:            millionstypes.WithdrawalState_IbcTransfer,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(0)),
	}
	err = app.MillionsKeeper.RemoveWithdrawal(ctx, withdrawal)
	suite.Require().ErrorIs(err, millionstypes.ErrPaymentAlreadyWithdrawn)

	// - Test list All withdrawals
	withdrawals := app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 5)
	// - Test account withdrawals
	accountWithdrawals := app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
	suite.Require().Len(accountWithdrawals, 5)
	// - Test pool withdrawals
	pools := app.MillionsKeeper.ListPools(ctx)
	for _, pool := range pools {
		// Test ListAccountPoolWithdrawals
		withdrawalsAccountPool := app.MillionsKeeper.ListAccountPoolWithdrawals(ctx, suite.addrs[0], pool.PoolId)
		suite.Require().Equal(len(withdrawalsBefore)+1, len(withdrawalsAccountPool))
		// Test ListPoolWithdrawals
		withdrawalsPool := app.MillionsKeeper.ListPoolWithdrawals(ctx, pool.PoolId)
		suite.Require().Equal(len(withdrawalsBefore)+1, len(withdrawalsPool))
	}

	// - Test remove withdrawals
	for _, withdrawal := range withdrawals {
		// Get the pool before remove withdrawal
		poolBefore, err := app.MillionsKeeper.GetPool(ctx, withdrawal.PoolId)
		suite.Require().NoError(err)
		// Get the Account withdrawal before remove withdrawal
		addr := sdk.MustAccAddressFromBech32(withdrawal.PaymentorAddress)
		accountWithdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, addr)
		// Get the Account pool payment before remove payment
		withdrawalsAccountPoolBefore := app.MillionsKeeper.ListAccountPoolWithdrawals(ctx, addr, poolBefore.PoolId)
		// Get the pool payment before remove payment
		withdrawalsPoolBefore := app.MillionsKeeper.ListPoolWithdrawals(ctx, poolBefore.PoolId)
		// Test GetPoolWithdrawal
		poolPayment, err := app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawal.PoolId, withdrawal.WithdrawalId)
		suite.Require().NoError(err)
		suite.Require().Equal(withdrawal.Amount.Amount.Int64(), poolPayment.Amount.Amount.Int64())
		suite.Require().Equal(withdrawal.PoolId, poolPayment.PoolId)
		// Remove the withdrawals
		err = app.MillionsKeeper.RemoveWithdrawal(ctx, withdrawal)
		suite.Require().NoError(err)
		// Decrement by 1 for each loop
		withdrawalsAccount := app.MillionsKeeper.ListAccountWithdrawals(ctx, addr)
		suite.Require().Equal(len(accountWithdrawals)-1, len(withdrawalsAccount))
		// Decrement by 1 for each loop
		withdrawalsAccountPool := app.MillionsKeeper.ListAccountPoolWithdrawals(ctx, addr, withdrawal.PoolId)
		suite.Require().Equal(len(withdrawalsAccountPoolBefore)-1, len(withdrawalsAccountPool))
		// Decrement by 1 for each loop
		withdrawalsPool := app.MillionsKeeper.ListPoolWithdrawals(ctx, withdrawal.PoolId)
		suite.Require().Equal(len(withdrawalsPoolBefore)-1, len(withdrawalsPool))
	}

	// - Test list All withdrawals
	// The last withdrawal has status WithdrawalState_IbcTransfer
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 0)
}

// TestWithdrawal_UpdateWithdrawalStatus test the logic of updating a withdrawal to the store
func (suite *KeeperTestSuite) TestWithdrawal_UpdateWithdrawalStatus() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Generate 5 withdrawals
	for i := 0; i < 5; i++ {
		poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
		paymentID := app.MillionsKeeper.GetNextPaymentIdAndIncrement(ctx)
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

		// Create a new payment and add it to the state
		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           poolID,
			PaymentId:        paymentID,
			PaymentorAddress: suite.addrs[0].String(),
			ToAddress:        suite.addrs[0].String(),
			State:            millionstypes.WithdrawalState_IcaUndelegate,
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		})
	}

	withdrawals := app.MillionsKeeper.ListWithdrawals(ctx)
	// - Test the error if we cannot find the withdrawal with wrong poolID
	panicF := func() {
		app.MillionsKeeper.UpdateWithdrawalStatus(ctx, uint64(0), withdrawals[0].WithdrawalId, millionstypes.WithdrawalState_IcaUndelegate, &time.Time{}, false)
	}
	suite.Require().Panics(panicF)
	// - Test the error if we cannot find the withdrawal with wrong withdrawalID
	panicF = func() {
		app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawals[0].PoolId, uint64(0), millionstypes.WithdrawalState_IcaUndelegate, &time.Time{}, false)
	}
	suite.Require().Panics(panicF)
	// - Test the error management in case of true event
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	for _, withdrawal := range withdrawals {
		app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawal.PoolId, withdrawal.WithdrawalId, millionstypes.WithdrawalState_IcaUndelegate, &time.Time{}, true)
	}
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	for i, withdrawal := range withdrawals {
		addr := sdk.MustAccAddressFromBech32(withdrawal.PaymentorAddress)
		// Test GetPoolWithdrawal
		poolWithdrawal, err := app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawal.PoolId, withdrawal.WithdrawalId)
		suite.Require().NoError(err)
		suite.Require().Equal(poolWithdrawal.Amount.Amount.Int64(), withdrawal.Amount.Amount.Int64())
		suite.Require().Equal(poolWithdrawal.PoolId, withdrawal.PoolId)
		// Test ListAccountWithdrawals
		withdrawalsAccounts := app.MillionsKeeper.ListAccountWithdrawals(ctx, addr)
		suite.Require().Equal(ctx.BlockTime(), withdrawalsAccounts[i].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), withdrawalsAccounts[i].UpdatedAtHeight)
		suite.Require().Equal(millionstypes.WithdrawalState_Failure, withdrawals[i].State)
		suite.Require().Equal(millionstypes.WithdrawalState_IcaUndelegate, withdrawals[i].ErrorState)
		// Test ListAccountPoolWithdrawals
		withdrawalsPoolAccounts := app.MillionsKeeper.ListAccountPoolWithdrawals(ctx, addr, withdrawal.PoolId)
		suite.Require().Equal(ctx.BlockTime(), withdrawalsPoolAccounts[0].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), withdrawalsPoolAccounts[0].UpdatedAtHeight)
		suite.Require().Equal(millionstypes.WithdrawalState_Failure, withdrawals[i].State)
		suite.Require().Equal(millionstypes.WithdrawalState_IcaUndelegate, withdrawals[i].ErrorState)
		// Test ListPoolWithdrawals
		withdrawalsPool := app.MillionsKeeper.ListPoolWithdrawals(ctx, withdrawal.PoolId)
		suite.Require().Equal(ctx.BlockTime(), withdrawalsPool[0].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), withdrawalsPool[0].UpdatedAtHeight)
		suite.Require().Equal(millionstypes.WithdrawalState_Failure, withdrawals[i].State)
		suite.Require().Equal(millionstypes.WithdrawalState_IcaUndelegate, withdrawals[i].ErrorState)
	}

	// - Test the error management in case of false event
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	for _, withdrawal := range withdrawals {
		app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawal.PoolId, withdrawal.WithdrawalId, millionstypes.WithdrawalState_IcaUndelegate, &time.Time{}, false)
	}
	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)
	for i, withdrawal := range withdrawals {
		// Test GetPoolWithdrawal
		poolWithdrawal, err := app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawal.PoolId, withdrawal.WithdrawalId)
		suite.Require().NoError(err)
		suite.Require().Equal(poolWithdrawal.Amount.Amount.Int64(), withdrawal.Amount.Amount.Int64())
		suite.Require().Equal(poolWithdrawal.PoolId, withdrawal.PoolId)
		// Test ListAccountWithdrawals
		withdrawalsAccounts := app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
		suite.Require().Equal(ctx.BlockTime(), withdrawalsAccounts[i].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), withdrawalsAccounts[i].UpdatedAtHeight)
		suite.Require().Equal(millionstypes.WithdrawalState_IcaUndelegate, withdrawals[i].State)
		suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawals[i].ErrorState)
		// Test ListAccountPoolWithdrawals
		withdrawalsPoolAccounts := app.MillionsKeeper.ListAccountPoolWithdrawals(ctx, suite.addrs[0], withdrawal.PoolId)
		suite.Require().Equal(ctx.BlockTime(), withdrawalsPoolAccounts[0].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), withdrawalsPoolAccounts[0].UpdatedAtHeight)
		suite.Require().Equal(millionstypes.WithdrawalState_IcaUndelegate, withdrawals[i].State)
		suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawals[i].ErrorState)
		// Test ListPoolWithdrawals
		withdrawalsPool := app.MillionsKeeper.ListPoolWithdrawals(ctx, withdrawal.PoolId)
		suite.Require().Equal(ctx.BlockTime(), withdrawalsPool[0].UpdatedAt)
		suite.Require().Equal(ctx.BlockHeight(), withdrawalsPool[0].UpdatedAtHeight)
		suite.Require().Equal(millionstypes.WithdrawalState_IcaUndelegate, withdrawals[i].State)
		suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawals[i].ErrorState)
	}
}

// TestWithdrawal_UndelegateWithdrawal tests the flow from the undelegation till the transfer
func (suite *KeeperTestSuite) TestWithdrawal_UndelegateWithdrawals() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
	drawDelta1 := 1 * time.Hour
	uatomAddresses := apptesting.AddTestAddrsWithDenom(app, ctx, 7, sdk.NewInt(1_000_0000_000), remotePoolDenom)
	frequency := uint64(4)
	// Assuming first epoch is 1, and nextEpochUnbonding is the 4th one
	for epoch := int64(1); epoch <= 4; epoch++ {
		epochInfo, err := TriggerEpochUpdate(suite)
		suite.Require().NoError(err)
		suite.Require().Equal(epoch, epochInfo.CurrentEpoch)

		_, err = TriggerEpochTrackerUpdate(suite, epochInfo)
		suite.Require().NoError(err)
	}
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:              poolID,
		Bech32PrefixValAddr: remoteBech32PrefixValAddr,
		ChainId:             remoteChainId,
		Denom:               remotePoolDenom,
		NativeDenom:         remotePoolDenom,
		ConnectionId:        remoteConnectionId,
		TransferChannelId:   remoteTransferChannelId,
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
		AvailablePrizePool: sdk.NewCoin(remotePoolDenom, math.NewInt(1000)),
	}))

	// List the pools
	pools := app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 1)

	// Verify that the validators is enabled and the bondedamount
	suite.Require().Equal(true, pools[0].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[0].Validators[0].BondedAmount)

	// Create a new payment and simulate successful transfered payment
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[0].PoolId,
		PaymentorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.PaymentState_IcaDelegate,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
	})

	err := app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[0].LocalAddress), sdk.Coins{sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	payments := app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	payment, err := app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)

	// Simulate a delegation on native chain for remote pool
	splits := []*millionstypes.SplitDelegation{{ValidatorAddress: cosmosPoolValidator, Amount: sdk.NewInt(int64(1_000_000))}}
	err = app.MillionsKeeper.OnDelegatePaymentOnRemoteZoneCompleted(ctx, payment.PoolId, payment.PaymentId, splits, false)
	suite.Require().NoError(err)

	payments = app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)

	// Add withdrawal to the state
	app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: uatomAddresses[0].String(),
		ToAddress:        uatomAddresses[0].String(),
		State:            millionstypes.WithdrawalState_Pending,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
	})

	withdrawals := app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	suite.Require().Len(withdrawals, 1)

	// Remove payment to simulate flow
	app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
	})

	withdrawals = app.MillionsKeeper.ListWithdrawals(ctx)

	err = app.MillionsKeeper.AddEpochUnbonding(ctx, withdrawals[0], false)
	suite.Require().NoError(err)

	// There should be ne more payments
	payments = app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	suite.Require().Len(payments, 0)

	// Test if pool is remote
	suite.Require().Equal(false, pools[0].IsLocalZone(ctx))
	withdrawal, err := app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)
	suite.Require().NotNil(payment, "Not nil")
	suite.Require().Equal(millionstypes.WithdrawalState_Pending, withdrawal.State)
	suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawal.ErrorState)

	// Simulate failed ackResponse AckResponseStatus_FAILURE
	// Get the millions internal module tracker
	epochTracker, err := app.MillionsKeeper.GetEpochTracker(ctx, epochstypes.DAY_EPOCH, millionstypes.WithdrawalTrackerType)
	suite.Require().NoError(err)
	// Get epoch unbonding
	currentEpochUnbonding, err := app.MillionsKeeper.GetEpochPoolUnbonding(ctx, epochTracker.EpochNumber+frequency, 1)
	suite.Require().NoError(err)
	// Remove to simulate epoch flow
	err = app.MillionsKeeper.RemoveEpochUnbonding(ctx, currentEpochUnbonding)
	suite.Require().NoError(err)

	app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId, millionstypes.WithdrawalState_IcaUndelegate, &time.Time{}, false)
	err = app.MillionsKeeper.OnUndelegateWithdrawalsOnRemoteZoneCompleted(ctx, withdrawal.GetPoolId(), []uint64{withdrawal.GetWithdrawalId()}, &time.Time{}, true)
	suite.Require().NoError(err)
	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)
	// Pushed back to the epoch unbonding
	suite.Require().Equal(millionstypes.WithdrawalState_Pending, withdrawal.State)
	suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawal.ErrorState)

	undbondingTime := ctx.BlockTime().Add(time.Second)

	// Simulate ackResponse AckResponseStatus_Success
	app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId, millionstypes.WithdrawalState_IcaUndelegate, &time.Time{}, false)
	err = app.MillionsKeeper.OnUndelegateWithdrawalsOnRemoteZoneCompleted(ctx, withdrawal.GetPoolId(), []uint64{withdrawal.GetWithdrawalId()}, &undbondingTime, false)
	suite.Require().NoError(err)
	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.WithdrawalState_IcaUnbonding, withdrawal.State)
	suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawal.ErrorState)
	suite.Require().WithinDuration(*withdrawal.UnbondingEndsAt, undbondingTime, time.Second)

	// Test local pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:      2,
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
	}))
	// List pools
	pools = app.MillionsKeeper.ListPools(ctx)

	// - Test if the validator is enabled and bondedamount
	suite.Require().Equal(true, pools[1].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[1].Validators[0].BondedAmount)

	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[1].PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_IbcTransfer,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])

	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)

	// Simulate the transfer to native chain
	err = app.MillionsKeeper.TransferPaymentToRemoteZone(ctx, payment.PoolId, payment.PaymentId)
	suite.Require().NoError(err)

	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)

	// Add withdrawal
	app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
		State:            millionstypes.WithdrawalState_Pending,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})

	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
	suite.Require().Len(withdrawals, 1)

	// Remove payment to respect the flow
	app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	// There should be ne more payments
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 0)

	err = app.MillionsKeeper.AddEpochUnbonding(ctx, withdrawals[0], false)
	suite.Require().NoError(err)

	// Test local pool
	suite.Require().Equal(true, pools[1].IsLocalZone(ctx))
	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.WithdrawalState_Pending, withdrawal.State)
	suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawal.ErrorState)

	// Get the millions internal module tracker
	epochTracker, err = app.MillionsKeeper.GetEpochTracker(ctx, epochstypes.DAY_EPOCH, millionstypes.WithdrawalTrackerType)
	suite.Require().NoError(err)

	// Get epoch unbonding
	currentEpochUnbonding, err = app.MillionsKeeper.GetEpochPoolUnbonding(ctx, epochTracker.EpochNumber+frequency, 2)
	suite.Require().NoError(err)

	// Simulate successful undelegation flow
	err = app.MillionsKeeper.UndelegateWithdrawalsOnRemoteZone(ctx, currentEpochUnbonding)
	suite.Require().NoError(err)
	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.WithdrawalState_IcaUnbonding, withdrawal.State)
	suite.Require().Equal(millionstypes.WithdrawalState_Unspecified, withdrawal.ErrorState)
	// There is 21 days unbonding time
	suite.Require().WithinDuration(*withdrawal.UnbondingEndsAt, undbondingTime, 21*24*time.Hour)
}

// TestWithdrawal_TransferWithdrawal tests from the undelegation point till the transfer to Local is completed
func (suite *KeeperTestSuite) TestWithdrawal_TransferWithdrawal() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
	drawDelta1 := 1 * time.Hour
	uatomAddresses := apptesting.AddTestAddrsWithDenom(app, ctx, 7, sdk.NewInt(1_000_0000_000), remotePoolDenom)

	// Remote pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:              poolID,
		Bech32PrefixValAddr: remoteBech32PrefixValAddr,
		ChainId:             remoteChainId,
		Denom:               remotePoolDenom,
		NativeDenom:         remotePoolDenom,
		ConnectionId:        remoteConnectionId,
		TransferChannelId:   remoteTransferChannelId,
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
		AvailablePrizePool: sdk.NewCoin(remotePoolDenom, math.NewInt(1000)),
	}))

	// List pools
	pools := app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 1)

	// Verify that the validators is enabled and the bondedamount
	suite.Require().Equal(true, pools[0].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[0].Validators[0].BondedAmount)

	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[0].PoolId,
		PaymentorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[0].LocalAddress), sdk.Coins{sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	payments := app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	payment, err := app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)

	unbondingTime := ctx.BlockTime().Add(-2 * time.Second)

	// Simulate that we reached the unbonding step already (done in previous test)
	app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: uatomAddresses[0].String(),
		ToAddress:        uatomAddresses[0].String(),
		State:            millionstypes.WithdrawalState_IcaUnbonding,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
		UnbondingEndsAt:  &unbondingTime,
	})

	withdrawals := app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	suite.Require().Len(withdrawals, 1)

	app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
	})

	payments = app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	suite.Require().Len(payments, 0)

	withdrawal, err := app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)

	// Update status to simulate transfer from to local chain
	app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawal.PoolId, withdrawal.WithdrawalId, millionstypes.WithdrawalState_IbcTransfer, withdrawal.UnbondingEndsAt, false)

	// Simulate failed ackResponse AckResponseStatus_FAILURE
	err = app.MillionsKeeper.OnTransferWithdrawalToRecipientCompleted(ctx, withdrawal.GetPoolId(), withdrawal.GetWithdrawalId(), true)
	suite.Require().NoError(err)
	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.WithdrawalState_Failure, withdrawal.State)
	suite.Require().Equal(millionstypes.WithdrawalState_IbcTransfer, withdrawal.ErrorState)

	// Update status to simulate transfer from to local chain
	app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawal.PoolId, withdrawal.WithdrawalId, millionstypes.WithdrawalState_IbcTransfer, withdrawal.UnbondingEndsAt, false)
	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)

	// Simulate success ackResponse AckResponseStatus_Success
	err = app.MillionsKeeper.OnTransferWithdrawalToRecipientCompleted(ctx, withdrawal.GetPoolId(), withdrawal.GetWithdrawalId(), false)
	suite.Require().NoError(err)
	// There should be no more withdrawal if successfully processed
	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	suite.Require().Len(withdrawals, 0)

	// Test local pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:      app.MillionsKeeper.GetNextPoolID(ctx),
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
	}))
	pools = app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 2)

	// - Test if the validator is enabled and bonded amount
	suite.Require().Equal(true, pools[1].Validators[0].IsEnabled)
	suite.Require().Equal(sdk.NewInt(1_000_000), pools[1].Validators[0].BondedAmount)

	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[1].PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])

	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)

	// Add withdrawal
	app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
		State:            millionstypes.WithdrawalState_IcaUnbonding,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		UnbondingEndsAt:  &unbondingTime,
	})

	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
	suite.Require().Len(withdrawals, 1)

	// Remove payment to respect the flow
	app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})

	// There should be no more payments
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 0)

	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].PaymentId)
	suite.Require().NoError(err)
	app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawal.PoolId, withdrawal.WithdrawalId, millionstypes.WithdrawalState_IbcTransfer, withdrawal.UnbondingEndsAt, false)
	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].PaymentId)
	suite.Require().NoError(err)

	err = app.MillionsKeeper.TransferWithdrawalToRecipient(ctx, withdrawal.PoolId, withdrawal.WithdrawalId)
	suite.Require().NoError(err)

	// There should no more withdrawal if successfully processed
	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
	suite.Require().Len(withdrawals, 0)
}

// TestWithdrawal_BankSend tests from the undelegation point the bank send to native is completed
func (suite *KeeperTestSuite) TestWithdrawal_BankSend() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
	drawDelta1 := 1 * time.Hour
	uatomAddresses := apptesting.AddTestAddrsWithDenom(app, ctx, 7, sdk.NewInt(1_000_0000_000), remotePoolDenom)

	// Remote pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:              poolID,
		Bech32PrefixValAddr: remoteBech32PrefixValAddr,
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

	// List pools
	pools := app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 1)

	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[0].PoolId,
		PaymentorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[0].LocalAddress), sdk.Coins{sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	payments := app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	payment, err := app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)

	unbondingTime := ctx.BlockTime().Add(-2 * time.Second)

	// Simulate that we reached the unbonding step already (done in previous test)
	app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: uatomAddresses[0].String(),
		ToAddress:        cosmosAccAddress,
		State:            millionstypes.WithdrawalState_IcaUnbonding,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
		UnbondingEndsAt:  &unbondingTime,
	})

	withdrawals := app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	suite.Require().Len(withdrawals, 1)

	app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
	})

	payments = app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	suite.Require().Len(payments, 0)

	withdrawal, err := app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)

	// Update status to simulate tx
	app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawal.PoolId, withdrawal.WithdrawalId, millionstypes.WithdrawalState_IbcTransfer, withdrawal.UnbondingEndsAt, false)

	// Simulate failed ackResponse AckResponseStatus_FAILURE
	err = app.MillionsKeeper.OnTransferWithdrawalToRecipientCompleted(ctx, withdrawal.GetPoolId(), withdrawal.GetWithdrawalId(), true)
	suite.Require().NoError(err)
	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)
	suite.Require().Equal(millionstypes.WithdrawalState_Failure, withdrawal.State)
	suite.Require().Equal(millionstypes.WithdrawalState_IbcTransfer, withdrawal.ErrorState)

	// Update status to simulate transfer from to local chain
	app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawal.PoolId, withdrawal.WithdrawalId, millionstypes.WithdrawalState_IbcTransfer, withdrawal.UnbondingEndsAt, false)
	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)

	// Simulate success ackResponse AckResponseStatus_Success
	err = app.MillionsKeeper.OnTransferWithdrawalToRecipientCompleted(ctx, withdrawal.GetPoolId(), withdrawal.GetWithdrawalId(), false)
	suite.Require().NoError(err)
	// There should be no more withdrawal if successfully processed
	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	suite.Require().Len(withdrawals, 0)
}

// TestWithdrawal_ProcessWithdrawal tests to process a withdrawal considering the destination address and pool zone tests
func (suite *KeeperTestSuite) TestWithdrawal_ProcessWithdrawal() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
	poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
	drawDelta1 := 1 * time.Hour
	uatomAddresses := apptesting.AddTestAddrsWithDenom(app, ctx, 7, sdk.NewInt(1_000_0000_000), remotePoolDenom)

	// Remote pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:              poolID,
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

	// List pools
	pools := app.MillionsKeeper.ListPools(ctx)
	suite.Require().Len(pools, 1)

	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[0].PoolId,
		PaymentorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
	})
	err := app.BankKeeper.SendCoins(ctx, uatomAddresses[0], sdk.MustAccAddressFromBech32(pools[0].LocalAddress), sdk.Coins{sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)

	payments := app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	payment, err := app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)

	unbondingTime := ctx.BlockTime().Add(-2 * time.Second)

	// Add withdrawal with native pool toAddress
	app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: uatomAddresses[0].String(),
		ToAddress:        cosmosAccAddress,
		State:            millionstypes.WithdrawalState_IcaUnbonding,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
		UnbondingEndsAt:  &unbondingTime,
	})

	withdrawals := app.MillionsKeeper.ListAccountWithdrawals(ctx, uatomAddresses[0])
	suite.Require().Len(withdrawals, 1)

	app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: uatomAddresses[0].String(),
		WinnerAddress:    uatomAddresses[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(remotePoolDenom, sdk.NewInt(1_000_000)),
	})

	payments = app.MillionsKeeper.ListAccountPayments(ctx, uatomAddresses[0])
	suite.Require().Len(payments, 0)

	withdrawal, err := app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)

	// Update status to simulate transfer from to local chain
	app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawal.PoolId, withdrawal.WithdrawalId, millionstypes.WithdrawalState_IbcTransfer, withdrawal.UnbondingEndsAt, false)

	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].WithdrawalId)
	suite.Require().NoError(err)
	pools = app.MillionsKeeper.ListPools(ctx)
	// Error is triggered as failed to retrieve open active channel port (intended error)
	// Hence BankSendFromNativeChain is triggered
	err = app.MillionsKeeper.TransferWithdrawalToRecipient(ctx, withdrawal.PoolId, withdrawal.WithdrawalId)
	suite.Require().Error(err)
	isLocalToAddress, _, err := pools[0].AccAddressFromBech32(withdrawal.ToAddress)
	suite.Require().NoError(err)
	suite.Require().Equal(isLocalToAddress, false)

	// Test local pool
	app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
		PoolId:      app.MillionsKeeper.GetNextPoolID(ctx),
		Denom:       localPoolDenom,
		NativeDenom: localPoolDenom,
		Validators: []millionstypes.PoolValidator{
			{
				OperatorAddress: suite.valAddrs[0].String(),
				BondedAmount:    sdk.NewInt(1_000_000),
				IsEnabled:       true,
			},
		},
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
	suite.Require().Len(pools, 2)

	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           pools[1].PoolId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pools[1].IcaPaymentAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])

	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)

	// Add withdrawal
	app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		ToAddress:        suite.addrs[0].String(),
		State:            millionstypes.WithdrawalState_IcaUnbonding,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
		UnbondingEndsAt:  &unbondingTime,
	})

	withdrawals = app.MillionsKeeper.ListAccountWithdrawals(ctx, suite.addrs[0])
	suite.Require().Len(withdrawals, 1)

	// Remove payment to respect the flow
	app.MillionsKeeper.RemovePayment(ctx, &millionstypes.Payment{
		PoolId:           payment.PoolId,
		PaymentId:        payment.PaymentId,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_Success,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})

	// There should be no more payments
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	suite.Require().Len(payments, 0)

	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].PaymentId)
	suite.Require().NoError(err)
	app.MillionsKeeper.UpdateWithdrawalStatus(ctx, withdrawal.PoolId, withdrawal.WithdrawalId, millionstypes.WithdrawalState_IbcTransfer, withdrawal.UnbondingEndsAt, false)
	withdrawal, err = app.MillionsKeeper.GetPoolWithdrawal(ctx, withdrawals[0].PoolId, withdrawals[0].PaymentId)
	suite.Require().NoError(err)

	// No Error should be triggered
	// Hence TransferToLocalChain is triggered for a local pool with local address
	err = app.MillionsKeeper.TransferWithdrawalToRecipient(ctx, withdrawal.PoolId, withdrawal.WithdrawalId)
	suite.Require().NoError(err)
	isLocalToAddress, _, err = pools[1].AccAddressFromBech32(withdrawal.ToAddress)
	suite.Require().NoError(err)
	suite.Require().Equal(isLocalToAddress, true)
}

// TestWithdrawal_DequeueMaturedWithdrawal tests the dequeue process of matured withdrawals
func (suite *KeeperTestSuite) TestWithdrawal_DequeueMaturedWithdrawal() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx

	// Add 4 withdrawals
	for i := 0; i < 4; i++ {
		poolID := app.MillionsKeeper.GetNextPoolIDAndIncrement(ctx)
		paymentID := app.MillionsKeeper.GetNextPaymentIdAndIncrement(ctx)
		drawDelta1 := 1 * time.Hour
		app.MillionsKeeper.AddPool(ctx, newValidPool(suite, millionstypes.Pool{
			PoolId:      poolID,
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
		}))

		unbondingTime := ctx.BlockTime().Add(2 * time.Second)

		// Status WithdrawalState_IcaUnbonding pushes directly into the queue
		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           poolID,
			PaymentId:        paymentID,
			State:            millionstypes.WithdrawalState_IcaUnbonding,
			PaymentorAddress: suite.addrs[0].String(),
			ToAddress:        suite.addrs[0].String(),
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(1_000_000))),
			UnbondingEndsAt:  &unbondingTime,
		})
	}

	// 1 second before the queue should be empty
	maturedQueue := app.MillionsKeeper.DequeueMaturedWithdrawalQueue(ctx, ctx.BlockTime().Add(time.Second))
	suite.Require().Len(maturedQueue, 0)

	// 2 seconds after the queue should contain the withdrawals
	maturedQueue = app.MillionsKeeper.DequeueMaturedWithdrawalQueue(ctx, ctx.BlockTime().Add(2*time.Second))
	suite.Require().Len(maturedQueue, 4)
}

// TestWithdrawal_BalanceWithdrawal test the paymentor balance after a transfer back to local chain
func (suite *KeeperTestSuite) TestWithdrawal_BalanceWithdrawal() {
	// Set the app context
	app := suite.app
	ctx := suite.ctx
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

	// Initialize the local pool
	pool := newValidPool(suite, millionstypes.Pool{
		PoolId:      1,
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

	// Initialize the balance
	balanceBefore := app.BankKeeper.GetBalance(ctx, suite.addrs[0], localPoolDenom)
	suite.Require().Equal(int64(1_000_000_000_0), balanceBefore.Amount.Int64())

	// Create payment
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           1,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_IbcTransfer,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	// Send coin
	err := app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pool.IcaPaymentAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	payments := app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	payment, err := app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[0].PaymentId)
	suite.Require().NoError(err)
	// Simulate the transfer to native chain
	err = app.MillionsKeeper.TransferPaymentToRemoteZone(ctx, payment.PoolId, payment.PaymentId)
	suite.Require().NoError(err)

	// Create payment
	app.MillionsKeeper.AddPayment(ctx, &millionstypes.Payment{
		PoolId:           1,
		PaymentorAddress: suite.addrs[0].String(),
		WinnerAddress:    suite.addrs[0].String(),
		State:            millionstypes.PaymentState_IbcTransfer,
		Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000)),
	})
	err = app.BankKeeper.SendCoins(ctx, suite.addrs[0], sdk.MustAccAddressFromBech32(pool.IcaPaymentAddress), sdk.Coins{sdk.NewCoin(localPoolDenom, sdk.NewInt(1_000_000))})
	suite.Require().NoError(err)
	payments = app.MillionsKeeper.ListAccountPayments(ctx, suite.addrs[0])
	payment, err = app.MillionsKeeper.GetPoolPayment(ctx, payments[0].PoolId, payments[1].PaymentId)
	suite.Require().NoError(err)
	// Simulate the transfer to native chain
	err = app.MillionsKeeper.TransferPaymentToRemoteZone(ctx, payment.PoolId, payment.PaymentId)
	suite.Require().NoError(err)

	for _, payment := range payments {
		// We set one unbondingEndsAt in the past and one in the future
		unbondingEndsAt := ctx.BlockTime().Add(-10 * time.Second)
		// Add the withdrawal with the modified unbonding time
		app.MillionsKeeper.AddWithdrawal(ctx, millionstypes.Withdrawal{
			PoolId:           1,
			PaymentId:        payment.PaymentId,
			State:            millionstypes.WithdrawalState_Pending,
			PaymentorAddress: suite.addrs[0].String(),
			ToAddress:        suite.addrs[0].String(),
			Amount:           sdk.NewCoin(localPoolDenom, sdk.NewInt(int64(1_000_000))),
			UnbondingEndsAt:  &unbondingEndsAt,
		})
	}

	// Test that there is 2 withdrawals
	withdrawals := app.MillionsKeeper.ListWithdrawals(ctx)
	suite.Require().Len(withdrawals, 2)

	for _, w := range withdrawals {
		err = app.MillionsKeeper.AddEpochUnbonding(ctx, w, false)
		suite.Require().NoError(err)
	}

	// Get the millions internal module tracker
	epochTracker, err := app.MillionsKeeper.GetEpochTracker(ctx, epochstypes.DAY_EPOCH, millionstypes.WithdrawalTrackerType)
	suite.Require().NoError(err)

	// Get epoch unbonding
	currentEpochUnbonding, err := app.MillionsKeeper.GetEpochPoolUnbonding(ctx, epochTracker.EpochNumber+frequency, 1)
	suite.Require().NoError(err)

	// Trigger undelegation flow
	err = app.MillionsKeeper.UndelegateWithdrawalsOnRemoteZone(ctx, currentEpochUnbonding)
	suite.Require().NoError(err)

	// Test that the balance should remain unchanged
	balance := app.BankKeeper.GetBalance(ctx, suite.addrs[0], localPoolDenom)
	suite.Require().Equal(balanceBefore.Amount.Int64()-2_000_000, balance.Amount.Int64())

	ctx = ctx.WithBlockTime(now.Add(21 * 24 * time.Hour))
	_, err = app.StakingKeeper.CompleteUnbonding(ctx, sdk.MustAccAddressFromBech32(pool.IcaPaymentAddress), suite.valAddrs[0])
	suite.Require().NoError(err)

	// There should be no withdrawals to dequeue
	maturedWithdrawals := app.MillionsKeeper.DequeueMaturedWithdrawalQueue(ctx, ctx.BlockTime())
	suite.Require().Len(maturedWithdrawals, 2)

	// Dequeue matured withdrawals and trigger the transfer to the local chain
	for i, mw := range maturedWithdrawals {
		app.MillionsKeeper.UpdateWithdrawalStatus(ctx, mw.GetPoolId(), mw.GetWithdrawalId(), millionstypes.WithdrawalState_IbcTransfer, withdrawals[i].UnbondingEndsAt, false)
		err = app.MillionsKeeper.TransferWithdrawalToRecipient(ctx, mw.GetPoolId(), mw.GetWithdrawalId())
		// Test that there should be no error
		suite.Require().NoError(err)
	}

	// Test that there should be no matured withdrawal left
	maturedWithdrawals = app.MillionsKeeper.DequeueMaturedWithdrawalQueue(ctx, ctx.BlockTime())
	suite.Require().Len(maturedWithdrawals, 0)

	// Test that the balance should compensate 2 payments transfered back
	balance = app.BankKeeper.GetBalance(ctx, suite.addrs[0], localPoolDenom)
	suite.Require().Equal(balanceBefore.Amount.Int64(), balance.Amount.Int64())
}
