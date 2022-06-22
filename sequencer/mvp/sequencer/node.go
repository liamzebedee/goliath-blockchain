package sequencer

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/liamzebedee/goliath-blockchain/sequencer/mvp/sequencer/messages"
	"github.com/libp2p/go-libp2p-core/crypto"
)

type SequencerMode uint
const (
	PrimaryMode SequencerMode = iota
	ReplicaMode
)

type SequencerNode struct {
	Seq *SequencerCore
	P2P *P2PNode
	RPC *RPCNode
	Mode SequencerMode
}

func NewSequencerNode(
	dbPath string, 
	rpcPort string, 
	p2pPort string, 
	mode SequencerMode, 
	p2pPrivateKeyRaw string, 
	bootstrapPeersStr string, 
	operatorPrivateKey string,
) (*SequencerNode) {
	// TODO: use sync=FULL for database durability during power loss.
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(fmt.Errorf("couldn't open database %s: %s", dbPath, err))
	}

	err = db.Ping()
	if err != nil {
		panic(fmt.Errorf("couldn't connect to database: %s", err))
	}

	seq := NewSequencerCore(db, operatorPrivateKey)

	// RPC.
	rpcAddr := fmt.Sprintf("0.0.0.0:%s", rpcPort)
	rpc := NewRPCNode(rpcAddr, seq)
	
	// P2P.
	p2pAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", p2pPort)
	bootstrapPeers, err := StringsToAddrs(strings.Split(bootstrapPeersStr, ","))
	if err != nil {
		panic(fmt.Errorf("couldn't parse bootstrap peers: %s", err))
	}

	var p2pPrivateKey crypto.PrivKey
	if p2pPrivateKeyRaw != "" {
		p2pPrivateKey = P2PParsePrivateKey(p2pPrivateKeyRaw)
	} else {
		p2pPrivateKey = P2PGeneratePrivateKey()
	}
	p2p, err := NewP2PNode(p2pAddr, p2pPrivateKey, bootstrapPeers)
	if err != nil {
		panic(fmt.Errorf("couldn't create network node: %s", err))
	}

	node := SequencerNode{
		Seq: seq,
		P2P: p2p,
		RPC: rpc,
		Mode: mode,
	}
	
	return &node
}

func (n *SequencerNode) Start() {
	// Hook them up.
	if n.Mode == PrimaryMode {
		n.Seq.OnNewBlock(func (block *messages.Block) {
			go n.P2P.GossipNewBlock(block)
		})
	}

	if n.Mode == ReplicaMode {
		n.P2P.ListenForNewBlocks(func (block *messages.Block) {
			n.Seq.ProcessBlock(block)
		})
		
		if false {
			go func(){
				fmt.Println("P2P: bootstrapping P2P connections...")

				// Wait until they're connected for the test.
				waitConnectedP2P := make(chan bool)
				numPeersToWaitForConnected := 3
				go func() {
					i := 0
					host := n.P2P.Host

					for true {
						peers := host.Network().Peers()
						fmt.Printf("waiting for connections (%2d): num_peers=%d\n", i, len(peers))
						i++
						
						if len(peers) >= numPeersToWaitForConnected  {
							waitConnectedP2P <- true
							break
						}

						time.Sleep(1000 * time.Millisecond)
					}
				}()

				<-waitConnectedP2P
				fmt.Println("P2P: sufficiently connected!")
			}()
		}

		// Now we just need a way for the replicas to sync up to the latest block.
		// 1. Get the latest hash from the sequencer.
		// 2. Gossip request all of the hashes from peers.
		go func(){
			// Fetch 1000 blocks at a time.
			// What do we do? 
			// Message "IWANT" to the P2P network.
			// Select peers.
			// Literally you want a DHT-like routing mechanism for this.
		}()
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		go func() {
			defer wg.Done()
			n.P2P.Start()
		}()
		go func() {
			defer wg.Done()
			n.RPC.Start()
		}()
	}()
	wg.Wait()
}

// Starts the routine to fetch missing block history.
// The pubsub network is used for disseminating new blocks only.
// Replicas sync blocks they've missed by requesting history from peers.
func (n *SequencerNode) FetchHistory() {

}

func (n *SequencerNode) Close() {
	if err := n.P2P.Close(); err != nil {
		panic(err)
	}
}