package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/spf13/cobra"
)

func writeCSV(path string, records [][]string) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	csvWriter := csv.NewWriter(f)
	err = csvWriter.WriteAll(records)
	if err != nil {
		panic(err)
	}
}

// GenesisState is minimum structure to import airdrop accounts
type GenesisState struct {
	AppState AppState `json:"app_state"`
}

// AppState is app state structure for app state
type AppState struct {
	Staking interface{}
	Bank    interface{}
}

// SnapshotAccount provide fields of snapshot per account
type SnapshotAccount struct {
	Address       string
	Balance       sdk.Int
	StakedBalance sdk.Int
	TotalBalance  sdk.Int
}

// ExportRichestSnapshotCmd generates a snapshot.csv from a provided chain genesis export.
func ExportRichestSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-richest-snapshot [genesis-file] [output-snapshot-csv]",
		Short: "Export richest snapshot from genesis export",
		Long: `Export richest snapshot from genesis export
Example:
	furyad export-richest-snapshot ./snapshot-furya-richest.json ./snapshot-furya-richest.csv
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			codec := clientCtx.Codec

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			config.SetRoot(clientCtx.HomeDir)

			genesisFile := args[0]
			snapshotOutput := args[1]

			// exclude module accounts
			excludeAddrs := make(map[string]bool)
			excludeAddrs["furya17xpfvakm2amg962yls6f84z3kell8c5llvck70"] = true
			excludeAddrs["furya1jv65s3grqf6v6jl3dp4t6c9t9rk99cd8g2l2ud"] = true
			excludeAddrs["furya1vlthgax23ca9syk7xgaz347xmf4nunefgadpez"] = true
			excludeAddrs["furya1zw7guf74ez4mlmsxlt0kcgg9yj6hx94z4fff5v"] = true
			excludeAddrs["furya1m3h30wlvsf8llruxtpukdvsy0km2kum88yuwjj"] = true
			excludeAddrs["furya1tygms3xhhs3yv487phx3dw4a95jn7t7lwwwg63"] = true
			excludeAddrs["furya1yl6hdjhmkf37639730gffanpzndzdpmhp2dlz3"] = true
			excludeAddrs["furya1fl48vsnmsdzcv85q5d2q4z5ajdha8yu36wjev9"] = true
			excludeAddrs["furya10d07y265gmmuvt4z0w9aw880jnsr700j4hgnrp"] = true

			decimal := sdk.NewDec(1000_000)

			// Read genesis file
			genesisJson, err := os.Open(genesisFile)
			if err != nil {
				return err
			}
			defer genesisJson.Close()

			byteValue, _ := ioutil.ReadAll(genesisJson)

			var genState GenesisState

			err = json.Unmarshal(byteValue, &genState)
			if err != nil {
				return err
			}

			bankBytes, err := json.Marshal(genState.AppState.Bank)
			if err != nil {
				panic(err)
			}
			bankGen := banktypes.GenesisState{}
			codec.MustUnmarshalJSON(bankBytes, &bankGen)

			// Produce the map of address to total atom balance, both staked and unstaked
			snapshotAccs := make(map[string]SnapshotAccount)

			for _, balance := range bankGen.Balances {
				acc, ok := snapshotAccs[balance.Address]
				if !ok {
					acc = SnapshotAccount{
						Address:       balance.Address,
						Balance:       sdk.ZeroInt(),
						StakedBalance: sdk.ZeroInt(),
					}
				}
				acc.Balance = balance.Coins.AmountOf("ufury")
				snapshotAccs[balance.Address] = acc
			}

			stakingBytes, err := json.Marshal(genState.AppState.Staking)
			if err != nil {
				panic(err)
			}
			stakingGen := stakingtypes.GenesisState{}
			codec.MustUnmarshalJSON(stakingBytes, &stakingGen)

			// Make a map from validator operator address to the  validator type
			validators := make(map[string]stakingtypes.Validator)
			for _, validator := range stakingGen.Validators {
				validators[validator.OperatorAddress] = validator
			}

			for _, delegation := range stakingGen.Delegations {
				address := delegation.DelegatorAddress
				if excludeAddrs[address] == true {
					continue
				}

				acc, ok := snapshotAccs[address]
				if !ok {
					acc = SnapshotAccount{
						Address:       delegation.DelegatorAddress,
						Balance:       sdk.ZeroInt(),
						StakedBalance: sdk.ZeroInt(),
					}
				}

				val := validators[delegation.ValidatorAddress]
				stakedAmount := delegation.Shares.MulInt(val.Tokens).Quo(val.DelegatorShares).RoundInt()
				acc.StakedBalance = acc.StakedBalance.Add(stakedAmount)
				snapshotAccs[address] = acc
			}

			for _, acc := range snapshotAccs {
				acc.TotalBalance = acc.Balance.Add(acc.StakedBalance)
				snapshotAccs[acc.Address] = acc
			}

			// iterate to find Osmo ownership percentage per account
			for address, acc := range snapshotAccs {
				snapshotAccs[address] = acc
			}

			sortedArr := []SnapshotAccount{}
			for _, acc := range snapshotAccs {
				if acc.TotalBalance.LT(sdk.NewInt(1000_000_000)) {
					continue
				}
				sortedArr = append(sortedArr, acc)
			}
			fmt.Println("totalSort", len(sortedArr))
			sort.Slice(sortedArr, func(i, j int) bool {
				return sortedArr[i].TotalBalance.GT(sortedArr[j].TotalBalance)
			})

			csvRecords := [][]string{{"address", "balance", "staked", "total"}}
			for _, acc := range sortedArr {
				csvRecords = append(csvRecords, []string{
					acc.Address,
					acc.Balance.ToDec().Quo(decimal).String(),
					acc.StakedBalance.ToDec().Quo(decimal).String(),
					acc.TotalBalance.ToDec().Quo(decimal).String(),
				})
			}

			writeCSV(snapshotOutput, csvRecords)

			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
