package messages

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/utils"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestBlockSigned(t *testing.T) {
	// 0x043e0b751273070a517b4c54393deb672e75a6d9dd731bd0b90f11bb178343dc2084ac3c86e289d0902fe40fbb7bb24efd2a342a95220347ed7cedd0dd19d629f5
	signer := utils.NewEthereumECDSASigner("3fd7f88cb790c6a8b54d4e1aaebba6775f427bb8fa2276e933b7c3440f164caa")
	sequenceMessage := ConstructSequenceMessage("0x0001", 0)
	block := ConstructBlock(sequenceMessage)
	
	signed_block := block.Signed(signer)
	assert.Len(t, block.Sig, 0, "Sig is not set")
	assert.True(t, len(signed_block.Sig) > 0, "Sig is set")

	digestHash := signed_block.SigHash()
	assert.True(t, len(signed_block.Sig) > 0, "Sig is still set")
	
	pubkey, err := crypto.Ecrecover(digestHash, signed_block.Sig)
	if err != nil {
		panic(err)
	}

	// expectedPubkey := crypto.FromECDSAPub(signer.GetPubkey())
	// fmt.Println("pubkey", hexutil.Encode(expectedPubkey))
	expectedPubkey := hexutil.MustDecode("0x043e0b751273070a517b4c54393deb672e75a6d9dd731bd0b90f11bb178343dc2084ac3c86e289d0902fe40fbb7bb24efd2a342a95220347ed7cedd0dd19d629f5")
	assert.True(t, bytes.Equal(pubkey, expectedPubkey), "invalid signer for block\n     got: %s\nexpected: %s\n", hexutil.Encode(pubkey), hexutil.Encode(expectedPubkey))

	// Verify signature.
	signatureValid := crypto.VerifySignature(pubkey, digestHash, signed_block.Sig[:len(signed_block.Sig)-1])
	assert.True(t, signatureValid, "signature invalid")


	// Now test the Mashal/Unmarshal.
	signed_block_enc, err := proto.Marshal(signed_block)
	assert.Nil(t, err)

	signed_block_dec := &Block{}
	err = proto.Unmarshal(signed_block_enc, signed_block_dec)
	
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(signed_block.SigHash(), signed_block_dec.SigHash()), "sighash(block) != sighash(decode(encode(block))")
	
	pubkey2, err := crypto.Ecrecover(signed_block_dec.SigHash(), signed_block_dec.Sig)
	if err != nil {
		panic(err)
	}
	assert.True(t, bytes.Equal(pubkey, pubkey2), "pubkeys dont match")
}
