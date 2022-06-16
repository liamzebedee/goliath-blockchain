package main

import (
	"crypto/ecdsa"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"os"

	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer"

	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/mattn/go-sqlite3"
)

const DEFAULT_PORT = "24444"
const DB_PATH = "db.sqlite"

func parsePrivateKey() *ecdsa.PrivateKey {
	privateKeyEnv := os.Getenv("PRIVATE_KEY")
	
	privateKey, err := crypto.HexToECDSA(privateKeyEnv)

	if err != nil {
		log.Fatal(err)
	}

	return privateKey
}

type SequencerMode int
const (
	PrimaryMode SequencerMode = iota
	ReplicaMode
)

var mode SequencerMode = PrimaryMode

func main() {
	
	// Arguments parsing.
	port := flag.String("port", DEFAULT_PORT, "port to listen on")
	mode_flag := flag.String("mode", "primary", "mode to operate in")
	
	// privateKey := parsePrivateKey()
    flag.Parse()

	switch *mode_flag {
	case "primary":
		mode = PrimaryMode
	case "replica":
		mode = ReplicaMode
	default:
		panic(fmt.Errorf("unknown sequencer mode: %s", *mode_flag))
	}

	fmt.Println("Goliath Sequencer")
	fmt.Println("Mode:", *mode_flag)

	// Core sequencer engine.
	db, err := sql.Open("sqlite3", DB_PATH)
	if err != nil {
		panic(fmt.Errorf("couldn't open database %s: %s", DB_PATH, err))
	}
	seq := sequencer.NewSequencerService(db)

	// RPC.
	addr := "0.0.0.0:" + *port
	rpc := sequencer.NewRPCNode(addr, seq)
	
	// P2P.
	node, err := sequencer.NewP2PNode("/ip4/0.0.0.0/tcp/24344")
	if err != nil {
		panic(fmt.Errorf("couldn't create network node: %s", err))
	}

	// Hook them up.
	if mode == PrimaryMode {
		go node.GossipNewBlocks(seq.BlockChannel)
	}

	if mode == ReplicaMode {
		receiveBlockChan := make(chan sequencer.Block)
		go node.ListenForNewBlocks(receiveBlockChan)
		go (func(){
			// block := <-receiveBlockChan
			// current block = 5
			// new block = ?
			// if currBlock.num < newBlock.num { core.ProcessBlock }
		})()
	}


	// Handle shutdowns.	
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go (func() {
		<-ch
		fmt.Println("Received signal, shutting down...")

		// shut the node down
		if err := node.Close(); err != nil {
			panic(err)
		}

		os.Exit(0)
	})()

	// Start RPC server.
	rpc.Start()
}