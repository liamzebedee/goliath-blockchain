package utils

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)


type Signer interface {
	Sign(digestHash []byte) (sig []byte, err error)
	GetPubkey() (*ecdsa.PublicKey)
	String() string
}


type EthereumECDSASigner struct {
	privateKey *ecdsa.PrivateKey
	publicKey *ecdsa.PublicKey
}

func NewEthereumECDSASignerFromKey(privateKey *ecdsa.PrivateKey) (*EthereumECDSASigner) {
	s := &EthereumECDSASigner{
		privateKey: privateKey,
	}
	s.publicKey = s.getPubkey()
	return s
}

func NewEthereumECDSASigner(privateKeyHex string) (*EthereumECDSASigner) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		panic(err)
	}

	s := &EthereumECDSASigner{
		privateKey: privateKey,
	}
	s.publicKey = s.getPubkey()
	return s
}

func (s *EthereumECDSASigner) Sign(digestHash []byte) (sig []byte, err error) {
	return crypto.Sign(digestHash, s.privateKey)
}

func (s *EthereumECDSASigner) GetPubkey() (*ecdsa.PublicKey) {
	return s.publicKey
}

func (s *EthereumECDSASigner) String() (string) {
	return hexutil.Encode(crypto.FromECDSAPub(s.publicKey))
}

func (s *EthereumECDSASigner) getPubkey() (*ecdsa.PublicKey) {
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