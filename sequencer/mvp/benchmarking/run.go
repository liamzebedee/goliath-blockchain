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
	// Create primary node.
	primary := sequencer.NewSequencerNode("file::memory:?cache=shared", "49000", "49001", sequencer.PrimaryMode, "0x08011240e6d9a1faa2fbf1e669169b8813e4439c5d304f82bccdf6a8da30d7e1679edd6e9ca03937ad7b1c86347c24db827cfd0da2743e4946d7437ed6e1571560cad484", "", "7345f1ef889724e400fedd3822f45fca1cfb09c1176c8538c9243b4b2461bec3")

	// Create 5 replicas.
	replicas := make([]*sequencer.SequencerNode, 5)
	replicas[0] = sequencer.NewSequencerNode(":memory:", "49100", "49101", sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWLMmULYCrke9PiATTDTmE4pMDCtxiffWTTM3mhTXgfw2K", "")
	replicas[1] = sequencer.NewSequencerNode(":memory:", "49200", "49201", sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWLMmULYCrke9PiATTDTmE4pMDCtxiffWTTM3mhTXgfw2K", "")
	replicas[2] = sequencer.NewSequencerNode(":memory:", "49300", "49301", sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWLMmULYCrke9PiATTDTmE4pMDCtxiffWTTM3mhTXgfw2K", "")
	replicas[3] = sequencer.NewSequencerNode(":memory:", "49400", "49401", sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWLMmULYCrke9PiATTDTmE4pMDCtxiffWTTM3mhTXgfw2K", "")
	replicas[4] = sequencer.NewSequencerNode(":memory:", "49500", "49501", sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWLMmULYCrke9PiATTDTmE4pMDCtxiffWTTM3mhTXgfw2K", "")

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
	
	for i := 0; i < 5; i++ {
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