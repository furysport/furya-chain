package client

import (
	"github.com/furysport/furya-chain/v2/x/feeburn/client/cli"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var UpdateTxFeeBurnPercentProposalHandler = govclient.NewProposalHandler(cli.NewUpdateTxFeeBurnPercentProposalHandler)
