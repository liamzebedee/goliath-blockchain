package sequencer

import (
	"testing"
	"database/sql"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer"
	"github.com/stretchr/testify/assert"
	"time"

	"encoding/json"
	// "strconv"
	"fmt"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func getMockSequencer() (*sequencer.SequencerService, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	seq := sequencer.NewSequencerService(db)
	return seq, nil
}

type Signer interface {
	Sign(digestHash []byte) (sig []byte, err error)
}


type SequenceMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Sig string `json:"sig"`
	Nonce string `json:"nonce"`
	Expires []interface{} `json:"expires"`
}

func (msg SequenceMessage) ToJSON() (string) {
	// msg := make(map[string]interface{})
	// msg["type"] = "goliath/0.0.0/signed-tx"
	// msg["data"] = txData
	// msg["sig"] = ""
	// msg["nonce"] = "1"
	msg_encoded, err := json.Marshal(msg)
	
	if err != nil {
		panic(err)
	}
	return string(msg_encoded)
}

// Returns a new SequenceMessage with a signature.
func (msg SequenceMessage) Signed(signer Signer) (SequenceMessage) {
	msg_signed := msg

	// Encode the message, hash it, and sign the hash.
	msg_encoded, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}

	data := []byte(msg_encoded)
	hash := crypto.Keccak256Hash(data)
	fmt.Println(hash.Hex())
	signature, err := signer.Sign(hash.Bytes())
	if err != nil {
		panic(err)
	}
	msg_signed.Sig = hexutil.Encode(signature)

	return msg_signed
}

func constructSequenceMessage(txData string, expiresIn time.Duration) (SequenceMessage) {
	msg := SequenceMessage{}
	msg.Type = "goliath/0.0.0/signed-tx"
	msg.Data = txData
	msg.Sig = ""
	msg.Nonce = "" // TODO
	expiry_conditions := make([]interface{}, 1)
	expiry_conditions[0] = []interface{}{"unix", time.Now().Add(expiresIn).UnixMilli(),}
	msg.Expires = expiry_conditions
	return msg
}

type EthereumECDSASigner struct {
	privateKey *ecdsa.PrivateKey
}

func NewEthereumECDSASigner(privateKeyHex string) (*EthereumECDSASigner) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		panic(err)
	}

	return &EthereumECDSASigner{
		privateKey: privateKey,
	}
}

func (s *EthereumECDSASigner) Sign(digestHash []byte) (sig []byte, err error) {
	return crypto.Sign(digestHash, s.privateKey)
}

func TestSequence(t *testing.T) {
    seq, err := getMockSequencer()
	if err != nil {
		t.Error(err)
	}

	// 
	// Failure conditions for sequencing:
	// 
	
	// 1. Message is malformed.
	msg := SequenceMessage{}
	seqno, err := seq.Sequence(msg.ToJSON())
	assert.EqualError(t, err, "message is malformed")

	// 2. Invalid signature.
	signer := NewEthereumECDSASigner("3977045d27df7e401ecf1596fd3ae86b59f666944f81ba8dbf547c2269902f6b")
	txData := "c4a6abb1cc341e7b796bdc0fb11c50a12d4e998cc4e8e3cb44badf185a8e00f7"
	
	msg = constructSequenceMessage(txData, 5 * time.Second)
	seqno, err = seq.Sequence(msg.ToJSON())
	assert.EqualError(t, err, "invalid signature")

	// 3. Message is expired.
	msg = constructSequenceMessage(txData, 0)
	msg = msg.Signed(signer)
	seqno, err = seq.Sequence(msg.ToJSON())
	assert.EqualError(t, err, "message expired")

	// Happy path!
	msg = constructSequenceMessage(txData, 1 * time.Second)
	msg = msg.Signed(signer)
	seqno, err = seq.Sequence(msg.ToJSON())
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, seqno, 1, "First tx sequenced should have sequence number of 0")
}

// func TestGet(t *testing.T) {
//     seq, err := getMockSequencer()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	sequences, err := seq.Get(0, 3)
// }

func TestInfo(t *testing.T) {
    seq, err := getMockSequencer()
	if err != nil {
		t.Error(err)
	}

	info, err := seq.Info()
	if err != nil {
		t.Error(err)
	}
	
	assert.Equal(t, info.Total, 0, "Total should be 0")
	assert.Equal(t, info.LastSequenceTime, 0, "LastSequenceTime should be 0")


	start := time.Now().UnixMilli()
	seq.Sequence("")
	end := time.Now().UnixMilli()
	
	assert.Equal(t, info.Total, 1, "Total should be 1")
	assert.True(
		t,
		start <= info.LastSequenceTime && info.LastSequenceTime < end,
		"LastSequenceTime should be a recent timestamp.\nstart=%d\nLastSequenceTime=%d\nend=%d\n",
		start, info.LastSequenceTime, end,
	)
}