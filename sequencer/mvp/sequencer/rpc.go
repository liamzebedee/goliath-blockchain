package sequencer

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"
)

type RPCNode struct {
	addr string
}

func NewRPCNode(addr string, seq *SequencerService) (*RPCNode) {
	// JSON-RPC server.
	server := rpc.NewServer()
	server.RegisterName("sequencer", seq)

	http.HandleFunc("/", server.ServeHTTP)

	return &RPCNode{
		addr: addr,
	}
}

func (n *RPCNode) Start() {
	// Start RPC server.
	fmt.Println("RPC listening on http://" + n.addr)
	log.Fatal(http.ListenAndServe(n.addr, nil))
}