package keeper_test

import (
	"testing"

	testkeeper "github.com/furysport/furya-chain/v2/testutil/keeper"

	"github.com/furysport/furya-chain/v2/x/feeburn/types"
	"github.com/stretchr/testify/require"
)

func TestGetParams(t *testing.T) {
	k, ctx := testkeeper.FeeburnKeeper(t)
	params := types.DefaultParams()

	err := k.SetParams(ctx, params)
	require.NoError(t, err)
	require.EqualValues(t, params, k.GetParams(ctx))
}
