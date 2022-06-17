package sequencer

import (
	"database/sql"
	"fmt"
)

type SequencerMode int
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

func NewSequencerNode(dbPath string, rpcPort string, p2pPort string, mode SequencerMode) (*SequencerNode) {
	// Core sequencer engine.

	// TODO: use sync=FULL for database durability during power loss.
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		panic(fmt.Errorf("couldn't open database %s: %s", dbPath, err))
	}

	err = db.Ping()
	if err != nil {
		panic(fmt.Errorf("couldn't connect to database: %s", err))
	}

	seq := NewSequencerCore(db)

	// RPC.
	rpcAddr := fmt.Sprintf("0.0.0.0:%s", rpcPort)
	rpc := NewRPCNode(rpcAddr, seq)
	
	// P2P.
	p2pAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", p2pPort)
	p2p, err := NewP2PNode(p2pAddr)
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
		go n.P2P.GossipNewBlocks(n.Seq.BlockChannel)
	}

	if n.Mode == PrimaryMode {
		receiveBlockChan := make(chan Block)
		go n.P2P.ListenForNewBlocks(receiveBlockChan)
		go (func(){
			// block := <-receiveBlockChan
			// current block = 5
			// new block = ?
			// if currBlock.num < newBlock.num { core.ProcessBlock }
		})()
	}

	go n.P2P.Start()
	go n.RPC.Start()
}

func (n *SequencerNode) Close() {
	if err := n.P2P.Close(); err != nil {
		panic(err)
	}
}