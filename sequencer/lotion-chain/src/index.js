let lotion = require('lotion')
const { join } = require('path')

const NETWORK_DIR = join(__dirname, '../networks/')
const NETWORK_ID = 'example-chain'

// While the default behavior of tendermint is still to create blocks approximately once per second, it is possible to disable empty blocks or set a block creation interval. In the former case, blocks will be created when there are new transactions or when the AppHash changes.
// consensus.create_empty_blocks=false

let app = lotion({
    initialState: {
        clock: 0,
        txs: [],
        txCount: 0
    },
    rpcPort: 54046
    // keyPath: join(NETWORK_DIR, '/config/priv_validator_key.json'),
    // genesisPath: join(NETWORK_DIR, '/config/genesis.json'),
})

app.home = join(NETWORK_DIR, '/', NETWORK_ID)

function transactionHandler(state, transaction) {
    let time = state.clock
    let tx = { 
        time, 
        hash: transaction.hash,
    }
    state.txs = [...state.txs, tx]
    state.txCount += 1
    state.clock += 1
}

app.use(transactionHandler)

app.start().then(appInfo => {
    console.log(`Started sequencer chain`)
    console.log(`Home: ${appInfo.home}`)
    console.log(`GCI: ${appInfo.GCI}`)
})
