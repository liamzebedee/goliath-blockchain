
sequencer
=========

Decentralized sequencer for Goliath PoC. 

Can process ~400 TPS using the Tendermint BFT consensus algorithm.

## Design.

### Data structures.

The sequencer stores a list of transactions, and notarises their time received. It's a distributed timestamp server (hey Satoshi). 

### Requirements.

The sequencer notarises transactions and assigns each one a timestamp, thus creating a total order.

 1. Must not require a token to use with the rest of Goliath.
 2. Must sequence at least 200 TPS.
 3. Must be Byzantine-Fault Tolerant.
 4. Must be able to be permissioned (eg. only authorised accounts can submit txs, for POC).

### Choice.

For the proof-of-concept, I've chosen Tendermint as a BFT-proof base, using the Cosmos framework to write a chain.

Why Tendermint? If the sequencer is just pretty much an append-only cryptographically authentifiable log, with a very minimal permissioning as to who can write to it, why not just use something like a [SecureScuttlebutt feed](https://ssbc.github.io/scuttlebutt-protocol-guide/) (the same MakerDAO uses for their price oracles)?

The advantage of Tendermint IMO is that it's production-ready decentralized infra. It's overkill in the long run, but in the short-term, we get networking, light clients, production-grade performance and knowledge (documentation, tooling). It's good for a PoC.

## Usage.

The sequencer is a public Tendermint blockchain with permissioned usage. Why permissioned? Because the POC doesn't have payment for txs yet, and I'm not paying for your ponzi lmao.

## Local development.

```sh
cd cosmos-sdk/
make install
cd build/

# Run the chain.
./simd --home ~/.simapp-node1 start
```





