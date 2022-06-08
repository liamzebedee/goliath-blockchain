

 - [ ] return the new contract address
 - [ ] insert logs into db
 - [ ] add endpoint for fetching account nonce and balance
 - [ ] add backing implementation which uses a google cloud sqlite db



Roadmap:

 * **Compression**. Google uses an extremely fast compression algo called Snappy throughout its infra - Bigtable databases, RPC calls, etc. [Calldata compression](https://github.com/ethereum-optimism/optimistic-specs/issues/10) is a well-applied in the Optimism rollup. 
 * **Parallel execution engine**. For the POC, the executor is a single node. There is a design here for a parallel executor, which uses the state dependencies of a transaction to schedule nonconflicting tx's in parallel. 
 * **Distributed execution network**. On top of the parallel executor, we can distribute execution to a network of executors (as one executor is likely capped by it's number of CPU's), further increasing throughput.
 * **Decentralized storage layer**. Based on a Google Bigtable design, implemented using P2P tech. Recursive STARK proofs + data availability tech.
 * **STARK-based execution proofs**. We can guarantee the writes to storage are authorised by running the transaction inside a provable VM, such as Cairo. 

