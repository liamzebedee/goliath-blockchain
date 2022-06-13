package main

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"net/http"

	"database/sql"

	"os"

	"github.com/ethereum/go-ethereum/common/hexutil"
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
	privateKey := parsePrivateKey()
    flag.Parse()

	// JSON-RPC server.
	sequencer := NewSequencerService(privateKey)
	server := rpc.NewServer()
	server.RegisterName("sequencer", sequencer)

	addr := "0.0.0.0:" + *port
	http.HandleFunc("/", server.ServeHTTP)
	
	// Database.
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	db.Exec(`
	CREATE TABLE sequence (
		index INTEGER PRIMARY KEY AUTOINCREMENT, 
		msg BLOB,
		txid BLOB
	);
	`)

	// Start RPC server.
	fmt.Println("Listening on http://" + addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

type SequencerService struct {
	privateKey *ecdsa.PrivateKey
}

func NewSequencerService(privateKey *ecdsa.PrivateKey) (SequencerService) {
	return SequencerService{
		privateKey: privateKey,
	}
}

// Assigns a sequence number for the transaction.
func (s *SequencerService) Sequence(tx string) (int, error) {
	// tx is string starting with 0x.
	// We sign a message as so:
	// {"type":"goliath/0.0.0/signed-tx","data":""}
	// Return the signature.
	data := []byte(tx)
	hash := crypto.Keccak256Hash(data)
	fmt.Println(hash.Hex())
	signature, err := crypto.Sign(hash.Bytes(), s.privateKey)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(hexutil.Encode(signature)) // 0x789a80053e4927d0a898db8e065e948f5cf086e32f9ccaa54c1908e22ac430c62621578113ddbb62d509bf6049b8fb544ab06d36f916685a2eb8e57ffadde02301
	return 1, nil
}

// Returns the transactions between index `from` and `to`.
func (s *SequencerService) Get(from, to uint64) (int, error) {

	return 1, nil
}