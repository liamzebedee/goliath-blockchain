package commands

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"os"

	"github.com/google/subcommands"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer"

	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/mattn/go-sqlite3"
)

type StartCmd struct {
  rpcport *string
  p2pport *string
  mode_flag *string
  peers *string
  dbPath *string
}

func (*StartCmd) Name() string     { return "start" }
func (*StartCmd) Synopsis() string { return "starts the sequencer node." }
func (*StartCmd) Usage() string {
  return `start:
  Starts a sequencer node.
`
}

func (cmd *StartCmd) SetFlags(f *flag.FlagSet) {
	// Arguments parsing.
	cmd.rpcport = f.String("rpcport", "24444", "RPC port to listen on")
	cmd.p2pport = f.String("p2pport", "24445", "P2P port to listen on")
	cmd.mode_flag = f.String("mode", "primary", "mode to operate in")
	cmd.peers = f.String("peers", "", "peers to join the pubsub network on")
	cmd.dbPath = f.String("dbpath", DB_PATH, "path to the database")
}

func (cmd *StartCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// privateKey := parsePrivateKey()
	privateKey := os.Getenv("PRIVATE_KEY")

	var mode sequencer.SequencerMode
	switch *cmd.mode_flag {
	case "primary":
		mode = sequencer.PrimaryMode
	case "replica":
		mode = sequencer.ReplicaMode
	default:
		panic(fmt.Errorf("unknown sequencer mode: %s", *cmd.mode_flag))
	}

	if privateKey == "" && mode == sequencer.PrimaryMode {
		panic("PRIVATE_KEY environment variable is empty!")
	}

	fmt.Println("Goliath Sequencer")
	fmt.Println("Mode:", *cmd.mode_flag)

	// Sequencer node.
	node := sequencer.NewSequencerNode(
		getDatabasePathWithOptions(*cmd.dbPath),
		*cmd.rpcport,
		*cmd.p2pport,
		mode,
		privateKey,
		*cmd.peers,
		"",
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
	
	return 0
}


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

