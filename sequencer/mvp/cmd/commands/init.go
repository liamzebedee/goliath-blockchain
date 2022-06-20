package commands

import (
	"context"
	"flag"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/subcommands"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/utils"
	"github.com/libp2p/go-libp2p"
	libp2pCrypto "github.com/libp2p/go-libp2p-core/crypto"

	_ "github.com/mattn/go-sqlite3"
)

type InitCmd struct {
	p2pport *string
}

func (*InitCmd) Name() string     { return "init" }
func (*InitCmd) Synopsis() string { return "initializes configuration for a sequencer node." }
func (*InitCmd) Usage() string {
  return `init:
  Generates an initial configuration for setting up a sequencer primary/replica.
`
}

func (cmd *InitCmd) SetFlags(f *flag.FlagSet) {
	cmd.p2pport = f.String("p2pport", "24445", "P2P port to listen on")
}

func (cmd *InitCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	fmt.Println("Initializing a sequencer primary...\n")
	
	// - p2p private key
	// - sequencer private key for signing
	// - output sequencer pubkey for following
	// - output sequencer multiaddress

	cmd.initCore()
	cmd.initP2P()

	return 0
}

func (cmd *InitCmd) initCore() {
	// Core.
	privateKey, err := ethCrypto.GenerateKey()
	if err != nil {
		panic(fmt.Errorf("error generating private key:", err))
	}
	signer := utils.NewEthereumECDSASignerFromKey(privateKey)

	fmt.Printf("Core operator private key: %s\n", hexutil.Encode(ethCrypto.FromECDSA(privateKey)))
	fmt.Printf("Core operator public key: %s\n", hexutil.Encode(ethCrypto.FromECDSAPub(signer.GetPubkey())))
}

func (cmd *InitCmd) initP2P() {
	privateKey := sequencer.P2PGeneratePrivateKey()

	rawPrivateKey, err := libp2pCrypto.MarshalPrivateKey(privateKey)
	if err != nil {
		panic(fmt.Errorf("error getting private key raw data: %s", err))
	}

	p2pAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%s", *cmd.p2pport)
	host, err := libp2p.New(
		libp2p.ListenAddrStrings(p2pAddr),
		libp2p.Identity(privateKey),
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("P2P multiaddr: %s/p2p/%s\n", host.Addrs()[0], host.ID())
	fmt.Printf("P2P private key: %s\n", hexutil.Encode(rawPrivateKey))
}