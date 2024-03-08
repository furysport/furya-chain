package feeburn_test

import (
	"testing"

	keepertest "github.com/furysport/furya-chain/v2/testutil/keeper"
	"github.com/furysport/furya-chain/v2/testutil/nullify"

	"github.com/furysport/furya-chain/v2/x/feeburn"
	"github.com/furysport/furya-chain/v2/x/feeburn/types"
	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),
	}

	k, ctx := keepertest.FeeburnKeeper(t)
	feeburn.InitGenesis(ctx, *k, genesisState)
	got := feeburn.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)
}
