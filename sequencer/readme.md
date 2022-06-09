
sequencer
=========

Decentralized sequencer for Goliath PoC. 

Can process ~400 TPS using the Tendermint BFT consensus algorithm.

## Design.
### Requirements.

 1. Must not require a token to use with the rest of Goliath.
 2. Must sequence at least 200 TPS.
 3. Must be Byzantine-Fault Tolerant.

### Choice.

For the proof-of-concept, I've chosen Tendermint as a BFT-proof base, using the Cosmos framework to write a chain.

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





