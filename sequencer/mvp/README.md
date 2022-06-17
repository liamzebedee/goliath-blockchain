sequencer
=========

This is a permissioned P2P transaction sequencer that can scale to a large number of transactions per second.

A sequencer takes a transaction and gives it a sequence number, providing a total ordering for all txs. The underlying data structure is a cryptographically-authentified append-only log.

It is permissioned - only certain accounts are allowed to append to the sequence.

The node can operate in "primary" or "replica" mode. Operating as a replica means mirroring the content of the primary.

The primary signs every new sequenced tx, and can be punished for equivocation (signing two conflicting statements about the tx sequencing) through slashing on Ethereum L2.

## Reference.

 * JSON-RPC API over HTTP.
 * P2P replication using libp2p's EpiSub gossip protocol.

## RPC methods.

sequencer_append
sequencer_read
sequencer_info

## Usage.

```sh
# Generate a private key.
node -e "console.log(require('ethers').Wallet.createRandom().privateKey)"
# 0xd96a6cca804b24f540dc41ac3f50e2acd7510c33662c3040bafc07bc95b035ed

mkdir tmp

# Run the sequencer primary.
PRIVATE_KEY=0x0801124098bba74fbc32342624d74e8e523644be41d1e745b21af54933735ea6f0d92de17f7858dd065ece3d57a79a48b203664a63c356fb53c2dd3c5ce6a92aca4ebc39 go run cmd/sequencer/main.go -dbpath tmp/db -mode primary

# Run a sequencer replica.
PRIVATE_KEY="" go run cmd/sequencer/main.go -dbpath tmp/db2 -mode replica -peers "/ip4/192.168.1.189/tcp/24445/p2p/12D3KooWJPxP7QYvfkDoHRXFirAixtvmy3dMjy1eszPza7oFqdgt" -rpcport 25445 -p2pport 25446
```

## Philosophy.

This is a fully-fledged tx sequencer in around ~1000 LOC.

```
(base) ➜  mvp git:(master) ✗ find . -name '*.go' | xargs wc -l
      85 ./cmd/sequencer/main.go
     248 ./sequencer/p2p.go
     122 ./sequencer/core_test.go
     233 ./sequencer/core.go
      55 ./sequencer/utils/test_utils.go
      79 ./sequencer/messages.go
      49 ./sequencer/utils.go
      43 ./sequencer/rpc.go
      92 ./sequencer/node.go
      83 ./benchmarking/run.go
    1089 total
```


