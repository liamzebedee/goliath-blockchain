package sequencer

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ethereum/go-ethereum/rpc"
)

type RPCNode struct {
	addr string
	httpServer http.Server
}


func NewRPCNode(addr string, seq *SequencerCore) (*RPCNode) {
	// JSON-RPC server.
	rpc := rpc.NewServer()
	rpc.RegisterName("sequencer", seq)

	// HTTP frontend.
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", rpc.ServeHTTP)

	httpServer := &http.Server{
		Addr:           addr,
		Handler:        serveMux,
		// ReadTimeout:    10 * time.Second,
		// WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	return &RPCNode{
		addr: addr,
		httpServer: *httpServer,
	}
}

func (n *RPCNode) Start() {
	// Start RPC server.
	fmt.Println("RPC listening on http://" + n.addr)
	log.Fatal(n.httpServer.ListenAndServe())
}