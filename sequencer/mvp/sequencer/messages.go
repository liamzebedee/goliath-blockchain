package sequencer

import (
	"encoding/json"

	// "strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/utils"
)

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
func (block Block) Signed(signer utils.Signer) (Block) {
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
