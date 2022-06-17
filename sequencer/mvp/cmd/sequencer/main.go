package main

import (
	"crypto/ecdsa"
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

const DB_PATH = "db.sqlite"

func parsePrivateKey() *ecdsa.PrivateKey {
	privateKeyEnv := os.Getenv("PRIVATE_KEY")
	
	privateKey, err := crypto.HexToECDSA(privateKeyEnv)

	if err != nil {
		log.Fatal(err)
	}

	return privateKey
}

func getDatabasePathWithOptions(filepath string) string {
	return fmt.Sprintf("file:%s?cache=shared", filepath)
}


func main() {
	// Arguments parsing.
	rpcport := flag.String("rpcport", "24444", "RPC port to listen on")
	p2pport := flag.String("p2pport", "24445", "P2P port to listen on")
	mode_flag := flag.String("mode", "primary", "mode to operate in")
	peers := flag.String("peers", "", "peers to join the pubsub network on")
	dbPath := flag.String("dbpath", DB_PATH, "path to the database")
	// privateKey := parsePrivateKey()
	privateKey := os.Getenv("PRIVATE_KEY")

    flag.Parse()

	var mode sequencer.SequencerMode
	switch *mode_flag {
	case "primary":
		mode = sequencer.PrimaryMode
	case "replica":
		mode = sequencer.ReplicaMode
	default:
		panic(fmt.Errorf("unknown sequencer mode: %s", *mode_flag))
	}

	if privateKey == "" && mode == sequencer.PrimaryMode {
		panic("PRIVATE_KEY environment variable is empty!")
	}

	fmt.Println("Goliath Sequencer")
	fmt.Println("Mode:", *mode_flag)

	// Core sequencer engine.
	// db, err := sql.Open("sqlite3", DB_PATH)
	// if err != nil {
	// 	panic(fmt.Errorf("couldn't open database %s: %s", DB_PATH, err))
	// }
	node := sequencer.NewSequencerNode(
		getDatabasePathWithOptions(*dbPath),
		*rpcport,
		*p2pport,
		mode,
		privateKey,
		*peers,
	)

	// Handle shutdowns.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go (func() {
		<-ch
		fmt.Println("\nReceived kill signal, shutting down...")

		// shut the node down
		node.Close()

		os.Exit(0)
	})()

	// Start the node.
	node.Start()
}