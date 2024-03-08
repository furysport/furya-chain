# Furya Mainnet

This mainnet will start with the node version `b4fbc78`.

## Minimum hardware requirements

- 8-16GB RAM
- 100GB of disk space
- 2 cores

## Genesis Instruction

### Install node

```bash
git clone https://github.com/furysport/furya-chain
cd furya-chain
git checkout b4fbc78
make install
```

### Check Node version

```bash
# Get node version (should be b4fbc78)
furyad version

# Get node long version (should be ???)
furyad version --long | grep commit
```

### Initialize Chain

```bash
furyad init MONIKER --chain-id=furya-1
```

Set minimum gas price to 0.

```bash
sed -i 's/minimum-gas-prices = ".*"/minimum-gas-prices = "0ufury"/' $HOME/.furyad/config/app.toml
```

### Download pre-genesis

```bash
curl -s https://raw.githubusercontent.com/furysport/furya-chain/release/v1.0.x/networks/mainnet/pre-genesis.json > ~/.furyad/config/genesis.json
```

## Create gentx

Create wallet

```bash
furyad keys add KEY_NAME
```

Fund yourself `25000000ufury`

```bash
furyad add-genesis-account $(furyad keys show KEY_NAME -a) 25000000ufury
```

Use half (`10000000ufury`) for self-delegation

```bash
furyad gentx KEY_NAME 10000000ufury --chain-id=furya-1
```

If all goes well, you will see a message similar to the following:

```bash
Genesis transaction written to "/home/user/.furyad/config/gentx/gentx-******.json"
```

### Submit genesis transaction

- Fork this repo
- Copy the generated gentx json file to `networks/mainnet/gentx/`
- Commit and push to your repo
- Create a PR on this repo
