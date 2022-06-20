package sequencer

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/messages"
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
		receiveBlockChan := make(chan *messages.Block, 1) // TODO event handler here
		go n.P2P.ListenForNewBlocks(receiveBlockChan)
		
		go (func(){
			for {
				block := <-receiveBlockChan
				
				fmt.Println("verifying block:", block.PrettyHash())
				go func(){
					err := n.Seq.ProcessBlock(block)
					if err != nil {
						fmt.Println("error while verifying block", block.PrettyHash(), ":", err)
					} else {
						fmt.Println("verification success for block:", block.PrettyHash())
					}
				}()
			}
		})()

		// Now we just need a way for the replicas to sync up to the latest block.
		// 1. Get the latest hash from the sequencer.
		// 2. Gossip request all of the hashes from peers.
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

func (n *SequencerNode) Close() {
	if err := n.P2P.Close(); err != nil {
		panic(err)
	}
}