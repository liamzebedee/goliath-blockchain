package sequencer

import (
	"encoding/json"
	// "strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)


type Signer interface {
	Sign(digestHash []byte) (sig []byte, err error)
}

const SEQUENCE_MESSAGE_TYPE = "goliath/0.0.0/signed-tx"

type SequenceMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Sig string `json:"sig"`
	Nonce string `json:"nonce"`
	Expires [][]interface{} `json:"expires"`
}

func (msg SequenceMessage) ToJSON() (string) {
	msg_encoded, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return string(msg_encoded)
}

func (msg SequenceMessage) SigHash() ([]byte) {
	msg_unsigned := msg
	msg_unsigned.Sig = ""

	// Encode the message, hash it.
	msg_encoded, err := json.Marshal(msg_unsigned)
	if err != nil {
		panic(err)
	}

	data := []byte(msg_encoded)
	hash := crypto.Keccak256Hash(data)
	
	return hash.Bytes()
}

// Returns a new SequenceMessage with a signature.
func (msg SequenceMessage) Signed(signer Signer) (SequenceMessage) {
	msg_signed := msg
	
	signature, err := signer.Sign(msg.SigHash())
	if err != nil {
		panic(err)
	}
	
	msg_signed.Sig = hexutil.Encode(signature)

	return msg_signed
}