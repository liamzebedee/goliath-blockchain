package main

import (
	"log"
	"time"

	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/utils"
	_ "github.com/mattn/go-sqlite3"
)

func generateMockSequenceTx(signer sequencer.Signer) (sequencer.SequenceMessage) {
	msg := utils.ConstructSequenceMessage("foo", 5 * time.Second)
	msg = msg.SetFrom(signer.GetPubkey())
	msg = msg.Signed(signer)
	return msg
}

func main() {
	// Create primary node.
	primary := sequencer.NewSequencerNode("file::memory:?cache=shared", "49000", "49001", sequencer.PrimaryMode)

	// Create 5 replicas.
	replicas := make([]*sequencer.SequencerNode, 5)
	replicas[0] = sequencer.NewSequencerNode(":memory:", "49100", "49101", sequencer.ReplicaMode)

	// Start them up.
	go primary.Start()
	// go replicas[0].Start()

	defer primary.Close()
	// defer replicas[0].Close()

	// Add some transactions to the sequencer.
	signer := utils.NewEthereumECDSASigner("3977045d27df7e401ecf1596fd3ae86b59f666944f81ba8dbf547c2269902f6b")
	// msgs := make([]sequencer.SequenceMessage, 1000)
	
	for i := 0; i < 1000; i++ {
		msg := generateMockSequenceTx(signer)
		// msgs[i] = generateMockSequenceTx(signer)
		go (func(){
			_, err := primary.Seq.Sequence(msg.ToJSON())
			if err != nil {
				log.Fatal(err)
			}
		})()
	}

	

	time.Sleep(10 * time.Second)

	// Post 10K tps to the primary node.
	// Test the pubsub.
}