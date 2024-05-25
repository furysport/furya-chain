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

func (k queryServer) Payments(goCtx context.Context, req *types.QueryPaymentsRequest) (*types.QueryPaymentsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Make sure context request is valid
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	store := ctx.KVStore(k.storeKey)
	paymentStore := prefix.NewStore(store, types.GetPaymentsKey())

	// Make the paginated query
	var payments []types.Payment
	pageRes, err := query.Paginate(paymentStore, req.Pagination, func(key []byte, value []byte) error {
		var payment types.Payment
		if err := k.cdc.Unmarshal(value, &payment); err != nil {
			return err
		}

		payments = append(payments, payment)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryPaymentsResponse{Payments: payments, Pagination: pageRes}, nil
}

func (k queryServer) PoolPayments(goCtx context.Context, req *types.QueryPoolPaymentsRequest) (*types.QueryPaymentsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Make sure context request is valid
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	store := ctx.KVStore(k.storeKey)
	paymentStore := prefix.NewStore(store, types.GetPoolPaymentsKey(req.GetPoolId()))

	if !k.HasPool(ctx, req.PoolId) {
		return nil, status.Error(codes.NotFound, types.ErrPoolNotFound.Error())
	}

	// Make the paginated query
	var payments []types.Payment
	pageRes, err := query.Paginate(paymentStore, req.Pagination, func(key []byte, value []byte) error {
		var payment types.Payment
		if err := k.cdc.Unmarshal(value, &payment); err != nil {
			return err
		}

		payments = append(payments, payment)
		return nil
	})

	// Was there any error while acquiring the list of pools
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryPaymentsResponse{Payments: payments, Pagination: pageRes}, nil
}

func (k queryServer) PoolPayment(goCtx context.Context, req *types.QueryPoolPaymentRequest) (*types.QueryPaymentResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Make sure context request is valid
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	payment, err := k.GetPoolPayment(ctx, req.GetPoolId(), req.GetPaymentId())
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryPaymentResponse{Payment: payment}, nil
}

func (k queryServer) AccountPayments(goCtx context.Context, req *types.QueryAccountPaymentsRequest) (*types.QueryPaymentsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Make sure context request is valid
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addr, err := sdk.AccAddressFromBech32(req.GetPaymentorAddress())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	store := ctx.KVStore(k.storeKey)
	paymentStore := prefix.NewStore(store, types.GetAccountPaymentsKey(addr))

	// Make the paginated query
	var payments []types.Payment
	pageRes, err := query.Paginate(paymentStore, req.Pagination, func(key []byte, value []byte) error {
		var payment types.Payment
		if err := k.cdc.Unmarshal(value, &payment); err != nil {
			return err
		}

		payments = append(payments, payment)
		return nil
	})

	// Was there any error while acquiring the list of pools
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryPaymentsResponse{Payments: payments, Pagination: pageRes}, nil
}

func (k queryServer) AccountPoolPayments(goCtx context.Context, req *types.QueryAccountPoolPaymentsRequest) (*types.QueryPaymentsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Make sure context request is valid
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	addr, err := sdk.AccAddressFromBech32(req.GetPaymentorAddress())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if !k.HasPool(ctx, req.PoolId) {
		return nil, status.Error(codes.NotFound, types.ErrPoolNotFound.Error())
	}

	store := ctx.KVStore(k.storeKey)
	paymentStore := prefix.NewStore(store, types.GetAccountPoolPaymentsKey(addr, req.PoolId))

	// Make the paginated query
	var payments []types.Payment
	pageRes, err := query.Paginate(paymentStore, req.Pagination, func(key []byte, value []byte) error {
		var payment types.Payment
		if err := k.cdc.Unmarshal(value, &payment); err != nil {
			return err
		}

		payments = append(payments, payment)
		return nil
	})

	// Was there any error while acquiring the list of pools
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryPaymentsResponse{Payments: payments, Pagination: pageRes}, nil
}
