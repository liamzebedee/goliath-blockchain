cosmos notes
============

## todo

 - transaction contains (to, from, data, gas, gasprice, sequence)
 - sequence is defined as an integer? or should it be something else?
 - we could define sequence, and then make the scheduler read every block from the chain, fetch all new transactions in the block.
 - one account is permissioned to post data. it incurs no gas cost for doing so.


## Notes


Events - https://docs.cosmos.network/main/core/events.html

```
{
  "jsonrpc": "2.0",
  "method": "subscribe",
  "id": "0",
  "params": {
    "query": "tm.event='Tx' AND eventType.eventAttribute='attributeValue'"
  }
}
```