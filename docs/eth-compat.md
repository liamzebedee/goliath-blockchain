eth compatibility
=================

Goliath is compatible with Ethereum through using the EVM and implementing the Ethereum JSON-RPC node interface.

What is missing?

 * **blocks**. There are no blocks in goliath, only transactions. Like Optimism.


## Tx Receipts.

Contract deployment involves retaining the deployed contract address in state extraneous to the EVM `eval` application, and returning it in the tx receipt. This is a bit annoying.

There are two approaches:

 1) store it in the storage layer.
 2) store it in the executer/scheduler.

The scheduler communicates with the executer (sputnik), which communicates with storage. This is the rough interactions:

```
user send a tx to the RPC node via eth_sendTransaction
RPC node posts it to the sequencer
the scheduler listens to the sequencer and picks up the tx
scheduler invokes the executer via RPC, which runs the sputnik vm
executer flushes new state to storage layer and returns the receipt to the scheduler
RPC node awaits the sequencer, and then awaits the scheduler for tx receipt
    tx receipts are stored in the storage layer in a separate table for receipts.
    they will probably have a similar garbage collection policy to state slots. GC only occurs for non-archive nodes.
it then sends tx receipt back to the user
```

## Logs.

Logs can be returned in receipts rather easily. The realtime nature of streaming logs might be slightly harder.

```
at the RPC node

filters = []

receipts <- stream scheduler.receipts

now get a list of filters to push this 
receipts.map =>
    const { fromBlock, toBlock, address, topics } = filter
    if receipt.address == address:
        yes
    if toBlock = 'latest'
        yes
    # fromBlock is not implemented
    match_topic = receipt.filter =>
        receipt.logs.filter => 
            if log[0] == topic
                yes

push out the txs down the stream


or we could do it even simpler

while true:
    select * from logs
    where address = address
    and log0 == topic
    and time > latest
```


## Account abstraction.

For the POC, there are no accounts yet. 

We have the opportunity for account abstraction in our model, since we don't use the same execution context as the Ethereum chain.

## Gas.

There's no gas in our EVM (yet). This is a simpler model for the POC. NOTE: SputnikVM doesn't deduct gas from balances. It's trivial to implement, though worth noting.

