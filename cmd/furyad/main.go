package main

import (
	"os"

	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	app "github.com/furysport/furya-upgrade/app"
	appparams "github.com/furysport/furya-upgrade/app/params"
	"github.com/furysport/furya-upgrade/cmd/furyad/cmd"
)

func main() {
	appparams.SetAddressPrefixes()

	rootCmd, _ := cmd.NewRootCmd()
	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
