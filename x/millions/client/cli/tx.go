package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	gogotypes "github.com/cosmos/gogoproto/types"

	"github.com/furysport/furya-chain/v2/x/millions/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transaction subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdTxPayment(),
		CmdTxPaymentRetry(),
		CmdTxClaimPrize(),
		CmdTxDrawRetry(),
		CmdTxWithdrawPayment(),
		CmdTxWithdrawPaymentRetry(),
		CmdTxRestoreInterchainAccounts(),
		CmdTxPaymentEdit(),
		CmdTxGenerateSeed(),
	)
	return cmd
}

func CmdTxPayment() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "payment <pool_id> <amount>",
		Short: "Payment funds into a millions pool",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Payment funds into a millions pool.

Examples:
To create a classic payment
$ %s tx %s payment <pool_id> <amount>

To create a delegated payment (delegate drawing chances to another address)
$ %s tx %s payment <pool_id> <amount> --winner_address=<address>

To create a sponsorship payment (no drawing chances at all)
$ %s tx %s payment <pool_id> <amount> --sponsor=true`,
				version.AppName, types.ModuleName, version.AppName, types.ModuleName, version.AppName, types.ModuleName),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Acquire the client context
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			// Acquire the command arguments
			poolId, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			amount, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}

			// Acquire optional arguments
			winnerAddress, err := cmd.Flags().GetString("winner_address")
			if err != nil {
				return err
			}
			isSponsor, err := cmd.Flags().GetBool("sponsor")
			if err != nil {
				return err
			}

			// Build the message
			msg := types.NewMsgPayment(clientCtx.GetFromAddress().String(), amount, poolId)
			msg.WinnerAddress = winnerAddress
			msg.IsSponsor = isSponsor

			// Generate the transaction
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}
	cmd.Flags().String("winner_address", "", "(optional) winner address to direct the draw prizes to")
	cmd.Flags().Bool("sponsor", false, "(optional) active sponsor mode and waive this payment draw chances")
	flags.AddTxFlagsToCmd(cmd)
	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	return cmd
}

func CmdTxPaymentRetry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "payment-retry <pool_id> <payment_id>",
		Short: "Retry a failed payment",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Retry a payment stuck in a faulty state (ex: interchain tx issues).

Example:
$ %s tx %s payment-retry <pool_id> <payment_id>`,
				version.AppName, types.ModuleName),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Acquire the client context
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			// Acquire the command arguments
			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			paymentID, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			// Build the message
			msg := types.NewMsgPaymentRetry(clientCtx.GetFromAddress().String(), poolID, paymentID)

			// Generate the transaction
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	return cmd
}

func CmdTxPaymentEdit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "payment-edit <pool_id> <payment_id> [winner_address] [sponsor]",
		Short: "Edit a payment",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Edit a payment's winnerAddress to redirect the draw prize and/or sponsor mode that waives this payment draw chances.
Example:
To edit a delegated payment (delegate drawing chances to another address)
$ %s tx %s payment-edit <pool_id> <payment_id> --winner_address=<address>
To edit a sponsorship payment (no drawing chances at all)
$ %s tx %s payment-edit <pool_id> <payment_id> --sponsor=true
To edit a delegated payment and sponsorship
$ %s tx %s payment-edit <pool_id> <payment_id> --winner_address=<address> --sponsor=true`,
				version.AppName, types.ModuleName, version.AppName, types.ModuleName, version.AppName, types.ModuleName),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Acquire the client context
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			// Acquire the command arguments
			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			paymentID, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			// Acquire optional arguments
			winnerAddress, err := cmd.Flags().GetString("winner_address")
			if err != nil {
				return err
			}

			isSponsor, err := cmd.Flags().GetString("sponsor")
			if err != nil {
				return err
			}

			// Parse the boolean value from the string
			var isSponsorWrapper *gogotypes.BoolValue
			if isSponsor != "" {
				isSponsorBool, err := strconv.ParseBool(isSponsor)
				if err != nil {
					return err
				}
				isSponsorWrapper = &gogotypes.BoolValue{
					Value: isSponsorBool,
				}
			}

			// Build the message
			msg := types.NewMsgPaymentEdit(clientCtx.GetFromAddress().String(), poolID, paymentID)
			msg.WinnerAddress = winnerAddress
			msg.IsSponsor = isSponsorWrapper

			// Generate the transaction
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}

	cmd.Flags().String("winner_address", "", "(optional) winner address to direct the draw prizes to")
	cmd.Flags().String("sponsor", "", "(optional) active sponsor mode and waive this payment draw chances")
	flags.AddTxFlagsToCmd(cmd)
	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	return cmd
}

func CmdTxClaimPrize() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-prize <pool_id> <draw_id> <prize_id>",
		Short: "Claim a millions prize",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Claim a millions prize to send the funds to the prize winner address.

Example:
$ %s tx %s claim-prize <pool_id> <draw_id> <prize_id>`,
				version.AppName, types.ModuleName),
		),
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Acquire the client context
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			// Acquire the command arguments
			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			drawID, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			prizeID, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}

			// Build the message
			msg := types.NewMsgMsgClaimPrize(clientCtx.GetFromAddress().String(), poolID, drawID, prizeID)

			// Generate the transaction
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	return cmd
}

