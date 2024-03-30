#!/usr/bin/env bash

furyad tx nftstaking register-nft-staking --from validator --nft-identifier "identifier3" --nft-metadata "metadata" --reward-address "pop1snktzg6rrncqtct3acx2vz60aak2a6fke3ny3c" --reward-weight 1000 --chain-id=testing --home=$HOME/.furyad --keyring-backend=test --broadcast-mode=block --yes
furyad tx nftstaking set-nft-type-perms NFT_TYPE_DEFAULT SET_SERVER_ACCESS --from=validator --chain-id=testing --home=$HOME/.furyad --keyring-backend=test --broadcast-mode=block --yes
furyad tx nftstaking set-access-info $(furyad keys show -a validator --keyring-backend=test) server1#chan1#chan2,server2#chan3 --from=validator --chain-id=testing --home=$HOME/.furyad --keyring-backend=test --broadcast-mode=block --yes

furyad query bank balances pop1uef5c6tx7vhjyhfumhzdhvwkepshcmljyv4wh4
furyad query nftstaking access-infos
furyad query nftstaking access-info $(furyad keys show -a validator --keyring-backend=test)
furyad query nftstaking all-nfttype-perms
furyad query nftstaking has-permission $(furyad keys show -a validator --keyring-backend=test) aaa
furyad query nftstaking nfttype-perms aaa
furyad query nftstaking staking aaa
furyad query nftstaking stakings
furyad query nftstaking stakings_by_owner $(furyad keys show -a validator --keyring-backend=test)

