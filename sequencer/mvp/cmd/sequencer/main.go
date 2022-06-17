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


func main() {
	// Arguments parsing.
	port := flag.String("port", DEFAULT_PORT, "port to listen on")
	mode_flag := flag.String("mode", "primary", "mode to operate in")
	// peers := flag.String("peers", "", "peers to join the pubsub network on")
	
	// privateKey := parsePrivateKey()
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

	fmt.Println("Goliath Sequencer")
	fmt.Println("Mode:", *mode_flag)

	// Core sequencer engine.
	// db, err := sql.Open("sqlite3", DB_PATH)
	// if err != nil {
	// 	panic(fmt.Errorf("couldn't open database %s: %s", DB_PATH, err))
	// }

	node := sequencer.NewSequencerNode(
		DB_PATH,
		*port,
		"24344",
		mode,
	)

	// Handle shutdowns.	
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go (func() {
		<-ch
		fmt.Println("Received signal, shutting down...")

		// shut the node down
		node.Close()

		os.Exit(0)
	})()

	// Start the node.
	node.Start()
}