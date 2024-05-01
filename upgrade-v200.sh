#!/bin/bash

furyad140 tx gov submit-proposal software-upgrade "v2.0.0" \
--upgrade-height=150 \
--title="Upgrade to v2.0.0" --description="Upgrade to v2.0.0" \
--from=devfurya  \
--chain-id=devfurya-1 --home=$HOME/.furyad --yes -b block --deposit="100000000ufury" --fees 10000ufury --gas auto --gas-adjustment 1.5 --yes

furyad140 tx gov vote 1 yes --from devfurya --chain-id devfurya-1 \
--home $HOME/.furyad -b block -y --fees 10000ufury --gas auto --gas-adjustment 1.5

furyad140 query gov proposals

sleep 50

killall furyad140 &> /dev/null || true

furyad start --home=$HOME/.furyad

# Check mint module params update
furyad query mint params

# Check packet forward queries
furyad query packetforward params

# Check group module tx
furyad query group groups
VALIDATOR=$(furyad keys show -a validator --home $HOME/.furyad --keyring-backend=test)

furyad tx group create-group furya1uxqcel9xcdmx7rwfjgfrmdmzgmn7q3jql3cvhz "" group-members.json \
 --from validator --chain-id testing --keyring-backend=test \
 --chain-id=testing --home=$HOME/.furyad --yes -b sync

# Check newly added queries
# - inflation
furyad query mint inflation
# - stakingAPR
furyad query mint staking-apr
# IBC transfer test
