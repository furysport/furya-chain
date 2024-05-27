package millions

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/furysport/furya-chain/v2/x/millions/keeper"
	"github.com/furysport/furya-chain/v2/x/millions/types"
)

func NewHandler(k keeper.Keeper) sdk.Handler {
	msgServer := keeper.NewMsgServerImpl(k)

	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())
		goCtx := sdk.WrapSDKContext(ctx)

		switch msg := msg.(type) {
		case *types.MsgCredit:
			res, err := msgServer.Credit(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgCreditRetry:
			res, err := msgServer.CreditRetry(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgCreditEdit:
			res, err := msgServer.CreditEdit(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgClaimPrize:
			res, err := msgServer.ClaimPrize(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgDrawRetry:
			res, err := msgServer.DrawRetry(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgWithdrawCredit:
			res, err := msgServer.WithdrawCredit(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgWithdrawCreditRetry:
			res, err := msgServer.WithdrawCreditRetry(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgRestoreInterchainAccounts:
			res, err := msgServer.RestoreInterchainAccounts(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		case *types.MsgGenerateSeed:
			res, err := msgServer.GenerateSeed(goCtx, msg)
			return sdk.WrapServiceResult(ctx, res, err)
		default:
			errMsg := fmt.Sprintf("unrecognized %s message type: %T", types.ModuleName, msg)
			return nil, errorsmod.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}
	}
}
