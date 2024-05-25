package millions_test

import (
	"sort"
	"testing"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/stretchr/testify/require"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	apptypes "github.com/furysport/furya-chain/v2/app"
	epochstypes "github.com/furysport/furya-chain/v2/x/epochs/types"
	"github.com/furysport/furya-chain/v2/x/millions"
	millionstypes "github.com/furysport/furya-chain/v2/x/millions/types"
)

var (
	now      = time.Now().UTC()
	future   = now.Add(10 * time.Second)
	testAccs = []string{
		"furya19vp4wqsdyw8gftvkat0vrfqzvdrlp9wxp6fevr",
		"furya1s3zyjkd79tf83jew2f4cu4zhc9e5y53faqv3te",
		"furya1hag45rj53nfpj6357ql7nzq3p678s7lx3gpmyu",
		"furya1xutdwytajz6zn338wq7n4wx6m95076nepyu5zl",
		"furya10dtsxdl0uj0pw58yvlmsuy8p5mvzvewga5dgl8",
		"furya17p728nwzmkg5038pepfzeavxfu063aej0shwc8",
	}
	defaultValidators = []millionstypes.PoolValidator{{
		OperatorAddress: "furyavaloper1qx2dts3tglxcu0jh47k7ghstsn4nactufgmmlk",
		BondedAmount:    sdk.ZeroInt(),
	}}
	defaultSchedule   = millionstypes.DrawSchedule{DrawDelta: 404 * time.Hour, InitialDrawAt: now}
	defaultPrizeStrat = millionstypes.PrizeStrategy{PrizeBatches: []millionstypes.PrizeBatch{{PoolPercent: 100, Quantity: 1, DrawProbability: sdk.NewDec(1)}}}
)

