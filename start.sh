#!/bin/bash

rm -rf $HOME/.furyad/

cd $HOME

furyad init --chain-id=testing testing --home=$HOME/.furyad
furyad keys add validator --keyring-backend=test --home=$HOME/.furyad
furyad add-genesis-account $(furyad keys show validator -a --keyring-backend=test --home=$HOME/.furyad) 100000000000ufury,100000000000stake --home=$HOME/.furyad
furyad gentx validator 500000000stake --keyring-backend=test --home=$HOME/.furyad --chain-id=testing
furyad collect-gentxs --home=$HOME/.furyad

VALIDATOR=$(furyad keys show -a validator --keyring-backend=test --home=$HOME/.furyad)

sed -i '' -e 's/"owner": ""/"owner": "'$VALIDATOR'"/g' $HOME/.furyad/config/genesis.json
sed -i '' -e 's/"voting_period": "172800s"/"voting_period": "20s"/g' $HOME/.furyad/config/genesis.json
sed -i '' -e 's/enabled-unsafe-cors = false/enabled-unsafe-cors = true/g' $HOME/.furyad/config/app.toml 
sed -i '' -e 's/enable = false/enable = true/g' $HOME/.furyad/config/app.toml 
sed -i '' -e 's/cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/g' $HOME/.furyad/config/config.toml 
jq '.app_state.gov.voting_params.voting_period = "20s"'  $HOME/.furyad/config/genesis.json > temp.json ; mv temp.json $HOME/.furyad/config/genesis.json;

furyad start --home=$HOME/.furyad

# git checkout v1.3.0
# go install ./cmd/furyad
# sh start.sh
# furyad tx gov submit-proposal software-upgrade "v1.4.0" --upgrade-height=12 --title="title" --description="description" --from=validator --keyring-backend=test --chain-id=testing --home=$HOME/.furyad/ --yes --broadcast-mode=block --deposit="100000000stake"
# furyad tx gov vote 1 Yes --from=validator --keyring-backend=test --chain-id=testing --home=$HOME/.furyad/ --yes  --broadcast-mode=block
# furyad query gov proposals
# git checkout ica_controller
# go install ./cmd/furyad
# furyad start
# furyad query interchain-accounts controller params
