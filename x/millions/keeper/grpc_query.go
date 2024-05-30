package keeper

import (
	"github.com/furysport/furya-chain/x/millions/types"
)

type queryServer struct {
	Keeper
}

// NewQueryServerImpl returns an implementation of the QueryServer interface
// for the provided Keeper.
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}
