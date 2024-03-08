package v1

import (
	"github.com/furysport/furya-chain/v2/app/upgrades"
)

// UpgradeName defines the on-chain upgrade name for the Furya v!.0.1 upgrade.
// this upgrade includes the fix for pfm
const UpgradeName = "v1.0.1"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
}
