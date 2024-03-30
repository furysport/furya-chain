#!/usr/bin/env bash

furyad query bank balances $(furyad keys show -a validator --keyring-backend=test)
furyad tx mint burn-tokens 500000000stake --keyring-backend=test --from=validator --chain-id=testing --home=$HOME/.furyad/ --yes  --broadcast-mode=block
furyad query bank balances $(furyad keys show -a validator --keyring-backend=test)
