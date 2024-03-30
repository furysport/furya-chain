#!/bin/bash

rm -rf $HOME/.furyad/

cd $HOME

furyad140 init --chain-id=testing testing --home=$HOME/.furyad
furyad140 keys add validator --keyring-backend=test --home=$HOME/.furyad
furyad140 add-genesis-account $(furyad140 keys show validator -a --keyring-backend=test --home=$HOME/.furyad) 100000000000ufury,100000000000stake --home=$HOME/.furyad
furyad140 gentx validator 500000000stake --keyring-backend=test --home=$HOME/.furyad --chain-id=testing
furyad140 collect-gentxs --home=$HOME/.furyad

VALIDATOR=$(furyad140 keys show -a validator --keyring-backend=test --home=$HOME/.furyad)

sed -i '' -e 's/"owner": ""/"owner": "'$VALIDATOR'"/g' $HOME/.furyad/config/genesis.json
sed -i '' -e 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' $HOME/.furyad/config/app.toml 
sed -i '' -e 's/enable = false/enable = true/g' $HOME/.furyad/config/app.toml 
sed -i '' -e 's/cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/g' $HOME/.furyad/config/config.toml 
sed -i '' 's/"voting_period": "172800s"/"voting_period": "20s"/g' $HOME/.furyad/config/genesis.json

furyad140 start --home=$HOME/.furyad