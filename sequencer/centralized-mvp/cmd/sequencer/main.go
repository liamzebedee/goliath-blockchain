package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"net/http"

	"os"

	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	_ "github.com/mattn/go-sqlite3"
)

const DEFAULT_PORT = "24444"

func parsePrivateKey() *ecdsa.PrivateKey {
	privateKeyEnv := os.Getenv("PRIVATE_KEY")
	
	privateKey, err := crypto.HexToECDSA(privateKeyEnv)

	if err != nil {
		log.Fatal(err)
	}

	return privateKey
}

func main() {
	fmt.Println("Goliath Sequencer")
	
	// Arguments parsing.
	port := flag.String("port", DEFAULT_PORT, "port to listen on")
	// privateKey := parsePrivateKey()
    flag.Parse()

	// JSON-RPC server.
	sequencer := sequencer.NewSequencerService()
	server := rpc.NewServer()
	server.RegisterName("sequencer", sequencer)

	addr := "0.0.0.0:" + *port
	http.HandleFunc("/", server.ServeHTTP)

	// Start RPC server.
	fmt.Println("Listening on http://" + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}