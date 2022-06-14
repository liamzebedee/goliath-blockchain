This is a centralized sequencer that can scale to a large number of transactions per second.

## RPC methods.

goliath_getSequence
goliath_sequence


## Usage.

```sh
# Generate a private key.
node -e "console.log(require('ethers').Wallet.createRandom().privateKey)"
# 0xd96a6cca804b24f540dc41ac3f50e2acd7510c33662c3040bafc07bc95b035ed

# Run the sequnecer.
PRIVATE_KEY=d96a6cca804b24f540dc41ac3f50e2acd7510c33662c3040bafc07bc95b035ed go run cmd/sequencer/main.go
```