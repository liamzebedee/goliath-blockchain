package main

import (
	"fmt"
	"log"
	"time"

	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/utils"
	_ "github.com/mattn/go-sqlite3"
)

func generateMockSequenceTx(signer sequencer.Signer, nonce int) (sequencer.SequenceMessage) {
	msg := utils.ConstructSequenceMessage("foo", 5 * time.Second)
	msg.Nonce = string(nonce)
	msg = msg.SetFrom(signer.GetPubkey())
	msg = msg.Signed(signer)
	return msg
}

func main() {
	// Create primary node.
	primary := sequencer.NewSequencerNode("file::memory:?cache=shared", "49000", "49001", sequencer.PrimaryMode, "0x0801124098bba74fbc32342624d74e8e523644be41d1e745b21af54933735ea6f0d92de17f7858dd065ece3d57a79a48b203664a63c356fb53c2dd3c5ce6a92aca4ebc39", "")

	// Create 5 replicas.
	replicas := make([]*sequencer.SequencerNode, 5)
	replicas[0] = sequencer.NewSequencerNode(":memory:", "49100", "49101", sequencer.ReplicaMode, "", "/ip4/127.0.0.1/tcp/49001/p2p/12D3KooWJPxP7QYvfkDoHRXFirAixtvmy3dMjy1eszPza7oFqdgt")


	// Wait until they're connected for the test.
	waitConnectedP2P := make(chan bool)
	go func() {
		i := 0
		host := replicas[0].P2P.Host
		
		for true {
			conns := host.Network().Conns()
			fmt.Printf("waiting for connections (%d): %s\n", i, conns)
			i++
			
			if len(conns) > 0 {
				waitConnectedP2P <- true
				break
			}

			time.Sleep(10 * time.Millisecond)
		}
	}()


	// Start them up.
	go primary.Start()
	defer primary.Close()
	
	go replicas[0].Start()
	defer replicas[0].Close()

	<-waitConnectedP2P

	// Post 10K tps to the primary node.
	signer := utils.NewEthereumECDSASigner("3977045d27df7e401ecf1596fd3ae86b59f666944f81ba8dbf547c2269902f6b")
	// msgs := make([]sequencer.SequenceMessage, 1000)
	
	for i := 0; i < 3; i++ {
		msg := generateMockSequenceTx(signer, i)
		// msgs[i] = generateMockSequenceTx(signer)
		go (func(){
			_, err := primary.Seq.Sequence(msg.ToJSON())
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