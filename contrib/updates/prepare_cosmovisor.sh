#!/bin/bash

# this bash will prepare cosmosvisor to the build folder so that it can perform upgrade
# this script is supposed to be run by Makefile

# These fields should be fetched automatically in the future
# Need to do more upgrade to see upgrade patterns
OLD_VERSION=v3.0.4
# this command will retrieve the folder with the largest number in format v<number>
SOFTWARE_UPGRADE_NAME="v4.1.0"
BUILDDIR=$1
TESTNET_NVAL=$2
TESTNET_CHAINID=$3

# check if BUILDDIR is set
if [ -z "$BUILDDIR" ]; then
    echo "BUILDDIR is not set"
    exit 1
fi

# install old binary if not exist
if [ ! -f "_build/$OLD_VERSION.zip" ] &> /dev/null
then
    mkdir -p _build/old
    wget -c "https://github.com/furysport/furya-chain/archive/refs/tags/${OLD_VERSION}.zip" -O _build/${OLD_VERSION}.zip
    unzip _build/${OLD_VERSION}.zip -d _build
fi


if [ ! -f "$BUILDDIR/old/furyad" ] &> /dev/null
then
    mkdir -p BUILDDIR/old
    if [ ! "$(docker images -q furya/furyad.binary.old 2> /dev/null)" ]; then
        docker build --platform linux/amd64 --no-cache --build-arg source=./_build/furya-chain-${OLD_VERSION:1}/ --tag furya/furyad.binary.old ./_build/furya-chain-${OLD_VERSION:1}
    fi
    docker create --platform linux/amd64 --name old-temp furya/furyad.binary.old:latest
    mkdir -p $BUILDDIR/old
    docker cp old-temp:/usr/bin/furyad $BUILDDIR/old/
    docker rm old-temp
fi


# prepare cosmovisor config in TESTNET_NVAL nodes
if [ ! -f "$BUILDDIR/node0/furyad/config/genesis.json" ]; then docker run --rm \
    -v $BUILDDIR:/furyad:Z \
    --platform linux/amd64 \
    --entrypoint /furyad/old/furyad \
    furya/furyad-upgrade-env testnet init-files --v $TESTNET_NVAL --chain-id $TESTNET_CHAINID -o . --starting-ip-address 192.168.10.2 --minimum-gas-prices "0stake" --node-daemon-home furyad --keyring-backend=test --home=temp; \
fi

for (( i=0; i<$TESTNET_NVAL; i++ )); do
    CURRENT=$BUILDDIR/node$i/furyad

    # change gov params voting_period
    jq '.app_state.gov.voting_params.voting_period = "50s"' $CURRENT/config/genesis.json > $CURRENT/config/genesis.json.tmp && mv $CURRENT/config/genesis.json.tmp $CURRENT/config/genesis.json

    docker run --rm \
        -v $BUILDDIR:/furyad:Z \
        -e DAEMON_HOME=/furyad/node$i/furyad \
        -e DAEMON_NAME=furyad \
        -e DAEMON_RESTART_AFTER_UPGRADE=true \
        --entrypoint /furyad/cosmovisor \
        --platform linux/amd64 \
        furya/furyad-upgrade-env init /furyad/old/furyad
    mkdir -p $CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/bin
    cp $BUILDDIR/furyad $CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/bin
    touch $CURRENT/cosmovisor/upgrades/$SOFTWARE_UPGRADE_NAME/upgrade-info.json
done