var testGenesis = millionstypes.GenesisState{
	Params: millionstypes.Params{
		MinPaymentAmount:        sdk.NewInt(1_000_000),
		MaxPrizeStrategyBatches: 10,
		MaxPrizeBatchQuantity:   1000,
		MinDrawScheduleDelta:    1 * time.Hour,
		MaxDrawScheduleDelta:    366 * 24 * time.Hour, // 366 days
		PrizeExpirationDelta:    30 * 24 * time.Hour,  // 30 days
		MinPaymentDrawDelta:     5 * time.Minute,
	},
	NextPoolId:       6,
	NextPaymentId:    6,
	NextPrizeId:      6,
	NextWithdrawalId: 6,
	Pools: []millionstypes.Pool{
		{PoolId: 1, PoolType: millionstypes.PoolType_Staking, TvlAmount: sdk.NewInt(510), PaymentorsCount: 3, SponsorshipAmount: sdk.ZeroInt(), Denom: "denom-1", NativeDenom: "denom-1", NextDrawId: 3,
			ChainId: "c1", Validators: defaultValidators, MinPaymentAmount: sdk.NewInt(1_000_000), UnbondingDuration: time.Duration(millionstypes.DefaultUnbondingDuration), MaxUnbondingEntries: sdk.NewInt(millionstypes.DefaultMaxUnbondingEntries), AvailablePrizePool: sdk.NewCoin("denom-1", sdk.ZeroInt()),
			DrawSchedule: defaultSchedule, PrizeStrategy: defaultPrizeStrat,
			State: millionstypes.PoolState_Created, Bech32PrefixAccAddr: "furya", Bech32PrefixValAddr: "furyavaloper",
			FeeTakers: []millionstypes.FeeTaker{
				{Destination: authtypes.FeeCollectorName, Amount: sdk.NewDecWithPrec(millionstypes.DefaultFeeTakerAmount, 2), Type: millionstypes.FeeTakerType_LocalModuleAccount},
			},
		},
		{PoolId: 2, PoolType: millionstypes.PoolType_Staking, TvlAmount: sdk.NewInt(603), PaymentorsCount: 2, SponsorshipAmount: sdk.NewInt(401), Denom: "denom-2", NativeDenom: "denom-2", NextDrawId: 2,
			ChainId: "c1", Validators: defaultValidators, MinPaymentAmount: sdk.NewInt(1_000_000), UnbondingDuration: time.Duration(millionstypes.DefaultUnbondingDuration), MaxUnbondingEntries: sdk.NewInt(millionstypes.DefaultMaxUnbondingEntries), AvailablePrizePool: sdk.NewCoin("denom-2", sdk.ZeroInt()),
			DrawSchedule: defaultSchedule, PrizeStrategy: defaultPrizeStrat,
			State: millionstypes.PoolState_Ready, Bech32PrefixAccAddr: "furya", Bech32PrefixValAddr: "furyavaloper",
			FeeTakers: []millionstypes.FeeTaker{
				{Destination: authtypes.FeeCollectorName, Amount: sdk.NewDecWithPrec(millionstypes.DefaultFeeTakerAmount, 2), Type: millionstypes.FeeTakerType_LocalModuleAccount},
			},
		},
		{PoolId: 3, PoolType: millionstypes.PoolType_Staking, TvlAmount: sdk.NewInt(601), PaymentorsCount: 1, SponsorshipAmount: sdk.ZeroInt(), Denom: "denom-3", NativeDenom: "denom-3", NextDrawId: 1,
			ChainId: "c1", Validators: defaultValidators, MinPaymentAmount: sdk.NewInt(1_000_000), UnbondingDuration: time.Duration(millionstypes.DefaultUnbondingDuration), MaxUnbondingEntries: sdk.NewInt(millionstypes.DefaultMaxUnbondingEntries), AvailablePrizePool: sdk.NewCoin("denom-3", sdk.ZeroInt()),
			DrawSchedule: defaultSchedule, PrizeStrategy: defaultPrizeStrat,
			State: millionstypes.PoolState_Closed, Bech32PrefixAccAddr: "furya", Bech32PrefixValAddr: "furyavaloper",
			FeeTakers: []millionstypes.FeeTaker{
				{Destination: authtypes.FeeCollectorName, Amount: sdk.NewDecWithPrec(millionstypes.DefaultFeeTakerAmount, 2), Type: millionstypes.FeeTakerType_LocalModuleAccount},
			},
		},
		{PoolId: 4, PoolType: millionstypes.PoolType_Staking, TvlAmount: sdk.NewInt(400), PaymentorsCount: 1, SponsorshipAmount: sdk.ZeroInt(), Denom: "denom-4", NativeDenom: "denom-4", NextDrawId: 1,
			ChainId: "c1", Validators: defaultValidators, MinPaymentAmount: sdk.NewInt(1_000_000), UnbondingDuration: time.Duration(millionstypes.DefaultUnbondingDuration), MaxUnbondingEntries: sdk.NewInt(millionstypes.DefaultMaxUnbondingEntries), AvailablePrizePool: sdk.NewCoin("denom-4", sdk.ZeroInt()),
			DrawSchedule: defaultSchedule, PrizeStrategy: defaultPrizeStrat,
			State: millionstypes.PoolState_Created, Bech32PrefixAccAddr: "furya", Bech32PrefixValAddr: "furyavaloper",
			FeeTakers: []millionstypes.FeeTaker{
				{Destination: authtypes.FeeCollectorName, Amount: sdk.NewDecWithPrec(millionstypes.DefaultFeeTakerAmount, 2), Type: millionstypes.FeeTakerType_LocalModuleAccount},
			},
		},
		{PoolId: 5, PoolType: millionstypes.PoolType_Staking, TvlAmount: sdk.NewInt(0), PaymentorsCount: 0, SponsorshipAmount: sdk.ZeroInt(), Denom: "denom-5", NativeDenom: "denom-5", NextDrawId: 1,
			ChainId: "c1", Validators: defaultValidators, MinPaymentAmount: sdk.NewInt(1_000_000), UnbondingDuration: time.Duration(millionstypes.DefaultUnbondingDuration), MaxUnbondingEntries: sdk.NewInt(millionstypes.DefaultMaxUnbondingEntries), AvailablePrizePool: sdk.NewCoin("denom-5", sdk.ZeroInt()),
			DrawSchedule: defaultSchedule, PrizeStrategy: defaultPrizeStrat,
			State: millionstypes.PoolState_Closed, Bech32PrefixAccAddr: "furya", Bech32PrefixValAddr: "furyavaloper",
			FeeTakers: []millionstypes.FeeTaker{
				{Destination: authtypes.FeeCollectorName, Amount: sdk.NewDecWithPrec(millionstypes.DefaultFeeTakerAmount, 2), Type: millionstypes.FeeTakerType_LocalModuleAccount},
			},
		},
	},
	Payments: []millionstypes.Payment{
		{PoolId: 1, PaymentId: 1, PaymentorAddress: testAccs[0], WinnerAddress: testAccs[0], Amount: sdk.NewCoin("denom-1", sdk.NewInt(100)), State: millionstypes.PaymentState_IbcTransfer},
		{PoolId: 1, PaymentId: 2, PaymentorAddress: testAccs[0], WinnerAddress: testAccs[0], Amount: sdk.NewCoin("denom-1", sdk.NewInt(101)), State: millionstypes.PaymentState_IcaDelegate},
		{PoolId: 1, PaymentId: 3, PaymentorAddress: testAccs[1], WinnerAddress: testAccs[1], Amount: sdk.NewCoin("denom-1", sdk.NewInt(102)), State: millionstypes.PaymentState_Success},
		{PoolId: 1, PaymentId: 4, PaymentorAddress: testAccs[1], WinnerAddress: testAccs[1], Amount: sdk.NewCoin("denom-1", sdk.NewInt(103)), State: millionstypes.PaymentState_Failure},
		{PoolId: 1, PaymentId: 5, PaymentorAddress: testAccs[2], WinnerAddress: testAccs[2], Amount: sdk.NewCoin("denom-1", sdk.NewInt(104)), State: millionstypes.PaymentState_IbcTransfer},
		{PoolId: 2, PaymentId: 6, PaymentorAddress: testAccs[2], WinnerAddress: testAccs[2], Amount: sdk.NewCoin("denom-2", sdk.NewInt(200)), IsSponsor: true, State: millionstypes.PaymentState_IbcTransfer},
		{PoolId: 2, PaymentId: 7, PaymentorAddress: testAccs[3], WinnerAddress: testAccs[3], Amount: sdk.NewCoin("denom-2", sdk.NewInt(201)), IsSponsor: true, State: millionstypes.PaymentState_IbcTransfer},
		{PoolId: 2, PaymentId: 8, PaymentorAddress: testAccs[3], WinnerAddress: testAccs[3], Amount: sdk.NewCoin("denom-2", sdk.NewInt(202)), State: millionstypes.PaymentState_IbcTransfer},
		{PoolId: 3, PaymentId: 9, PaymentorAddress: testAccs[4], WinnerAddress: testAccs[4], Amount: sdk.NewCoin("denom-3", sdk.NewInt(300)), State: millionstypes.PaymentState_Success},
		{PoolId: 3, PaymentId: 10, PaymentorAddress: testAccs[4], WinnerAddress: testAccs[4], Amount: sdk.NewCoin("denom-3", sdk.NewInt(301)), State: millionstypes.PaymentState_Success},
		{PoolId: 4, PaymentId: 11, PaymentorAddress: testAccs[5], WinnerAddress: testAccs[5], Amount: sdk.NewCoin("denom-4", sdk.NewInt(400)), State: millionstypes.PaymentState_Success},
	},
	Draws: []millionstypes.Draw{
		{PoolId: 1, DrawId: 1, RandSeed: 10, TotalWinCount: 100, TotalWinAmount: sdk.NewInt(1000)},
		{PoolId: 1, DrawId: 2, RandSeed: 20, TotalWinCount: 200, TotalWinAmount: sdk.NewInt(2000)},
		{PoolId: 2, DrawId: 1, RandSeed: 30, TotalWinCount: 300, TotalWinAmount: sdk.NewInt(3000)},
	},
	Prizes: []millionstypes.Prize{
		{PoolId: 1, DrawId: 1, PrizeId: 1, State: millionstypes.PrizeState_Pending, WinnerAddress: testAccs[0], Amount: sdk.NewCoin("denom-1", sdk.NewInt(1)), ExpiresAt: future},
		{PoolId: 1, DrawId: 2, PrizeId: 2, State: millionstypes.PrizeState_Pending, WinnerAddress: testAccs[1], Amount: sdk.NewCoin("denom-1", sdk.NewInt(1)), ExpiresAt: now},
		{PoolId: 2, DrawId: 1, PrizeId: 3, State: millionstypes.PrizeState_Pending, WinnerAddress: testAccs[2], Amount: sdk.NewCoin("denom-2", sdk.NewInt(1)), ExpiresAt: now},
	},
	Withdrawals: []millionstypes.Withdrawal{
		{PoolId: 1, PaymentId: 3, WithdrawalId: 1, PaymentorAddress: testAccs[1], ToAddress: testAccs[1], Amount: sdk.NewCoin("denom-1", sdk.NewInt(102)), State: millionstypes.WithdrawalState_IcaUnbonding, UnbondingEndsAt: &now},
		{PoolId: 3, PaymentId: 9, WithdrawalId: 2, PaymentorAddress: testAccs[4], ToAddress: testAccs[4], Amount: sdk.NewCoin("denom-3", sdk.NewInt(300)), State: millionstypes.WithdrawalState_IcaUnbonding, UnbondingEndsAt: &future},
		{PoolId: 3, PaymentId: 10, WithdrawalId: 3, PaymentorAddress: testAccs[4], ToAddress: testAccs[4], Amount: sdk.NewCoin("denom-3", sdk.NewInt(301)), State: millionstypes.WithdrawalState_IcaUnbonding, UnbondingEndsAt: &future},
		{PoolId: 4, PaymentId: 11, WithdrawalId: 4, PaymentorAddress: testAccs[5], ToAddress: testAccs[5], Amount: sdk.NewCoin("denom-4", sdk.NewInt(400)), State: millionstypes.WithdrawalState_IcaUnbonding, UnbondingEndsAt: &now},
	},
	EpochTrackers: []millionstypes.EpochTracker{
		{EpochTrackerType: epochstypes.DAY_EPOCH, EpochIdentifier: epochstypes.DAY_EPOCH, NextEpochNumber: uint64(2), PreviousEpochNumber: uint64(0), NextEpochStartTime: future},
	},
	EpochUnbondings: []millionstypes.EpochUnbonding{
		{PoolId: 1, EpochIdentifier: "test", EpochNumber: 1, TotalAmount: sdk.NewCoin("ufury", sdk.NewInt(1000000)), WithdrawalIds: []uint64{1, 2}, WithdrawalIdsCount: 2},
	},
}

