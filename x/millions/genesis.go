package millions

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/furysport/furya-chain/v2/x/millions/keeper"
	"github.com/furysport/furya-chain/v2/x/millions/types"
)

func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	k.SetParams(ctx, genState.Params)
	k.SetNextPoolID(ctx, genState.NextPoolId)
	k.SetNextCreditID(ctx, genState.NextCreditId)
	k.SetNextPrizeID(ctx, genState.NextPrizeId)
	k.SetNextWithdrawalID(ctx, genState.NextWithdrawalId)

	for _, pool := range genState.Pools {
		// Voluntary reset TvlAmount, CreditorsCount and SponsorshipAmount since they will be recomputed from credits
		pool.TvlAmount = sdk.ZeroInt()
		pool.CreditorsCount = 0
		pool.SponsorshipAmount = sdk.ZeroInt()
		k.AddPool(ctx, &pool)
	}

	// Import credits
	for _, credit := range genState.Credits {
		k.AddCredit(ctx, &credit)
	}

	// Import draws
	for _, draw := range genState.Draws {
		k.SetPoolDraw(ctx, draw)
	}

	// Import prizes
	for _, prize := range genState.Prizes {
		k.AddPrize(ctx, prize)
	}

	// Import withdrawals
	for _, withdrawal := range genState.Withdrawals {
		k.AddWithdrawal(ctx, withdrawal)
	}

	// Import epoch trackers
	for _, epochTracker := range genState.EpochTrackers {
		k.SetEpochTracker(ctx, epochTracker, types.WithdrawalTrackerType)
	}

	for _, epochUnbonding := range genState.EpochUnbondings {
		k.SetEpochUnbonding(ctx, epochUnbonding)
	}
}

func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	params := k.GetParams(ctx)
	return &types.GenesisState{
		Params:           params,
		NextPoolId:       k.GetNextPoolID(ctx),
		NextCreditId:    k.GetNextCreditID(ctx),
		NextPrizeId:      k.GetNextPrizeID(ctx),
		NextWithdrawalId: k.GetNextWithdrawalID(ctx),
		Pools:            k.ListPools(ctx),
		Credits:         k.ListCredits(ctx),
		Draws:            k.ListDraws(ctx),
		Prizes:           k.ListPrizes(ctx),
		Withdrawals:      k.ListWithdrawals(ctx),
		EpochTrackers:    k.ListEpochTrackers(ctx),
		EpochUnbondings:  k.ListEpochUnbondings(ctx),
	}
}
