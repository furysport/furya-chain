package types

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

const (
	ModuleName = "millions"

	ModuleVersion = 6

	StoreKey = ModuleName

	RouterKey = ModuleName

	QuerierRoute = ModuleName
)

const (
	IBCTimeoutNanos    = 1_800_000_000_000   // 30 minutes
	IBCForceRetryNanos = 259_200_000_000_000 // 3 days
)

const (
	UnknownID uint64 = 0
)

const (
	ICATypeCredit   = "credit"
	ICATypePrizePool = "prizepool"
)

var (
	NextPoolIdPrefix               = []byte{0x01}
	PoolPrefix                     = []byte{0x02}
	NextCreditPrefix              = []byte{0x03}
	PoolCreditPrefix              = []byte{0x04}
	AccountPoolCreditPrefix       = []byte{0x05}
	DrawPrefix                     = []byte{0x06}
	PoolPrizePrefix                = []byte{0x07}
	AccountPoolPrizePrefix         = []byte{0x08}
	NextPrizePrefix                = []byte{0x09}
	PoolWithdrawalPrefix           = []byte{0x10}
	AccountPoolWithdrawalPrefix    = []byte{0x11}
	NextWithdrawalPrefix           = []byte{0x12}
	PrizeExpirationTimePrefix      = []byte{0x13}
	WithdrawalMaturationTimePrefix = []byte{0x14}
	EpochUnbondingPrefix           = []byte{0x15}
	EpochTrackerPrefix             = []byte{0x16}
	ParamsPrefix                   = []byte{0x20}
	// KeyIndexSeparator separator between combined keys
	KeyIndexSeparator = []byte{0xFF}
)

// CombineKeys combine bytes array into a single bytes.
func CombineKeys(keys ...[]byte) []byte {
	return bytes.Join(keys, KeyIndexSeparator)
}

func CombineStringKeys(keys ...string) string {
	return strings.Join(keys, string(KeyIndexSeparator))
}

func DecombineStringKeys(combined string) []string {
	return strings.Split(combined, string(KeyIndexSeparator))
}

func StringKeyToBytes(key string) []byte {
	return []byte(key)
}

// Pool

func GetPoolKey(poolID uint64) []byte {
	return CombineKeys(PoolPrefix, strconv.AppendUint([]byte{}, poolID, 10))
}

func NewPoolName(poolID uint64, addressType string) []byte {
	key := []byte(fmt.Sprintf("pool.%d.%s", poolID, addressType))
	return key
}

func NewPoolAddress(poolID uint64, addressType string) sdk.AccAddress {
	return address.Module(ModuleName, NewPoolName(poolID, addressType))
}

// Credits

func GetCreditsKey() []byte {
	return PoolCreditPrefix
}

func GetPoolCreditKey(poolID uint64, creditID uint64) []byte {
	return CombineKeys(PoolCreditPrefix, strconv.AppendUint([]byte{}, poolID, 10), strconv.AppendUint([]byte{}, creditID, 10))
}

func GetPoolCreditsKey(poolID uint64) []byte {
	return CombineKeys(PoolCreditPrefix, strconv.AppendUint([]byte{}, poolID, 10))
}

func GetAccountPoolCreditKey(addr sdk.Address, poolID uint64, creditID uint64) []byte {
	return CombineKeys(AccountPoolCreditPrefix, addr.Bytes(), strconv.AppendUint([]byte{}, poolID, 10), strconv.AppendUint([]byte{}, creditID, 10))
}

func GetAccountPoolCreditsKey(addr sdk.Address, poolID uint64) []byte {
	return CombineKeys(AccountPoolCreditPrefix, addr.Bytes(), strconv.AppendUint([]byte{}, poolID, 10))
}

func GetAccountCreditsKey(addr sdk.Address) []byte {
	return CombineKeys(AccountPoolCreditPrefix, addr.Bytes())
}

// Prizes

func GetPrizesKey() []byte {
	return PoolPrizePrefix
}

func GetPoolPrizesKey(poolID uint64) []byte {
	return CombineKeys(PoolPrizePrefix, strconv.AppendUint([]byte{}, poolID, 10))
}

func GetPoolDrawPrizesKey(poolID uint64, drawID uint64) []byte {
	return CombineKeys(PoolPrizePrefix, strconv.AppendUint([]byte{}, poolID, 10), strconv.AppendUint([]byte{}, drawID, 10))
}

