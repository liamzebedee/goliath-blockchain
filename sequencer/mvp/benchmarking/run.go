package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/messages"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/utils"
	"github.com/libp2p/go-libp2p-core/host"
	_ "github.com/mattn/go-sqlite3"
)

func generateMockSequenceTx(signer utils.Signer, nonce int) (*messages.SequenceTx) {
	msg := messages.ConstructSequenceMessage("0x4200", 25 * time.Hour)
	msg.Nonce = []byte(string(nonce))
	msg.SetFrom(signer.GetPubkey())
	msg = msg.Signed(signer)
	return msg
}

func main() {
	// Configuration.
	// numReplicas := 0
	// numSequenceTxs := 10000

	// Simulate the peer table after it's all done.
	// simulate(5, 10000, 10 * time.Second)
	simulate(16, 10000, 20 * time.Second)
	// simulate(16, 5, 5 * time.Second)
}

func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}


func waitForPeeringsConnected(host host.Host, num int) (chan bool) {
	waitConnectedP2P := make(chan bool)
	
	go func() {
		i := 0

		for true {
			conns := host.Network().Peers()
			fmt.Printf("[host %s] waiting for connections (%d): len=%d\n", host.ID().ShortString(), i, len(conns))
			i++
			
			if len(conns) >= num {
				waitConnectedP2P <- true
				break
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	return waitConnectedP2P
}

func simulate(numReplicas, numSequenceTxs int, waitDuration time.Duration) {
	// Create primary node.
	// file::memory:?cache=shared
	// primary := sequencer.NewSequencerNode("file:swag?cache=shared", "49000", "49001", sequencer.PrimaryMode, "0x08011240e6d9a1faa2fbf1e669169b8813e4439c5d304f82bccdf6a8da30d7e1679edd6e9ca03937ad7b1c86347c24db827cfd0da2743e4946d7437ed6e1571560cad484", "", "3fd7f88cb790c6a8b54d4e1aaebba6775f427bb8fa2276e933b7c3440f164caa")

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

	// Start them up.
	// go primary.Start()
	// defer primary.Close()

	for _, replica := range replicas {
		go replica.Start()
		defer replica.Close()
	}

	// Wait until they're connected for the test.
	// Primary - wait for num(replicas) / 2
	// <-waitForPeeringsConnected(primary.P2P.Host, min(numReplicas, 24))
	// <-waitForPeeringsConnected(primary.P2P.Host, min(5, 24))
	
	// Replicas - wait for at least 1 node
	var wg sync.WaitGroup
	for _, replica := range replicas {
		wg.Add(1)
		go func(replica *sequencer.SequencerNode){
			<-waitForPeeringsConnected(replica.P2P.Host, min(1, 24))
			// <-waitForPeeringsConnected(replica.P2P.Host, min(8, 24))
			wg.Done()
		}(replica)
	}
	wg.Wait()
	

	// Post 10K tps to the primary node.
	signer := utils.NewEthereumECDSASigner("3fd7f88cb790c6a8b54d4e1aaebba6775f427bb8fa2276e933b7c3440f164caa")
	msgs := make([]*messages.SequenceTx, numSequenceTxs)
	
	for i := 0; i < numSequenceTxs; i++ {
		msg := generateMockSequenceTx(signer, i)
		msgs[i] = msg
	}
	batchSize := 250


	rpcclient, err := rpc.Dial("http://localhost:49000")
	if err != nil {
		panic(fmt.Errorf("can't setup rpc: %s", err))
	}


	for i := 0; i < batchSize; i += batchSize {
		for _, msg := range msgs {
			// _, err := primary.Seq.Sequence(msg.ToHex())

			var res interface{}
			err = rpcclient.Call(res, "sequencer_sequence", msg.ToHex())

			if err != nil {
				log.Fatal(err)
			}
		}

		time.Sleep(25 * time.Millisecond)
	}
	

	wait := make(chan bool)
	go func(){
		time.Sleep(waitDuration)
		wait <- true
	}()

	<-wait

	for i, rep := range(replicas) {
		fmt.Printf(
			"seq #%3d: lastBlock=%5d waitingOn=%5d totalBlocks=%5d n_peers=%3d\n", 
			i, 
			rep.Seq.LastBlock.Height, 
			rep.Seq.WaitedBlocks(10000000),
			rep.Seq.TotalSeen, 
			len(rep.P2P.Host.Network().Peers()),
		)
	}

	fmt.Println("Core info")
}