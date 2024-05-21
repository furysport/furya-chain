#!/bin/bash

furyad10 tx gov submit-proposal software-upgrade "v2.0.0" \
--upgrade-height=100 \
--title="Upgrade to v2.0.0" --description="Upgrade to v2.0.0" \
--from=furya  \
--chain-id=furya-1 --yes -b block --deposit="100000000ufury" --fees 10000ufury --gas auto --gas-adjustment 1.5 --yes

furyad10 tx gov vote 1 yes --from furya --chain-id furya-1 \
-b block -y --fees 10000ufury --gas auto --gas-adjustment 1.5

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

furyad tx group create-group furya1f9eh8dh7j4nqe8nfq0lhpnr2elh5jr2w4nngt2 "" group-members.json \
 --from emergency 
 --chain-id=praetoria-1 --yes

# Check newly added queries
# - inflation
furyad query mint inflation
# - stakingAPR
furyad query mint staking-apr
# IBC transfer test
