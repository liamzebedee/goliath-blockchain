package utils

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)


type Signer interface {
	Sign(digestHash []byte) (sig []byte, err error)
	GetPubkey() (*ecdsa.PublicKey)
}


type EthereumECDSASigner struct {
	privateKey *ecdsa.PrivateKey
}

func NewEthereumECDSASignerFromKey(privateKey *ecdsa.PrivateKey) (*EthereumECDSASigner) {
	return &EthereumECDSASigner{
		privateKey: privateKey,
	}
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
	// TODO probably not the best way to get the pubkey.
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