func GetPoolDrawPrizeKey(poolID uint64, drawID uint64, prizeID uint64) []byte {
	return CombineKeys(PoolPrizePrefix, strconv.AppendUint([]byte{}, poolID, 10), strconv.AppendUint([]byte{}, drawID, 10), strconv.AppendUint([]byte{}, prizeID, 10))
}

func GetAccountPrizesKey(addr sdk.Address) []byte {
	return CombineKeys(AccountPoolPrizePrefix, addr.Bytes())
}

func GetAccountPoolPrizesKey(addr sdk.Address, poolID uint64) []byte {
	return CombineKeys(AccountPoolPrizePrefix, addr.Bytes(), strconv.AppendUint([]byte{}, poolID, 10))
}

func GetAccountPoolDrawPrizesKey(addr sdk.Address, poolID uint64, drawID uint64) []byte {
	return CombineKeys(AccountPoolPrizePrefix, addr.Bytes(), strconv.AppendUint([]byte{}, poolID, 10), strconv.AppendUint([]byte{}, drawID, 10))
}

func GetAccountPoolDrawPrizeKey(addr sdk.Address, poolID uint64, drawID uint64, prizeID uint64) []byte {
	return CombineKeys(AccountPoolPrizePrefix, addr.Bytes(), strconv.AppendUint([]byte{}, poolID, 10), strconv.AppendUint([]byte{}, drawID, 10), strconv.AppendUint([]byte{}, prizeID, 10))
}

func GetExpiringPrizeTimeKey(timestamp time.Time) []byte {
	return CombineKeys(PrizeExpirationTimePrefix, sdk.FormatTimeBytes(timestamp))
}

// Draws

func GetPoolDrawsKey(poolID uint64) []byte {
	return CombineKeys(DrawPrefix, strconv.AppendUint([]byte{}, poolID, 10))
}

func GetPoolDrawIDKey(poolID uint64, drawID uint64) []byte {
	return CombineKeys(DrawPrefix, strconv.AppendUint([]byte{}, poolID, 10), strconv.AppendUint([]byte{}, drawID, 10))
}

// Withdrawals

func GetWithdrawalsKey() []byte {
	return PoolWithdrawalPrefix
}

func GetPoolWithdrawalKey(poolID uint64, withdrawalID uint64) []byte {
	return CombineKeys(PoolWithdrawalPrefix, strconv.AppendUint([]byte{}, poolID, 10), strconv.AppendUint([]byte{}, withdrawalID, 10))
}

func GetPoolWithdrawalsKey(poolID uint64) []byte {
	return CombineKeys(PoolWithdrawalPrefix, strconv.AppendUint([]byte{}, poolID, 10))
}

func GetAccountPoolWithdrawalKey(addr sdk.Address, poolID uint64, withdrawalID uint64) []byte {
	return CombineKeys(AccountPoolWithdrawalPrefix, addr.Bytes(), strconv.AppendUint([]byte{}, poolID, 10), strconv.AppendUint([]byte{}, withdrawalID, 10))
}

func GetAccountPoolWithdrawalsKey(addr sdk.Address, poolID uint64) []byte {
	return CombineKeys(AccountPoolWithdrawalPrefix, addr.Bytes(), strconv.AppendUint([]byte{}, poolID, 10))
}

func GetAccountWithdrawalsKey(addr sdk.Address) []byte {
	return CombineKeys(AccountPoolWithdrawalPrefix, addr.Bytes())
}

func GetMaturedWithdrawalTimeKey(timestamp time.Time) []byte {
	return CombineKeys(WithdrawalMaturationTimePrefix, sdk.FormatTimeBytes(timestamp))
}

// Epochs

func GetEpochPoolUnbondingKey(epochID uint64, poolID uint64) []byte {
	return CombineKeys(EpochUnbondingPrefix, strconv.AppendUint([]byte{}, epochID, 10), strconv.AppendUint([]byte{}, poolID, 10))
}

func GetEpochUnbondingsKey(epochID uint64) []byte {
	return CombineKeys(EpochUnbondingPrefix, strconv.AppendUint([]byte{}, epochID, 10))
}

func GetEpochTrackerKey(epochIdentifier string, trackertype string) []byte {
	return CombineKeys(EpochTrackerPrefix, StringKeyToBytes(epochIdentifier), StringKeyToBytes(trackertype))
}
