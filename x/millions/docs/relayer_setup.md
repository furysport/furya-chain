## Starting the project

Before doing the following, please make sure you have a running instance of the furya binary, either via ignite or as a daemon

Run the docker compose in a separate terminal

`docker-compose -f tools/docker-compose.yml up -d`

Wait for the containers to be health, then the relayer will start, you can check this by doing:

`docker ps`

Once the containers are healthy, the relayer container will pop up.

## Create the links between chains

First, enter the furya-relayer container bash

Grab the furya-relayer container id by doing `docker ps` then do (and replace the id inside):

`docker exec -it b60875bae07a bash`

Once inside, you can type the following to create the links (repeat for each chain)

### Create link for Gaia <=> Furya

`hermes create connection --a-chain furya-millions-1 --b-chain gaia-devnet`

`hermes create channel --a-chain furya-millions-1 --a-connection connection-0 --a-port transfer --b-port transfer`

Transactions may fail but will succeed at some point.

### Create link Juno <=> Furya

`hermes create connection --a-chain furya-testnet --b-chain juno-devnet`

`hermes create channel --a-chain furya-testnet --a-connection connection-1 --a-port transfer --b-port transfer`

### Create link Osmosis <=> Furya

`hermes create connection --a-chain furya-testnet --b-chain osmosis-devnet`

`hermes create channel --a-chain furya-testnet --a-connection connection-2 --a-port transfer --b-port transfer`

## Sending coins from Gaia to local Furya one

Find the ID of the Gaia container:

`docker ps` then search for "furya-gaia"

Then enter the container (replace ID inside)

`docker exec -it 773da473109d bash`

Then type

`gaiad tx ibc-transfer transfer transfer channel-0 furya16rlynj5wvzwts5lqep0je5q4m3eaepn58hmzu5 1000000uatom --from genesis_key --keyring-backend test --chain-id gaia-devnet --fees 200uatom`

If you type 

`furya query bank balances furya1vvhncsjsdy2d7cnge2773kd49pkvvv9wsrt8ra`

You should see something like this with the IBC denom from Gaia

```
balances:
- amount: "1000000"
  denom: ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2
- amount: "100000000000"
  denom: uatom
- amount: "119900000000"
  denom: ufury
pagination:
  next_key: null
  total: "0"
```