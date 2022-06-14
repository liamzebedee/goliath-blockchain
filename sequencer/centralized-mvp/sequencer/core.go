package sequencer

import (
	"fmt"
	"log"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type SequencerService struct {
	// privateKey *ecdsa.PrivateKey
	db *sql.DB
}

func GetDefaultDatabase() (*sql.DB) {
	// Database.
	db, err := sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal(err)
	}
	// TODO:
	// defer db.Close()
	return db
}

func NewSequencerService(db *sql.DB) (*SequencerService) {
	db.Exec(`
	CREATE TABLE sequence (
		index INTEGER PRIMARY KEY AUTOINCREMENT, 
		msg BLOB,
		txid BLOB
	);
	`)

	return &SequencerService{
		// privateKey *ecdsa.PrivateKey
		// privateKey: privateKey,
		db: db,
	}
}

// Assigns a sequence number for the transaction.
func (s *SequencerService) Sequence(msg string) (int, error) {
	// Decode the sequence message.
	// Verify the signature according to the Ethereum VM installation.
	// Then append to the log.
	fmt.Println(msg)
	// Save to DB.
	return 1, nil
}

// Returns the transactions between index `from` and `to`.
func (s *SequencerService) Get(from, to uint64) (int, error) {

	return 1, nil
}

type SequencerInfo struct {
	Total int64            `json:"total"`
	// milliseconds.
	LastSequenceTime int64 `json:"lastSequenceTime"`
}

// Get the sequencer info.
// - total number of sequenced txs.
// - latest received tx time.
func (s *SequencerService) Info() (SequencerInfo, error) {
	
	// info := make(map[string]interface{})
	// info["total"] = 2
	// info["latest"] = 21

	info := SequencerInfo{
		Total: 0,
		LastSequenceTime: 0,
	}
	return info, nil
}