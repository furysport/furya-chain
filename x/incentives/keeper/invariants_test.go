package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	furyaapptesting "github.com/furysport/furya-chain/v2/app/testing"

	"github.com/furysport/furya-chain/v2/x/incentives/keeper"
)

func TestUnreleasedIncentivesInvariant(t *testing.T) {
	app := furyaapptesting.MakeSimpleMockApp()
	ctx := app.BaseApp.NewContext(false, tmproto.Header{})

	for _, mockSchedule := range mockSchedulesReleased {
		app.IncentivesKeeper.SetSchedule(ctx, mockSchedule)
	}

	invariant := keeper.TotalUnreleasedIncentives(app.IncentivesKeeper)

	// set incorrect balances for the incentives module account
	maccAddr := app.IncentivesKeeper.GetModuleAddress()
	app.BankKeeper.InitGenesis(
		ctx,
		&banktypes.GenesisState{
			Balances: []banktypes.Balance{{
				Address: maccAddr.String(),
				Coins:   sdk.NewCoins(sdk.NewCoin("ufury", sdk.NewInt(123)), sdk.NewCoin("uastro", sdk.NewInt(456))),
			}},
		},
	)

	msg, broken := invariant(ctx)
	require.True(t, broken)
	require.Equal(
		t,
		`incentives: total-unreleased-incentives invariant
	sum of unreleased incentives: 7192uastro,8637ufury
	module account balances: 456uastro,123ufury
`,
		msg,
	)

	// set the correct balances for the incentives module account
	app.BankKeeper.InitGenesis(
		ctx,
		&banktypes.GenesisState{
			Balances: []banktypes.Balance{{
				Address: maccAddr.String(),
				Coins:   sdk.NewCoins(sdk.NewCoin("ufury", sdk.NewInt(8637)), sdk.NewCoin("uastro", sdk.NewInt(7192))),
			}},
		},
	)

	_, broken = invariant(ctx)
	require.False(t, broken)
}
