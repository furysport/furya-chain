package v200

import (
	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/furysport/furya-upgrade/app/upgrades"
	// intertxtypes "github.com/furysport/furya-upgrade/x/intertx/types"
	// icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	// packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward/types"
	consensusparamtypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	alliancetypes "github.com/terra-money/alliance/x/alliance/types"
)

const (
	// UpgradeName defines the on-chain upgrade name.
	UpgradeName = "v2.0.0"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			// icacontrollertypes.StoreKey,
			// intertxtypes.StoreKey,
			// packetforwardtypes.StoreKey,
			crisistypes.StoreKey,
			consensusparamtypes.StoreKey,
			ibcfeetypes.StoreKey,
			group.StoreKey,
			alliancetypes.StoreKey,
		},
	},
}
