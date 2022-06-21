sequencer
=========

This is a permissioned P2P transaction sequencer that can scale to a large number of transactions per second.

A sequencer takes a transaction and gives it a sequence number, providing a total ordering for all txs. The underlying data structure is a cryptographically-authentified append-only log.

It is permissioned - only certain accounts are allowed to append to the sequence.

The node can operate in "primary" or "replica" mode. Operating as a replica means mirroring the content of the primary.

The primary signs every new sequenced tx, and can be punished for equivocation (signing two conflicting statements about the tx sequencing) **through slashing on Ethereum L2**.

## Reference.

 * JSON-RPC API over HTTP.
 * P2P replication using libp2p's EpiSub gossip protocol.

## RPC methods.

 - sequencer_append
 - sequencer_read
 - sequencer_info

## Usage.

```sh
./scripts/build.sh

# Make a directory for data.
mkdir tmp

# Run the sequencer primary.
PRIVATE_KEY=0x0801124098bba74fbc32342624d74e8e523644be41d1e745b21af54933735ea6f0d92de17f7858dd065ece3d57a79a48b203664a63c356fb53c2dd3c5ce6a92aca4ebc39 ./cmd/sequencer/sequencer start -dbpath tmp/db -mode primary

# Run a sequencer replica.
PRIVATE_KEY="" ./cmd/sequencer/sequencer start -dbpath tmp/db2 -mode replica -peers "/ip4/192.168.1.189/tcp/24445/p2p/12D3KooWJPxP7QYvfkDoHRXFirAixtvmy3dMjy1eszPza7oFqdgt" -rpcport 25445 -p2pport 25446
```

## Development.

```sh
(base) ➜  cmd git:(master) ✗ go run sequencer/main.go init
Initializing a sequencer primary...

Core operator private key: 0xf13edce794610d4485a1837014d4168f65135faf81219bbb837cc410391d435d
Core operator public key: 0x04c436bb61a162f5e6c1f7b83576251d7629c7b52ca8779a2ce0400dcfb08a0a0b95733e857f287ee018fd4268597859201a2c4a87f90a533c70c793512d44867e
P2P multiaddr: /ip4/192.168.1.189/tcp/24445/p2p/12D3KooWLMmULYCrke9PiATTDTmE4pMDCtxiffWTTM3mhTXgfw2K
P2P private key: 0x08011240e6d9a1faa2fbf1e669169b8813e4439c5d304f82bccdf6a8da30d7e1679edd6e9ca03937ad7b1c86347c24db827cfd0da2743e4946d7437ed6e1571560cad484
```

```sh
PRIVATE_KEY="" go run cmd/sequencer/main.go start -dbpath tmp/db2 -mode replica -peers "/ip4/192.168.1.189/tcp/24445/p2p/12D3KooWJPxP7QYvfkDoHRXFirAixtvmy3dMjy1eszPza7oFqdgt" -rpcport 25445 -p2pport 25446
```

## Benchmarking.

```sh
# macOS users need to do this, as the benchmark uses over 1000 open files.
ulimit -S -n 10000

# Run an external sequencer node.
PRIVATE_KEY="0x08011240e6d9a1faa2fbf1e669169b8813e4439c5d304f82bccdf6a8da30d7e1679edd6e9ca03937ad7b1c86347c24db827cfd0da2743e4946d7437ed6e1571560cad484" OPERATOR_PRIVATE_KEY="3fd7f88cb790c6a8b54d4e1aaebba6775f427bb8fa2276e933b7c3440f164caa" go run cmd/sequencer/main.go start -dbpath "" -mode primary -peers "" -rpcport 49000 -p2pport 49001

# Now run the benchmarks.
cd benchmarking
GOLOG_LOG_LEVEL=info go run run.go
```

## Philosophy.

This is a fully-fledged tx sequencer in around ~2000 LOC.

```
(base) ➜  mvp git:(master) ✗ find . -name '*.go' | xargs wc -l
      95 ./cmd/sequencer/main.go
     254 ./sequencer/p2p.go
     137 ./sequencer/messages/methods.go
     422 ./sequencer/messages/defs.pb.go
     128 ./sequencer/core_test.go
     317 ./sequencer/core.go
      48 ./sequencer/utils/test_utils.go
      49 ./sequencer/utils.go
      43 ./sequencer/rpc.go
     112 ./sequencer/node.go
      84 ./benchmarking/run.go
    1689 total
```


