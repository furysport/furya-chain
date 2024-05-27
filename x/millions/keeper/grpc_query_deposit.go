package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/furysport/furya-chain/v2/x/millions/types"
)

func (k queryServer) Credits(goCtx context.Context, req *types.QueryCreditsRequest) (*types.QueryCreditsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Make sure context request is valid
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	store := ctx.KVStore(k.storeKey)
	creditStore := prefix.NewStore(store, types.GetCreditsKey())

	// Make the paginated query
	var credits []types.Credit
	pageRes, err := query.Paginate(creditStore, req.Pagination, func(key []byte, value []byte) error {
		var credit types.Credit
		if err := k.cdc.Unmarshal(value, &credit); err != nil {
			return err
		}

		credits = append(credits, credit)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryCreditsResponse{Credits: credits, Pagination: pageRes}, nil
}

func (k queryServer) PoolCredits(goCtx context.Context, req *types.QueryPoolCreditsRequest) (*types.QueryCreditsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Make sure context request is valid
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	store := ctx.KVStore(k.storeKey)
	creditStore := prefix.NewStore(store, types.GetPoolCreditsKey(req.GetPoolId()))

	if !k.HasPool(ctx, req.PoolId) {
		return nil, status.Error(codes.NotFound, types.ErrPoolNotFound.Error())
	}

	// Make the paginated query
	var credits []types.Credit
	pageRes, err := query.Paginate(creditStore, req.Pagination, func(key []byte, value []byte) error {
		var credit types.Credit
		if err := k.cdc.Unmarshal(value, &credit); err != nil {
			return err
		}

		credits = append(credits, credit)
		return nil
	})

	// Was there any error while acquiring the list of pools
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryCreditsResponse{Credits: credits, Pagination: pageRes}, nil
}

func (k queryServer) PoolCredit(goCtx context.Context, req *types.QueryPoolCreditRequest) (*types.QueryCreditResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Make sure context request is valid
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	credit, err := k.GetPoolCredit(ctx, req.GetPoolId(), req.GetCreditId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryCreditResponse{Credit: credit}, nil
}

func (k queryServer) AccountCredits(goCtx context.Context, req *types.QueryAccountCreditsRequest) (*types.QueryCreditsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Make sure context request is valid
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addr, err := sdk.AccAddressFromBech32(req.GetCreditorAddress())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	store := ctx.KVStore(k.storeKey)
	creditStore := prefix.NewStore(store, types.GetAccountCreditsKey(addr))

	// Make the paginated query
	var credits []types.Credit
	pageRes, err := query.Paginate(creditStore, req.Pagination, func(key []byte, value []byte) error {
		var credit types.Credit
		if err := k.cdc.Unmarshal(value, &credit); err != nil {
			return err
		}

		credits = append(credits, credit)
		return nil
	})

	// Was there any error while acquiring the list of pools
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryCreditsResponse{Credits: credits, Pagination: pageRes}, nil
}

func (k queryServer) AccountPoolCredits(goCtx context.Context, req *types.QueryAccountPoolCreditsRequest) (*types.QueryCreditsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Make sure context request is valid
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addr, err := sdk.AccAddressFromBech32(req.GetCreditorAddress())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if !k.HasPool(ctx, req.PoolId) {
		return nil, status.Error(codes.NotFound, types.ErrPoolNotFound.Error())
	}

	store := ctx.KVStore(k.storeKey)
	creditStore := prefix.NewStore(store, types.GetAccountPoolCreditsKey(addr, req.PoolId))

	// Make the paginated query
	var credits []types.Credit
	pageRes, err := query.Paginate(creditStore, req.Pagination, func(key []byte, value []byte) error {
		var credit types.Credit
		if err := k.cdc.Unmarshal(value, &credit); err != nil {
			return err
		}

		credits = append(credits, credit)
		return nil
	})

	// Was there any error while acquiring the list of pools
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryCreditsResponse{Credits: credits, Pagination: pageRes}, nil
}
