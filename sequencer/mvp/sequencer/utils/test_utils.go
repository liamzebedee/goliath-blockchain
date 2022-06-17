package utils

import (
	"crypto/ecdsa"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer"
)

func ConstructSequenceMessage(txData string, expiresIn time.Duration) (sequencer.SequenceMessage) {
	msg := sequencer.SequenceMessage{}
	msg.Type = "goliath/0.0.0/signed-tx"
	msg.Data = txData
	msg.Sig = ""
	msg.Nonce = "" // TODO
	expiry_conditions := make([][]interface{}, 1)
	expiry_conditions[0] = []interface{}{"unix", time.Now().Add(expiresIn).UnixMilli(), }
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

func (s *EthereumECDSASigner) GetPubkey() (*ecdsa.PublicKey) {
	// TODO proabbly not the best way to get the pubkey.
	badData, _ := hexutil.Decode("0xaaaa")
	digest := crypto.Keccak256Hash(badData).Bytes()
	sig, err := s.Sign(digest)
	if err != nil {
		panic(err)
	}
	pub, err := crypto.SigToPub(digest, sig)
	if err != nil {
		panic(err)
	}
	return pub
}