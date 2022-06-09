Roadmap
=======

The Goliath blockchain is a very new architecture, and as such, we want to deliver and validate pieces.

For the first milestone, we will be pushing out the first testnet release, called _"David"_. 

## Releases.

### Goliath 1.0 - David.

**Goal**: test the SputnikVM EVM implementation for bugs, test the throughput/capacity of the Goliath architecture, validate things work to the public (devs, VC's).

**Features**:

 - EVM-compatible blockchain with massive capacity. Sequencing decoupled from execution decoupled from storage.
 - Anyone can run a node and mirror the chain.
 - Centralized permissioned sequencer (a la Starkware, Optimism, etc.).
 - No gas or fee model (yet).

 - [x] implement stateless EVM - using SputnikVM
 - [x] connect EVM storage to some persistent database backend - SQL
 - [x] deploy ethereum contract to EVM **manually** (Hardhat's `Greeter.sol`)
 - [x] deploy contract using `seth` or `cast`
 - [x] implement basic Ethereum JSON-RPC node
 - [ ] implement a basic tendermint sequencer - submit tx, set time, 2 node BFT network.
 - [ ] implement the scheduler-executer - read historical + current txs from scheduler, execute them and write to db.
 - [ ] Update data model in SQLite to use sequencer timestamp as key.
 - [ ] return the new contract address
 - [ ] insert logs into db
 - [ ] add endpoint for fetching account nonce and balance
 - [ ] add backing implementation which uses a google cloud sqlite db
 - [ ] Deploy entire thing to Google Cloud.

### (future)

The design right now is decentralized at the sequencing layer, though as the blockchain gets larger, the execution/storage cost will increase for nodes (aka: the big block debates of bitcoin). There are two approaches to addressing this:

 1. **Garbage collecting old storage leaves** (ie. retaining only the latest values). There is no need to retain these for non-archive nodes, as there are no reorgs possible on the sequencer layer. This should save considerable cost. 
 2. **Decentralizing the storage layer**. How can we do this? Well, using STARK proofs, we can build a decentralised storage network, where each storage leaf stored by nodes is trustlessly proven to be the correct value based on a STARK proof of the transaction which set it. This is something that I've actively prototyped and is available in the [Quark blockchain repo here](https://github.com/liamzebedee/quark-blockchain).

Below are the list of remaining features before a Goliath mainnet release:

 - [ ] gas/fee model.
 - [ ] light client compatibility.
 - [ ] STARK proofs for updates to world state, a la [Quark](https://github.com/liamzebedee/quark-blockchain)'s design.
 - [ ] distributed execution network and scheduler.
 - [ ] decentralised Bigtable for storage.
 - [ ] garbage collect old slots from state / move them into cheaper storage.
 - [ ] more performant sequencer using Byzantine Atomic Broadcast.


#### Research areas.

 - Data availability network for chain data.
 - Fast sync, checkpointing.
 - Periodic STARK proofs of entire world state, committed to Ethereum.

