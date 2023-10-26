package cli

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	appparams "github.com/furysport/fury-chain/app/params"
	airdroptypes "github.com/furysport/fury-chain/x/airdrop/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

func parseStarsAirdropAmount(path string) [][]string {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}

	return records
}

// AllocateStarsAirdropCmd returns allocate further airdrop cobra Command.
func AllocateStarsAirdropCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "allocate-stars-airdrop [airdrop_file_path] [start_index] [msgs_per_tx]",
		Short: "Allocate stars airdrop",
		Long: `Allocate stars airdrop.
Example:
	furyad tx airdrop allocate-stars-airdrop Airdrop_HuahuaPunks_Feuille_1.csv 0 500 --from=validator --keyring-backend=test --chain-id=testing --home=$HOME/.furyad/ --yes --broadcast-mode=block --gas=10000000
`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			newAllocations := []airdroptypes.AirdropAllocation{}
			allocationRecords := parseStarsAirdropAmount(args[0])
			for _, line := range allocationRecords[1:] {
				starsAddr, amountStr := line[0], line[1]
				amountDec := sdk.MustNewDecFromStr(amountStr)
				amount := amountDec.Mul(sdk.NewDec(1000_000)).TruncateInt()

				params := &airdroptypes.QueryAllocationRequest{Address: starsAddr}

				allocation := airdroptypes.AirdropAllocation{
					Chain:         "stargaze",
					Address:       starsAddr,
					Amount:        sdk.NewInt64Coin(appparams.BaseCoinUnit, 0),
					ClaimedAmount: sdk.NewInt64Coin(appparams.BaseCoinUnit, 0),
				}

				// get previous allocation if exists
				queryClient := airdroptypes.NewQueryClient(clientCtx)
				res, err := queryClient.Allocation(context.Background(), params)
				if err == nil && res.Allocation != nil {
					allocation = *res.Allocation
				}

				allocation.Amount.Amount = allocation.Amount.Amount.Add(amount)
				newAllocations = append(newAllocations, allocation)
			}

			startIndex, err := strconv.Atoi(args[1])
			if err != nil {
				return err
			}
			threshold, err := strconv.Atoi(args[2])
			if err != nil {
				return err
			}

			msgs := []sdk.Msg{}
			for index, allocation := range newAllocations {
				if index < startIndex {
					continue
				}
				msg := airdroptypes.NewMsgSetAllocation(
					clientCtx.FromAddress.String(),
					allocation,
				)
				msgs = append(msgs, msg)
				if len(msgs) >= threshold || index+1 == len(newAllocations) {
					err := tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msgs...)
					if err != nil {
						return err
					}
					fmt.Println("executed until index", index)
					msgs = []sdk.Msg{}
				}
			}

			fmt.Println("finalized execution of stars airdrop")

			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	_ = cmd.MarkFlagRequired(flags.FlagFrom)

	return cmd
}
