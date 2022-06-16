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

# Run the sequnecer.
PRIVATE_KEY=d96a6cca804b24f540dc41ac3f50e2acd7510c33662c3040bafc07bc95b035ed go run cmd/sequencer/main.go

# Run a sequencer mirror.
```