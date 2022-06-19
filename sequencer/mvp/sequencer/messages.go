package sequencer

import (
	"crypto/ecdsa"
	"encoding/json"

	// "strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)


type Signer interface {
	Sign(digestHash []byte) (sig []byte, err error)
	GetPubkey() (*ecdsa.PublicKey)
}

const SEQUENCE_MESSAGE_TYPE = "goliath/0.0.0/signed-tx"

type SequenceMessage struct {
	Type string `json:"type"`
	From string `json:"from"`
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

func (msg SequenceMessage) SetFrom(pubkey *ecdsa.PublicKey) (SequenceMessage) {
	msg.From = hexutil.Encode(crypto.CompressPubkey(pubkey))
	return msg
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

type Block struct {
	sequenceMsg []byte
	Sig string
}

func (block Block) SigHash() ([]byte) {
	unsigned := block

	encoded, err := json.Marshal(unsigned)
	if err != nil {
		panic(err)
	}
	hash := crypto.Keccak256Hash([]byte(encoded))
	
	return hash.Bytes()
}

// Returns a new SequenceMessage with a signature.
func (block Block) Signed(signer Signer) (Block) {
	signed := block
	
	signature, err := signer.Sign(block.SigHash())
	if err != nil {
		panic(err)
	}
	
	signed.Sig = hexutil.Encode(signature)

	return signed
}

func (b Block) String() string {
	hash := crypto.Keccak256Hash(b.sequenceMsg)
	return hexutil.Encode(hash.Bytes())
}
