package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	millionstypes "github.com/furysport/furya-chain/v2/x/millions/types"
)

func (suite *KeeperTestSuite) TestParams_Validation() {
	app := suite.app
	ctx := suite.ctx

	params := app.MillionsKeeper.GetParams(ctx)
	suite.Require().NoError(params.ValidateBasics())

	// MinPaymentAmount should always be gte than MinAcceptablePaymentAmount
	params.MinPaymentAmount = sdk.NewInt(millionstypes.MinAcceptablePaymentAmount - 1)
	suite.Require().Error(params.ValidateBasics())
	params.MinPaymentAmount = sdk.NewInt(millionstypes.MinAcceptablePaymentAmount)
	suite.Require().NoError(params.ValidateBasics())

	// MaxPrizeStrategyBatches should always be gt 0
	params.MaxPrizeStrategyBatches = 0
	suite.Require().Error(params.ValidateBasics())
	params.MaxPrizeStrategyBatches = 1
	suite.Require().NoError(params.ValidateBasics())

	// MaxPrizeBatchQuantity should always be gt 0
	params.MaxPrizeBatchQuantity = 0
	suite.Require().Error(params.ValidateBasics())
	params.MaxPrizeBatchQuantity = 1
	suite.Require().NoError(params.ValidateBasics())

	// MinDrawScheduleDelta should always be gte MinAcceptableDrawDelta
	params.MinDrawScheduleDelta = millionstypes.MinAcceptableDrawDelta - 1
	suite.Require().Error(params.ValidateBasics())
	params.MinDrawScheduleDelta = millionstypes.MinAcceptableDrawDelta
	suite.Require().NoError(params.ValidateBasics())

	// MaxDrawScheduleDelta should always be gte MinDrawScheduleDelta
	params.MaxDrawScheduleDelta = params.MinDrawScheduleDelta - 1
	suite.Require().Error(params.ValidateBasics())
	params.MaxDrawScheduleDelta = params.MinDrawScheduleDelta
	suite.Require().NoError(params.ValidateBasics())

	// PrizeExpirationDelta should always be gte MinAcceptablePrizeExpirationDelta
	params.PrizeExpirationDelta = millionstypes.MinAcceptablePrizeExpirationDelta - 1
	suite.Require().Error(params.ValidateBasics())
	params.PrizeExpirationDelta = millionstypes.MinAcceptablePrizeExpirationDelta
	suite.Require().NoError(params.ValidateBasics())

	// MinPaymentDrawDelta should always be gte MinAcceptablePaymentDrawDelta
	params.MinPaymentDrawDelta = millionstypes.MinAcceptablePaymentDrawDelta - 1
	suite.Require().Error(params.ValidateBasics())
	params.MinPaymentDrawDelta = millionstypes.MinAcceptablePaymentDrawDelta
	suite.Require().NoError(params.ValidateBasics())
}
