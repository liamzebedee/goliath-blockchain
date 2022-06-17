package sequencer

import (
	"bytes"
	"encoding/json"
	"fmt"

	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type SequencerCore struct {
	// privateKey *ecdsa.PrivateKey
	db *sql.DB

	BlockChannel chan Block

	Total int64
	// milliseconds.
	LastSequenceTime int64
}

func NewSequencerCore(db *sql.DB) (*SequencerCore) {
	fmt.Println("migrating database")
	_, err := db.Exec(`
	CREATE TABLE sequence (
		num INTEGER PRIMARY KEY AUTOINCREMENT, 
		msg BLOB,
		txid BLOB
	);
	`)
	fmt.Println("migration complete")

	if err != nil {
		panic(err)
	}

	return &SequencerCore{
		// privateKey *ecdsa.PrivateKey
		// privateKey: privateKey,
		BlockChannel: make(chan Block),
		db: db,
	}
}

func (s *SequencerCore) Close() {
	s.db.Close()
}

// func (s *SequencerService) disseminateBlocks() {
// 	for {
// 		block := <-s.blockChannel
// 		// select {
// 		// case :
// 		// }
// 	}
// }

func (s *SequencerCore) ProcessBlock(block Block) (error) {
	// current block = 5
	// new block = ?
	// if currBlock.num < newBlock.num { core.ProcessBlock }

	// TODO
	var msg SequenceMessage

	err := json.Unmarshal([]byte(block.sequenceMsg), &msg)
	if err != nil {
		return err
	}

	err = s.verifySequenceMessage(msg)
	if err != nil {
		return err
	}
	
	// Then append to the log.
	_, err = s.db.Exec(
		"INSERT INTO sequence values (?, ?, ?)",
		nil,
		block.sequenceMsg,
		nil,
	)
	if err != nil {
		return fmt.Errorf("error writing tx to db: %s", err)
	}

	return nil
}

func (s *SequencerCore) verifySequenceMessage(msg SequenceMessage) (error) {
	if msg.Type != SEQUENCE_MESSAGE_TYPE {
		return fmt.Errorf("unhandled message type: %s", msg.Type)
	}

	if len(msg.Data) == 0 || msg.Sig == "" {
		return fmt.Errorf("message is malformed")
	}

	if len(msg.From) == 0 || msg.From == "0x" {
		return fmt.Errorf("message is malformed")
	}

	// Verify signature.
	digestHash := msg.SigHash()
	signature, err := hexutil.Decode(msg.Sig)

	fmt.Printf("sequence hash=%s\n", hexutil.Encode(digestHash))

	if err != nil {
		// TODO
		fmt.Println("error while parsing msg.Sig", err.Error())
		return fmt.Errorf("invalid signature")
	}

	pubkey, err := crypto.Ecrecover(digestHash, signature)
	if err != nil {
		// TODO
		fmt.Println("error while recovering pubkey:", err.Error())
		return fmt.Errorf("invalid signature")
	}
	
	fromField, err := hexutil.Decode(msg.From)
	if err != nil {
		return fmt.Errorf("message is malformed")
	}

	fromPubkey, err := crypto.DecompressPubkey(fromField)
	if err != nil {
		return fmt.Errorf("message is malformed")
	}

	if !bytes.Equal(pubkey, crypto.FromECDSAPub(fromPubkey)) {
		fmt.Println("message signature is for different pubkey:")
		return fmt.Errorf("invalid signature")
	}

	// remove recovery id (last byte) from signature.
	signatureValid := crypto.VerifySignature(pubkey, digestHash, signature[:len(signature)-1])
	if !signatureValid {
		return fmt.Errorf("invalid signature")
	}

	// Check expiry conditions.
	for _, expiry_cond := range msg.Expires {
		expiry_check_type := expiry_cond[0]

		if expiry_check_type == "unix" {
			expiry_time := expiry_cond[1].(float64)

			if err != nil {
				// TODO
				fmt.Println("error while parsing expiry check time", err.Error())
				return fmt.Errorf("message is malformed")
			}
			
			if int64(expiry_time) < time.Now().UnixMilli() {
				return fmt.Errorf("message expired")
			}
		} else {
			return fmt.Errorf("unknown expiry condition '%s'", expiry_check_type)
		}
	}

	if err != nil {
		return err
	}

	return nil
}

// Assigns a sequence number for the transaction.
func (s *SequencerCore) Sequence(msgData string) (int64, error) {
	var msg SequenceMessage

	err := json.Unmarshal([]byte(msgData), &msg)
	if err != nil {
		return 0, err
	}

	err = s.verifySequenceMessage(msg)
	if err != nil {
		return 0, err
	}
	
	// Then append to the log.
	// Save to DB.
	res, err := s.db.Exec(
		"INSERT INTO sequence values (?, ?, ?)",
		nil,
		msgData,
		nil,
	)
	if err != nil {
		return 0, fmt.Errorf("error writing tx to db: %s", err)
	}

	newBlock := Block{
		sequenceMsg: []byte(msgData),
	}

	s.BlockChannel <- newBlock

	lastId, err := res.LastInsertId()
	if err != nil {
		// TODO
		panic(err)
	}

	return lastId, nil
}

// Returns the transactions between index `from` and `to`.
func (s *SequencerCore) Get(from, to uint64) (int, error) {
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
func (s *SequencerCore) Info() (SequencerInfo, error) {
	// info := make(map[string]interface{})
	// info["total"] = 2
	// info["latest"] = 21

	info := SequencerInfo{
		Total: 0,
		LastSequenceTime: 0,
	}
	
	return info, nil
}