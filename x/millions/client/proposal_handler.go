package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"

	"github.com/furysport/furya-chain/x/millions/client/cli"
)

var (
	RegisterPoolProposalHandler = govclient.NewProposalHandler(cli.CmdProposalRegisterPool)
	UpdatePoolProposalHandler   = govclient.NewProposalHandler(cli.CmdProposalUpdatePool)
	ClosePoolProposalHandler    = govclient.NewProposalHandler(cli.CmdProposalClosePool)
	UpdateParamsProposalHandler = govclient.NewProposalHandler(cli.CmdProposalUpdateParams)
)
