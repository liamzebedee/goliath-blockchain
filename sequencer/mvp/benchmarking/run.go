package main

import (
	"fmt"
	"log"
	"time"

	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/messages"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/utils"
	_ "github.com/mattn/go-sqlite3"
)

func generateMockSequenceTx(signer utils.Signer, nonce int) (*messages.SequenceTx) {
	msg := messages.ConstructSequenceMessage("0x4200", 5 * time.Second)
	msg.Nonce = []byte(string(nonce))
	msg.SetFrom(signer.GetPubkey())
	msg = msg.Signed(signer)
	return msg
}

func main() {
	// Configuration.
	numReplicas := 1
	numSequenceTxs := 3

	// Create primary node.
	primary := sequencer.NewSequencerNode("file::memory:?cache=shared", "49000", "49001", sequencer.PrimaryMode, "0x08011240e6d9a1faa2fbf1e669169b8813e4439c5d304f82bccdf6a8da30d7e1679edd6e9ca03937ad7b1c86347c24db827cfd0da2743e4946d7437ed6e1571560cad484", "", "3fd7f88cb790c6a8b54d4e1aaebba6775f427bb8fa2276e933b7c3440f164caa")

	// Create 5 replicas.
	replicas := make([]*sequencer.SequencerNode, numReplicas)
	for i, _ := range(replicas) {
		// Sets up some ports which are human-readable, in case we want to read the libp2p logs.
		// node i=0 rpcPort=49100 p2pPort=49101
		// node i=1 rpcPort=49200 p2pPort=49201 
		// node i=2 rpcPort=49300 p2pPort=49301 
		// etc.
		rpcPort := 49100 + i*100
		p2pPort := 49101 + i*100
		replicas[i] = sequencer.NewSequencerNode(":memory:", fmt.Sprint(rpcPort), fmt.Sprint(p2pPort), sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWLMmULYCrke9PiATTDTmE4pMDCtxiffWTTM3mhTXgfw2K", "")
	}
	
	// Wait until they're connected for the test.
	waitConnectedP2P := make(chan bool)
	go func() {
		i := 0
		host := primary.P2P.Host

		for true {
			conns := host.Network().Conns()
			fmt.Printf("waiting for connections (%d): %s\n", i, conns)
			i++
			
			if len(conns) > 0 {
				waitConnectedP2P <- true
				break
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()


	// Start them up.
	go primary.Start()
	defer primary.Close()

	for _, replica := range replicas {
		go replica.Start()
		defer replica.Close()
	}
	
	<-waitConnectedP2P

	// Post 10K tps to the primary node.
	signer := utils.NewEthereumECDSASigner("3fd7f88cb790c6a8b54d4e1aaebba6775f427bb8fa2276e933b7c3440f164caa")
	// msgs := make([]sequencer.SequenceMessage, 1000)
	
	for i := 0; i < numSequenceTxs; i++ {
		msg := generateMockSequenceTx(signer, i)
		// msgs[i] = generateMockSequenceTx(signer)
		go (func(){
			_, err := primary.Seq.Sequence(msg.ToHex())
			if err != nil {
				log.Fatal(err)
			}
		})()
	}

	wait := make(chan bool)
	<-wait
	// Test the pubsub.
	// time.Sleep(10 * time.Second)
}