package keeper

import (
	"github.com/furysport/furya-chain/v2/x/icacallbacks/types"
)

var _ types.QueryServer = Keeper{}
