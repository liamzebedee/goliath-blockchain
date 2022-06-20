package messages

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/golang/protobuf/proto"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/utils"
)

func (msg *SequenceTx) ToHex() (string) {
	enc, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}

	return hexutil.Encode(enc)
}

func (msg *SequenceTx) SigHash() ([]byte) {
	unsigned := proto.Clone(msg).(*SequenceTx)
	unsigned.Sig = []byte{}

	// Encode the message, hash it.
	msg_encoded, err := proto.Marshal(unsigned)
	if err != nil {
		panic(err)
	}

	hash := crypto.Keccak256Hash(msg_encoded)
	return hash.Bytes()
}

func (msg *SequenceTx) SetFrom(pubkey *ecdsa.PublicKey) {
	msg.From = crypto.CompressPubkey(pubkey)
}

// Returns a new SequenceMessage with a signature.
func (msg *SequenceTx) Signed(signer utils.Signer) (*SequenceTx) {
	signature, err := signer.Sign(msg.SigHash())
	if err != nil {
		panic(err)
	}
	
	signed := proto.Clone(msg).(*SequenceTx)
	signed.Sig = signature

	return signed
}


func generateNonce() []byte {
	nonce := make([]byte, 32)
	_, err := rand.Read(nonce)
	if err != nil {
		panic(fmt.Errorf("error generating random data for nonce: %s", err))
	}
	return nonce
}


func ConstructSequenceMessage(txData string, expiresIn time.Duration) (*SequenceTx) {
	msg := SequenceTx{}
	msg.Data = hexutil.MustDecode(txData)
	msg.Sig = []byte{}
	msg.Nonce = generateNonce()
	msg.Expires = make([]*ExpiryCondition, 1)
	msg.Expires[0] = &ExpiryCondition{
		Condition: &ExpiryCondition_Unix{
			Unix: &UNIXExpiryCondition{
				Time: uint64(time.Now().Add(expiresIn).UnixMilli()),
			},
		},
	}
	return &msg
}





// Now for the block.

func ConstructBlock(prevBlockHash []byte, sequenceMessage *SequenceTx) (*Block) {
	block := &Block{
		PrevBlockHash: prevBlockHash,
		Body: sequenceMessage,
		Sig: []byte{},
	}

	return block
}

func (block *Block) SigHash() ([]byte) {
	unsigned := proto.Clone(block).(*Block)
	unsigned.Sig = []byte{}

	buf, err := proto.Marshal(unsigned)
	if err != nil {
		panic(err)
	}

	hash := crypto.Keccak256Hash(buf)
	return hash.Bytes()
}

func (block *Block) Signed(signer utils.Signer) (*Block) {
	signature, err := signer.Sign(block.SigHash())
	if err != nil {
		panic(err)
	}
	
	signed := proto.Clone(block).(*Block)
	signed.Sig = signature
	return signed
}

func (block *Block) PrettyString() string {
	var bodyStr string = ""
	if body := block.GetBody(); body != nil {
		bodyStr = hexutil.Encode(body.SigHash())
	}

	return fmt.Sprintf(
		"block [hash=%s prev=%s seq_tx=%s]", 
		hexutil.Encode(block.SigHash()),
		hexutil.Encode(block.PrevBlockHash),
		bodyStr,	
	)
}

func (block *Block) PrettyHash() string {
	return hexutil.Encode(block.SigHash())
}