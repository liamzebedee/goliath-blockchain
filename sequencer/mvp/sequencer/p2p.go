package sequencer

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
	libp2pHost "github.com/libp2p/go-libp2p-core/host"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type P2PNode struct {
	host libp2pHost.Host
	ctx context.Context
	newBlocks *pubsub.Topic
}

func NewP2PNode(multiaddr string) (*P2PNode, error) {
	host, err := libp2p.New(
		libp2p.ListenAddrStrings(multiaddr),
	)

	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// create a new PubSub service using the GossipSub router
	pubsub, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		panic(err)
	}

	// join the pubsub topic
	newBlocks, err := pubsub.Join(topicName("newblocks"))
	if err != nil {
		return nil, err
	}


	node := &P2PNode{
		host: host,
		ctx: ctx,
		newBlocks: newBlocks,
	}

	return node, nil
}

func (n *P2PNode) Start() {
	// err := n.host.Network().Listen()
	fmt.Printf("P2P listening on %s\n", n.host.Addrs()[0])
	// if err != nil {
	// 	panic(fmt.Errorf("error listening on p2p: %s", err))
	// }
}

func topicName(name string) string {
	return "goliath-sequencer/" + name
}

func (n *P2PNode) GossipNewBlocks(newBlockChan chan Block) {
	for {
		block := <- newBlockChan
		n.newBlocks.Publish(n.ctx, block.data)
	}
}

func (n *P2PNode) ListenForNewBlocks(newBlockChan chan Block) {
	sub, err := n.newBlocks.Subscribe()
	if err != nil {
		panic(err)
	}

	for {
		msg, err := sub.Next(n.ctx)
		if err != nil {
			close(newBlockChan)
			return
		}

		block := Block{
			data: msg.Data,
		}

		fmt.Printf("new block: %s\n", block)
		newBlockChan <- block

		// only forward messages delivered by others
		// if msg.ReceivedFrom == cr.self {
		// 	continue
		// }

		// cm := new(ChatMessage)
		// err = json.Unmarshal(msg.Data, cm)
		// if err != nil {
		// 	continue
		// }
		// // send valid messages onto the Messages channel
		// cr.Messages <- cm
	}
}

func (n *P2PNode) Close() (error) {
	return n.host.Close()
}