func TestInitGenesis(t *testing.T) {
	app := apptypes.SetupForTesting(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	ctx = ctx.WithBlockTime(now.Add(time.Second))

	// Init the module genesis state
	millions.InitGenesis(ctx, *app.MillionsKeeper, testGenesis)

	// Make sure params match
	params := app.MillionsKeeper.GetParams(ctx)
	require.Equal(t, testGenesis.Params, params)
	require.Equal(t, testGenesis.Params.MinPaymentAmount, params.MinPaymentAmount)
	require.Equal(t, testGenesis.Params.MaxPrizeStrategyBatches, params.MaxPrizeStrategyBatches)
	require.Equal(t, testGenesis.Params.MaxPrizeBatchQuantity, params.MaxPrizeBatchQuantity)
	require.Equal(t, testGenesis.Params.MinDrawScheduleDelta, params.MinDrawScheduleDelta)
	require.Equal(t, testGenesis.Params.MaxDrawScheduleDelta, params.MaxDrawScheduleDelta)
	require.Equal(t, testGenesis.Params.PrizeExpirationDelta, params.PrizeExpirationDelta)
	require.Equal(t, testGenesis.Params.MinPaymentDrawDelta, params.MinPaymentDrawDelta)

	// Make sure genesis next poolID matches
	nextPoolID := app.MillionsKeeper.GetNextPoolID(ctx)
	require.Equal(t, testGenesis.NextPoolId, nextPoolID)

	// Make sure genesis next paymentID matches
	nextPaymentID := app.MillionsKeeper.GetNextPaymentID(ctx)
	require.Equal(t, testGenesis.NextPaymentId, nextPaymentID)

	// Make sure genesis next prizeID matches
	nextPrizeID := app.MillionsKeeper.GetNextPrizeID(ctx)
	require.Equal(t, testGenesis.NextPrizeId, nextPrizeID)

	// Make sure genesis next withdrawalID matches
	nextWithdrawalID := app.MillionsKeeper.GetNextWithdrawalID(ctx)
	require.Equal(t, testGenesis.NextWithdrawalId, nextWithdrawalID)

	// Pools should all have been imported
	pools := app.MillionsKeeper.ListPools(ctx)
	require.Len(t, pools, len(testGenesis.Pools))
	for i, p := range pools {
		require.Equal(t, testGenesis.Pools[i].PoolId, p.PoolId)
		require.Equal(t, testGenesis.Pools[i].PoolType, p.PoolType)
		require.Equal(t, testGenesis.Pools[i].TvlAmount.Int64(), p.TvlAmount.Int64())
		require.Equal(t, testGenesis.Pools[i].PaymentorsCount, p.PaymentorsCount)
		require.Equal(t, testGenesis.Pools[i].SponsorshipAmount.Int64(), p.SponsorshipAmount.Int64())
		require.Equal(t, testGenesis.Pools[i].Denom, p.Denom)
		require.Equal(t, testGenesis.Pools[i].NextDrawId, p.NextDrawId)
		// Pool payments should all have been imported
		payments := app.MillionsKeeper.ListPoolPayments(ctx, p.PoolId)
		if p.PoolId == 1 {
			require.Len(t, payments, 5)
		} else if p.PoolId == 2 {
			require.Len(t, payments, 3)
		} else if p.PoolId == 3 {
			require.Len(t, payments, 2)
		} else if p.PoolId == 4 {
			require.Len(t, payments, 1)
		} else if p.PoolId == 5 {
			require.Len(t, payments, 0)
		}
	}

	// Payments should all have been imported
	payments := app.MillionsKeeper.ListPayments(ctx)
	require.Len(t, payments, len(testGenesis.Payments))
	sort.SliceStable(payments, func(i, j int) bool {
		return payments[i].PaymentId < payments[j].PaymentId
	})
	for i, d := range payments {
		require.Equal(t, testGenesis.Payments[i].PoolId, d.PoolId)
		require.Equal(t, testGenesis.Payments[i].PaymentId, d.PaymentId)
		require.Equal(t, testGenesis.Payments[i].PaymentorAddress, d.PaymentorAddress)
		require.Equal(t, testGenesis.Payments[i].WinnerAddress, d.WinnerAddress)
		require.Equal(t, testGenesis.Payments[i].Amount, d.Amount)
		require.Equal(t, testGenesis.Payments[i].IsSponsor, d.IsSponsor)
		require.Equal(t, testGenesis.Payments[i].State, d.State)
	}

	// Account payments should all have been imported
	for i := 0; i < len(testAccs); i++ {
		addr := sdk.MustAccAddressFromBech32(testAccs[i])
		payments := app.MillionsKeeper.ListAccountPayments(ctx, addr)
		if i == len(testAccs)-1 {
			require.Len(t, payments, 1)
		} else {
			require.Len(t, payments, 2)
		}
	}

	// Draws should have been imported
	draws := app.MillionsKeeper.ListDraws(ctx)
	require.Len(t, draws, len(testGenesis.Draws))
	for i, d := range draws {
		require.Equal(t, testGenesis.Draws[i].PoolId, d.PoolId)
		require.Equal(t, testGenesis.Draws[i].DrawId, d.DrawId)
		require.Equal(t, int64(10*(i+1)), d.RandSeed)
		require.Equal(t, uint64(100*(i+1)), d.TotalWinCount)
		require.Equal(t, sdk.NewInt(1000*int64(i+1)), d.TotalWinAmount)
	}

	// ListPoolDraws should have been imported
	for _, pd := range testGenesis.Draws {
		poolDraws := app.MillionsKeeper.ListPoolDraws(ctx, pd.PoolId)
		if pd.PoolId == 1 {
			require.Len(t, poolDraws, 2)
		} else {
			require.Len(t, poolDraws, 1)
		}
	}

	// Prizes should have been imported
	prizes := app.MillionsKeeper.ListPrizes(ctx)

	require.Len(t, prizes, len(testGenesis.Prizes))
	for i, pz := range prizes {
		require.Equal(t, testGenesis.Prizes[i].PoolId, pz.PoolId)
		require.Equal(t, testGenesis.Prizes[i].DrawId, pz.DrawId)
		require.Equal(t, testGenesis.Prizes[i].PrizeId, pz.PrizeId)
		require.Equal(t, testGenesis.Prizes[i].State, pz.State)
		require.Equal(t, testGenesis.Prizes[i].WinnerAddress, pz.WinnerAddress)
		require.Equal(t, testGenesis.Prizes[i].Amount, pz.Amount)
	}

	// ListAccountPrizes should have been imported
	for _, ap := range testGenesis.Prizes {
		addr := sdk.MustAccAddressFromBech32(ap.WinnerAddress)
		accPrizes := app.MillionsKeeper.ListAccountPrizes(ctx, addr)
		require.Len(t, accPrizes, 1)
	}

	// Test Expired Prizes Queue Import
	for i := 0; i < len(testGenesis.Withdrawals); i++ {
		col := app.MillionsKeeper.GetPrizeIDsEPCBQueue(ctx, now)
		require.Len(t, col.PrizesIds, 2)
		for i, pid := range col.PrizesIds {
			require.Equal(t, col.PrizesIds[i].PoolId, pid.PoolId)
			require.Equal(t, col.PrizesIds[i].PrizeId, pid.PrizeId)
		}
	}

	// Withdrawals
	withdrawals := app.MillionsKeeper.ListWithdrawals(ctx)
	require.Len(t, withdrawals, len(testGenesis.Withdrawals))
	for i, w := range withdrawals {
		require.Equal(t, testGenesis.Withdrawals[i].PoolId, w.PoolId)
		require.Equal(t, testGenesis.Withdrawals[i].PaymentId, w.PaymentId)
		require.Equal(t, testGenesis.Withdrawals[i].WithdrawalId, w.WithdrawalId)
		require.Equal(t, testGenesis.Withdrawals[i].WithdrawalId, w.WithdrawalId)
		require.Equal(t, testGenesis.Withdrawals[i].PaymentorAddress, w.PaymentorAddress)
		require.Equal(t, testGenesis.Withdrawals[i].ToAddress, w.ToAddress)
		require.Equal(t, testGenesis.Withdrawals[i].Amount, w.Amount)
		require.Equal(t, testGenesis.Withdrawals[i].State, w.State)
	}

	// Account withdrawals should have been imported
	for _, aw := range testGenesis.Withdrawals {
		addr := sdk.MustAccAddressFromBech32(aw.PaymentorAddress)
		accWithdrawals := app.MillionsKeeper.ListAccountWithdrawals(ctx, addr)
		if aw.PoolId == 1 {
			require.Len(t, accWithdrawals, 1)
		} else if aw.PoolId == 3 {
			require.Len(t, accWithdrawals, 2)
		} else {
			require.Len(t, accWithdrawals, 1)
		}
	}

	// Test Matured Withdrawal Queue Import
	for i := 0; i < len(testGenesis.Withdrawals); i++ {
		col := app.MillionsKeeper.GetWithdrawalIDsMaturedQueue(ctx, now)
		require.Len(t, col.WithdrawalsIds, 2)
		for i, wid := range col.WithdrawalsIds {
			require.Equal(t, col.WithdrawalsIds[i].PoolId, wid.PoolId)
			require.Equal(t, col.WithdrawalsIds[i].WithdrawalId, wid.WithdrawalId)
		}
	}

	epochTrackers := app.MillionsKeeper.ListEpochTrackers(ctx)
	require.Len(t, epochTrackers, len(testGenesis.EpochTrackers))
	for i, epochTracker := range epochTrackers {
		require.Equal(t, testGenesis.EpochTrackers[i].EpochIdentifier, epochTracker.EpochIdentifier)
		require.Equal(t, testGenesis.EpochTrackers[i].EpochNumber, epochTracker.EpochNumber)
		require.Equal(t, testGenesis.EpochTrackers[i].EpochTrackerType, epochTracker.EpochTrackerType)
		require.Equal(t, testGenesis.EpochTrackers[i].NextEpochNumber, epochTracker.NextEpochNumber)
		require.Equal(t, testGenesis.EpochTrackers[i].NextEpochStartTime, epochTracker.NextEpochStartTime)
	}

	// Make sure epoch unbondings have been imported
	epochUnbondings := app.MillionsKeeper.ListEpochUnbondings(ctx)
	require.Len(t, epochUnbondings, len(testGenesis.EpochUnbondings))
	for i, epochUnbonding := range epochUnbondings {
		require.Equal(t, testGenesis.EpochUnbondings[i].EpochIdentifier, epochUnbonding.EpochIdentifier)
		require.Equal(t, testGenesis.EpochUnbondings[i].EpochNumber, epochUnbonding.EpochNumber)
		require.Equal(t, testGenesis.EpochUnbondings[i].WithdrawalIds, epochUnbonding.WithdrawalIds)
		require.Equal(t, testGenesis.EpochUnbondings[i].WithdrawalIdsCount, epochUnbonding.WithdrawalIdsCount)
		require.Equal(t, testGenesis.EpochUnbondings[i].TotalAmount.Amount.Int64(), epochUnbonding.TotalAmount.Amount.Int64())
	}
}

func TestExportGenesis(t *testing.T) {
	app := apptypes.SetupForTesting(false)
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})
	ctx = ctx.WithBlockTime(now.Add(time.Second))

	millions.InitGenesis(ctx, *app.MillionsKeeper, testGenesis)
	exportGenesis := millions.ExportGenesis(ctx, *app.MillionsKeeper)

	// Test params export
	require.Equal(t, exportGenesis.Params, testGenesis.Params)
	require.Equal(t, exportGenesis.Params.MinPaymentAmount, testGenesis.Params.MinPaymentAmount)
	require.Equal(t, exportGenesis.Params.MaxPrizeStrategyBatches, testGenesis.Params.MaxPrizeStrategyBatches)
	require.Equal(t, exportGenesis.Params.MaxPrizeBatchQuantity, testGenesis.Params.MaxPrizeBatchQuantity)
	require.Equal(t, exportGenesis.Params.MinDrawScheduleDelta, testGenesis.Params.MinDrawScheduleDelta)
	require.Equal(t, exportGenesis.Params.MaxDrawScheduleDelta, testGenesis.Params.MaxDrawScheduleDelta)
	require.Equal(t, exportGenesis.Params.PrizeExpirationDelta, testGenesis.Params.PrizeExpirationDelta)
	require.Equal(t, exportGenesis.Params.MinPaymentDrawDelta, testGenesis.Params.MinPaymentDrawDelta)

	// Test IDs export
	require.Equal(t, exportGenesis.NextPoolId, testGenesis.NextPoolId)
	require.Equal(t, exportGenesis.NextPaymentId, testGenesis.NextPaymentId)
	require.Equal(t, exportGenesis.NextPrizeId, testGenesis.NextPrizeId)
	require.Equal(t, exportGenesis.NextWithdrawalId, testGenesis.NextWithdrawalId)

	// Test Pools export
	require.Len(t, exportGenesis.Pools, len(testGenesis.Pools))
	for i, p := range testGenesis.Pools {
		require.Equal(t, exportGenesis.Pools[i].PoolId, p.PoolId)
		require.Equal(t, exportGenesis.Pools[i].PoolType, p.PoolType)
		require.Equal(t, exportGenesis.Pools[i].TvlAmount.Int64(), p.TvlAmount.Int64())
		require.Equal(t, exportGenesis.Pools[i].PaymentorsCount, p.PaymentorsCount)
		require.Equal(t, exportGenesis.Pools[i].SponsorshipAmount.Int64(), p.SponsorshipAmount.Int64())
		require.Equal(t, exportGenesis.Pools[i].Denom, p.Denom)
		require.Equal(t, exportGenesis.Pools[i].NextDrawId, p.NextDrawId)
		// Pool payments should all have been exported
		payments := app.MillionsKeeper.ListPoolPayments(ctx, p.PoolId)
		if p.PoolId == 1 {
			require.Len(t, payments, 5)
		} else if p.PoolId == 2 {
			require.Len(t, payments, 3)
		} else if p.PoolId == 3 {
			require.Len(t, payments, 2)
		} else if p.PoolId == 4 {
			require.Len(t, payments, 1)
		} else if p.PoolId == 5 {
			require.Len(t, payments, 0)
		}
	}

	// Test payments export
	require.Len(t, exportGenesis.Payments, len(testGenesis.Payments))
	sort.SliceStable(exportGenesis.Payments, func(i, j int) bool {
		return exportGenesis.Payments[i].PaymentId < exportGenesis.Payments[j].PaymentId
	})
	for i, d := range testGenesis.Payments {
		require.Equal(t, exportGenesis.Payments[i].PoolId, d.PoolId)
		require.Equal(t, exportGenesis.Payments[i].PaymentId, d.PaymentId)
		require.Equal(t, exportGenesis.Payments[i].PaymentorAddress, d.PaymentorAddress)
		require.Equal(t, exportGenesis.Payments[i].WinnerAddress, d.WinnerAddress)
		require.Equal(t, exportGenesis.Payments[i].Amount, d.Amount)
		require.Equal(t, exportGenesis.Payments[i].IsSponsor, d.IsSponsor)
		require.Equal(t, exportGenesis.Payments[i].State, d.State)
	}

	// Test draws export
	require.Len(t, exportGenesis.Draws, len(testGenesis.Draws))
	for i, dw := range testGenesis.Draws {
		require.Equal(t, exportGenesis.Draws[i].PoolId, dw.PoolId)
		require.Equal(t, exportGenesis.Draws[i].DrawId, dw.DrawId)
		require.Equal(t, int64(10*(i+1)), dw.RandSeed)
		require.Equal(t, uint64(100*(i+1)), dw.TotalWinCount)
		require.Equal(t, sdk.NewInt(1000*int64(i+1)), dw.TotalWinAmount)
	}

	// Test prizes export
	require.Len(t, exportGenesis.Prizes, len(testGenesis.Prizes))
	for i, pz := range testGenesis.Prizes {
		require.Equal(t, exportGenesis.Prizes[i].PoolId, pz.PoolId)
		require.Equal(t, exportGenesis.Prizes[i].DrawId, pz.DrawId)
		require.Equal(t, exportGenesis.Prizes[i].PrizeId, pz.PrizeId)
		require.Equal(t, exportGenesis.Prizes[i].State, pz.State)
		require.Equal(t, exportGenesis.Prizes[i].WinnerAddress, pz.WinnerAddress)
		require.Equal(t, exportGenesis.Prizes[i].Amount, pz.Amount)
	}

	// Test withdrawals export
	require.Len(t, exportGenesis.Withdrawals, len(testGenesis.Withdrawals))
	for i, w := range testGenesis.Withdrawals {
		require.Equal(t, exportGenesis.Withdrawals[i].PoolId, w.PoolId)
		require.Equal(t, exportGenesis.Withdrawals[i].PaymentId, w.PaymentId)
		require.Equal(t, exportGenesis.Withdrawals[i].WithdrawalId, w.WithdrawalId)
		require.Equal(t, exportGenesis.Withdrawals[i].WithdrawalId, w.WithdrawalId)
		require.Equal(t, exportGenesis.Withdrawals[i].PaymentorAddress, w.PaymentorAddress)
		require.Equal(t, exportGenesis.Withdrawals[i].ToAddress, w.ToAddress)
		require.Equal(t, exportGenesis.Withdrawals[i].Amount, w.Amount)
		require.Equal(t, exportGenesis.Withdrawals[i].State, w.State)
	}

	require.Len(t, exportGenesis.EpochTrackers, len(testGenesis.EpochTrackers))
	for i, epochTracker := range testGenesis.EpochTrackers {
		require.Equal(t, testGenesis.EpochTrackers[i].EpochIdentifier, epochTracker.EpochIdentifier)
		require.Equal(t, testGenesis.EpochTrackers[i].EpochNumber, epochTracker.EpochNumber)
		require.Equal(t, testGenesis.EpochTrackers[i].EpochTrackerType, epochTracker.EpochTrackerType)
		require.Equal(t, testGenesis.EpochTrackers[i].NextEpochNumber, epochTracker.NextEpochNumber)
		require.Equal(t, testGenesis.EpochTrackers[i].NextEpochStartTime, epochTracker.NextEpochStartTime)
	}

	// Test epoch unbondings export
	require.Len(t, exportGenesis.EpochUnbondings, len(testGenesis.EpochUnbondings))
	for i, epochUnbonding := range testGenesis.EpochUnbondings {
		require.Equal(t, testGenesis.EpochUnbondings[i].EpochIdentifier, epochUnbonding.EpochIdentifier)
		require.Equal(t, testGenesis.EpochUnbondings[i].EpochNumber, epochUnbonding.EpochNumber)
		require.Equal(t, testGenesis.EpochUnbondings[i].WithdrawalIds, epochUnbonding.WithdrawalIds)
		require.Equal(t, testGenesis.EpochUnbondings[i].WithdrawalIdsCount, epochUnbonding.WithdrawalIdsCount)
		require.Equal(t, testGenesis.EpochUnbondings[i].TotalAmount.Amount.Int64(), epochUnbonding.TotalAmount.Amount.Int64())
	}
}
