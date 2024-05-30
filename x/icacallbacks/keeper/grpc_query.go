package keeper

import (
	"github.com/furysport/furya-chain/x/icacallbacks/types"
)

var _ types.QueryServer = Keeper{}
