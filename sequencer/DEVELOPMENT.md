# Development.

Local development guide.

## Design overview.



## Chain setup.

This guide is a WIP.

```sh
cd cosmos-sdk/build
./simd tendermint unsafe-reset-all


# Keys
# This is for node 1 and 2.
# TODO: refactor to be non-interactive.
./simd --home ~/.simapp-node1 keys add key1
./simd --home ~/.simapp-node2 keys add key2

# =======================
# Setup the nodes.
# =======================

# 
# Node 1.
# 

# Initialize chain.
./simd --home ~/.simapp-node1 init foo --chain-id goliath-david-01

# Setup genesis state.
./simd --home ~/.simapp-node1 add-genesis-account key1 10000000000000000000000000stake
./simd --home ~/.simapp-node1 add-genesis-account key2 10000000000000000000000000stake

# Create genesis tx for validator.
./simd --home ~/.simapp-node1 gentx key1 1000000000000stake --chain-id goliath-david-01

# Genesis transaction written to "/Users/liamz/.simapp-node1/config/gentx/gentx-8533efe42cb70a546b94649eee9e9d39faf9c51c.json"


# 
# Node 2.
# 

# Initialize chain.
./simd --home ~/.simapp-node2 init foo --chain-id goliath-david-01

# Setup genesis state.
./simd --home ~/.simapp-node2 add-genesis-account key1 10000000000000000000000000stake
./simd --home ~/.simapp-node2 add-genesis-account key2 10000000000000000000000000stake

# Create genesis tx for validator.
./simd --home ~/.simapp-node2 gentx key2 1000000000000stake --chain-id goliath-david-01




# =======================
# Create the genesis.
# =======================


# Gather all genesis transactions for each node, to create the super-genesis.
# Gather all under ~/.simapp-node2/config/gentx
cp ~/.simapp-node2/config/gentx/*.json ~/.simapp-node1/config/gentx

./simd --home ~/.simapp-node1 collect-gentxs

# Copy the supergenesis file to each node's config.
cp ~/.simapp-node1/config/genesis.json ~/.simapp-node2/config/genesis.json



# =======================
# Configure the nodes.
# =======================


# Get each node's validator address.
# 
./simd --home ~/.simapp-node1 tendermint show-node-id
# 05eb20a8fd2cbe4c68ba0bbd203ab49edbf76882

./simd --home ~/.simapp-node2 tendermint show-node-id
# 8c6f3d9ce4bd766f6a5bf44f93fabae8fc584e2a


# Now configure node 2 as a persistent peer for node 1.
# 
code ~/.simapp-node1/config/config.toml
# persistent-peers = "NODE_2_VALIDATOR_ADDRESS@localhost:36656"

# Lastly, configure node 2 to use different ports to node 1.
# Otherwise, we can't run both on one machine!
# 


# Values I changed for a two node local network:
# These are located in configs/node2/config.toml
# 
# in config.toml:
# - pprof-laddr
# - rpc.laddr
# - p2p.laddr
# - proxy-app
# 
# on the CLI:
# - grpc.address
# - grpc-web.address
# 



# Now start the two nodes.
./simd --home ~/.simapp-node1 start
./simd --home ~/.simapp-node2 start --grpc.address 0.0.0.0:19090 --grpc-web.address 0.0.0.0:19091
```