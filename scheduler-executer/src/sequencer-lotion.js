const LotionConnect = require('lotion-connect')

const range = (start, n) => Array(n).fill(start).map((x, y) => x + y)

async function listenForTransactions(onTxs) {
    const lotion = await LotionConnect(process.env.GCI)
    const lightClient = lotion.lightClient
    
    // State.
    // TODO: for this proto, the executer does not look at past block history for transactions.
    let lastTxCount = await lotion.state.txCount
    // TODO: handle when we timeout from the sequencer.

    lightClient.on('update', async (header, commit, validators) => {
        const currentCount = await lotion.state.txCount
        console.log(`block time=${header.time} height=${header.height} hash=${commit.block_id.hash} txCount=${currentCount}`)

        if (lastTxCount < currentCount) {
            console.log(`Fetching txs...`)

            const num = currentCount - lastTxCount
            const txs = await Promise.all(
                range(lastTxCount, num).map(i => lotion.state.txs[i])
            )

            lastTxCount = currentCount

            onTxs(txs)
        }

        // Get the latest batch of transactions and execute them.
        // if (parseInt(header.num_txs) > 0) {
        //     // Process block.

        //     try {
        //         // let res = await lightClient.rpc.block({
        //         //     height: header.height
        //         // })

        //         // console.log(res.block.data.txs)
        //     } catch(ex) {
        //         console.error(ex)
        //         throw ex
        //     }
        // }
    })
}

async function executeTransaction(tx) {
    console.log(`Executing ${tx}`)
}

async function run() {
    let txQueue = []
    
    const onTxs = (txs) => txQueue.push(...txs)
    await listenForTransactions(onTxs)

    while (true) {
        // Process each tx in queue.
        if(txQueue.length === 0) { await new Promise((res,rej) => setTimeout(res, 20)); continue }
        const tx = txQueue.pop()
        await executeTransaction(tx)
    }
}

/*

{
  height: '252561',
  results: {
    DeliverTx: [ null ],
    EndBlock: { validator_updates: [Array] },
    BeginBlock: {}
  }
}

*/

run().catch(ex => {
    throw ex; 
})