func CmdTxWithdrawPayment() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw-payment <pool_id> <payment_id>",
		Short: "Withdraw a payment",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Withdraw a payment.
Launches the unbonding of the payment and wait for its completion in order to send the funds to the specified withdrawal address.

Examples:
To withdraw a payment to the paymentor address
$ %s tx %s withdraw-payment <pool_id> <payment_id>

To withdraw a payment to another address
$ %s tx %s withdraw-payment <pool_id> <payment_id> --to_address=<address>`,
				version.AppName, types.ModuleName, version.AppName, types.ModuleName),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Acquire the client context
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			// Acquire the command arguments
			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			paymentID, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			destAddr, err := cmd.Flags().GetString("to_address")
			if err != nil {
				return err
			}
			if destAddr == "" {
				destAddr = clientCtx.GetFromAddress().String()
			}

			// Build the message
			msg := types.NewMsgWithdrawPayment(clientCtx.GetFromAddress().String(), destAddr, poolID, paymentID)
			msg.ToAddress = destAddr

			// Generate the transaction
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}
	cmd.Flags().String("to_address", "", "(optional) address to send the payment back")
	flags.AddTxFlagsToCmd(cmd)
	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	return cmd
}

func CmdTxWithdrawPaymentRetry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw-payment-retry <pool_id> <withdrawal_id>",
		Short: "Retry a failed withdrawal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Retry a withdrawal stuck in a faulty state (ex: interchain tx issues).

Example:
$ %s tx %s withdraw-payment-retry <pool_id> <withdrawal_id>`,
				version.AppName, types.ModuleName),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Acquire the client context
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			// Acquire the command arguments
			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			withdrawalID, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			// Build the message
			msg := types.NewMsgWithdrawPaymentRetry(clientCtx.GetFromAddress().String(), poolID, withdrawalID)

			// Generate the transaction
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	return cmd
}

func CmdTxDrawRetry() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "draw-retry <pool_id> <draw_id>",
		Short: "Retry a failed draw",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Retry a draw stuck in a faulty state (ex: interchain tx issues).
Draws are only launched by the protocol itself but can be retried (if in a faulty state) by anyone.

Example:
$ %s tx %s draw-retry <pool_id> <draw_id>`,
				version.AppName, types.ModuleName),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Acquire the client context
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			// Acquire the command arguments
			poolID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			drawID, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			// Build the message
			msg := types.NewMsgDrawRetry(clientCtx.GetFromAddress().String(), poolID, drawID)

			// Generate the transaction
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	return cmd
}

func CmdTxRestoreInterchainAccounts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore-interchain-accounts <pool_id>",
		Short: "Try to restore the interchain accounts of a given pool after one closed",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Acquire the client context
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			// Acquire the command arguments
			poolId, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			// Build the message
			msg := types.NewMsgRestoreInterchainAccounts(clientCtx.GetFromAddress().String(), poolId)

			// Generate the transaction
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	return cmd
}

func CmdTxGenerateSeed() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-seed",
		Short: "Generate a consensus-based random seed and return it",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Acquire the client context
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			txf, err := tx.NewFactoryCLI(clientCtx, cmd.Flags())
			if err != nil {
				return err
			}
			txf = txf.WithTxConfig(clientCtx.TxConfig).WithAccountRetriever(clientCtx.AccountRetriever)

			// Build the message
			msg := types.NewMsgGenerateSeed(clientCtx.GetFromAddress().String())

			// Generate the transaction
			return tx.GenerateOrBroadcastTxWithFactory(clientCtx, txf, msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	return cmd
}
