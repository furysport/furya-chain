package v201

import (
	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/furysport/furya-chain/v2/app/upgrades"
	// intertxtypes "github.com/furysport/furya-chain/v2/x/intertx/types"
	// icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	// packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward/types"
	// 	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	// 	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	// 	"github.com/cosmos/cosmos-sdk/x/group"
	// 	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	// 	tokenfactorytypes "github.com/furysport/furya-chain/v2/x/tokenfactory/types"
	// 	alliancemoduletypes "github.com/terra-money/alliance/x/alliance/types"
	"github.com/lum-network/chain/x/epochs"
	"github.com/lum-network/chain/x/icacallbacks"
	icacallbackstypes "github.com/lum-network/chain/x/icacallbacks/types"
	"github.com/lum-network/chain/x/icqueries"
	icqueriestypes "github.com/lum-network/chain/x/icqueries/types"
	"github.com/lum-network/chain/x/millions"
	millionstypes "github.com/lum-network/chain/x/millions/types"
	epochstypes "github.com/lum-network/chain/x/epochs/types"

	
)

const (
	// UpgradeName defines the on-chain upgrade name.
	UpgradeName = "v2.0.1"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			// icacontrollertypes.StoreKey,
			// intertxtypes.StoreKey,
			// packetforwardtypes.StoreKey,
			icacallbackstypes.StoreKey,
			icqueriestypes.StoreKey,
			millionstypes.StoreKey,
			epochstypes.StoreKey,
		},
	},
}
