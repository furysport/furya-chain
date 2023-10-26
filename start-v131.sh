#!/bin/bash

rm -rf $HOME/.furyad/

cd $HOME

furyad131 init --chain-id=testing testing --home=$HOME/.furyad
furyad131 keys add validator --keyring-backend=test --home=$HOME/.furyad
furyad131 add-genesis-account $(furyad131 keys show validator -a --keyring-backend=test --home=$HOME/.furyad) 100000000000ufury,100000000000stake --home=$HOME/.furyad
furyad131 gentx validator 500000000stake --keyring-backend=test --home=$HOME/.furyad --chain-id=testing
furyad131 collect-gentxs --home=$HOME/.furyad

VALIDATOR=$(furyad131 keys show -a validator --keyring-backend=test --home=$HOME/.furyad)

sed -i '' -e 's/"owner": ""/"owner": "'$VALIDATOR'"/g' $HOME/.furyad/config/genesis.json
sed -i '' -e 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' $HOME/.furyad/config/app.toml 
sed -i '' -e 's/enable = false/enable = true/g' $HOME/.furyad/config/app.toml 
sed -i '' -e 's/cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/g' $HOME/.furyad/config/config.toml 
sed -i '' 's/"voting_period": "172800s"/"voting_period": "20s"/g' $HOME/.furyad/config/genesis.json

furyad131 start --home=$HOME/.furyad