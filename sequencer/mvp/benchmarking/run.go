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
	primary := sequencer.NewSequencerNode("file::memory:?cache=shared", "49000", "49001", sequencer.PrimaryMode, "0x0801124098bba74fbc32342624d74e8e523644be41d1e745b21af54933735ea6f0d92de17f7858dd065ece3d57a79a48b203664a63c356fb53c2dd3c5ce6a92aca4ebc39", "")

	// Create 5 replicas.
	replicas := make([]*sequencer.SequencerNode, 5)
	replicas[0] = sequencer.NewSequencerNode(":memory:", "49100", "49101", sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWJPxP7QYvfkDoHRXFirAixtvmy3dMjy1eszPza7oFqdgt")
	replicas[1] = sequencer.NewSequencerNode(":memory:", "49200", "49201", sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWJPxP7QYvfkDoHRXFirAixtvmy3dMjy1eszPza7oFqdgt")
	replicas[2] = sequencer.NewSequencerNode(":memory:", "49300", "49301", sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWJPxP7QYvfkDoHRXFirAixtvmy3dMjy1eszPza7oFqdgt")
	replicas[3] = sequencer.NewSequencerNode(":memory:", "49400", "49401", sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWJPxP7QYvfkDoHRXFirAixtvmy3dMjy1eszPza7oFqdgt")
	replicas[4] = sequencer.NewSequencerNode(":memory:", "49500", "49501", sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWJPxP7QYvfkDoHRXFirAixtvmy3dMjy1eszPza7oFqdgt")

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