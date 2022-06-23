Scheduler-executer
==================

## Usage.

EVM execution using SputnikVM is roughly 8ms for a tx.

```sh
go run cmd/executer/main.go
```

## How does it work?

This fetches every block from the Tendermint chain, queries the batch of sequenced transactions for that block, and then executes them.

Pseudocode:

```py
# Scheduler.
# 

db = sqlite.load(db_path)
latest_sequence = db.exec(SELECT * FROM executed ORDER BY sequence DESC LIMIT 1)[0].sequence
queue = []

# Fetch past sequences, and listen for new ones.
items = sequencer.sequenced(latest_sequence, latest, on_new=lambda item: queue.push(item))
for item in items:
    queue.push(item)

# Worker coroutine.
while item = queue.pop():
    execute(item)

def execute(item):
    # unpack tx data
    # pass to executer
    # execute
```


```py
# Executer.
# 

def execute(tx, sequence, output_leaves):
    # Verify the ordering of this transaction.
    # If we enforce an ordering based on last processed sequence number, the VM executer is constrained to executing each transaction serially. 
    # If we enforce an ordering based on data dependencies of state, we can execute non-overlapping transactions in parallel.
    
    # select count(*) as num from storage
    # where sequence > {}
    # and key in (leaves)

    # if num > 0:
    #   throw SequenceViolation("we cannot write to leaves in the past")
```