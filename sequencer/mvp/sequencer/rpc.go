package sequencer

import (
	"fmt"
	"log"
	"net/http"
	"runtime"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/golang/protobuf/proto"
)

type RPCNode struct {
	addr string
	httpServer http.Server
}

type SequencerService struct {
	seq *SequencerCore
}

func (s *SequencerService) Append(msgData string) (int64, error) {
	return s.seq.Sequence(msgData)
}

func (s *SequencerService) Get(from, to uint64) ([]byte, error) {
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			fmt.Println("RPC method " + " crashed: " + fmt.Sprintf("%v\n%s", err, buf))
			// errRes = errors.New("method handler crashed")
		}
	}()
	fmt.Printf("rpc: get(%d, %d)\n", from, to)

	reply, err := s.seq.Get(from, to)
	if err != nil {
		return nil, err
	}

	buf, err := proto.Marshal(reply)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return buf, nil
}

func (s *SequencerService) Info() ([]byte, error) {
	fmt.Printf("rpc: info()")
	reply, err := s.seq.Info()
	if err != nil {
		return nil, err
	}

	buf, err := proto.Marshal(reply)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func NewRPCNode(addr string, seq *SequencerCore) (*RPCNode) {
	// JSON-RPC server.
	rpc := rpc.NewServer()
	rpc.RegisterName("sequencer", &SequencerService{seq})

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