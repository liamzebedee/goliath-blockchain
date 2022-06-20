package sequencer

import (
	"context"
	"fmt"

	// "sync"
	// discovery "github.com/libp2p/go-libp2p-discovery"
	// dht "github.com/libp2p/go-libp2p-kad-dht"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/proto"
	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/messages"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	libp2pHost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var logger = log.Logger("p2p")
const DHT_RENDEZVOUS_MAGIC = "goliath/sequencer/queen-st-hungry-jacks"
const PUBSUB_TOPIC_NEW_BLOCKS = "newblocks"

type P2PNode struct {
	Host libp2pHost.Host
	ctx context.Context
	newBlocks *pubsub.Topic
}

func P2PGeneratePrivateKey() (crypto.PrivKey) {
	privateKey, _, err := crypto.GenerateKeyPair(
		crypto.Ed25519,
		-1,
	)
	if err != nil {
		panic(err)
	}

	return privateKey
}

func P2PParsePrivateKey(raw string) (crypto.PrivKey) {
	privateKeyBytes, err := hexutil.Decode(raw)
	if err != nil {
		panic(err)
	}

	privateKey, err := crypto.UnmarshalPrivateKey(privateKeyBytes)
	if err != nil {
		panic(err)
	}
	return privateKey
}

func NewP2PNode(multiaddr string, privateKey crypto.PrivKey, bootstrapPeers AddrList) (*P2PNode, error) {
	if privateKey == nil {
		privateKey = P2PGeneratePrivateKey()
	}

	host, err := libp2p.New(
		libp2p.ListenAddrStrings(multiaddr),
		libp2p.Identity(privateKey),
	)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// create a new PubSub service using the GossipSub router
	bootstrapPeerInfos := []peer.AddrInfo{}
	for _, addr := range(bootstrapPeers) {
		peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		bootstrapPeerInfos = append(bootstrapPeerInfos, *peerinfo)
	}

	pubsub, err := pubsub.NewGossipSub(
		ctx, 
		host,
		pubsub.WithPeerExchange(true),
		pubsub.WithDirectPeers(bootstrapPeerInfos),
	)
	if err != nil {
		panic(err)
	}

	// join the pubsub topic
	newBlocks, err := pubsub.Join(topicName(PUBSUB_TOPIC_NEW_BLOCKS))
	if err != nil {
		return nil, err
	}

	node := &P2PNode{
		Host: host,
		ctx: ctx,
		newBlocks: newBlocks,
	}

	// discover peers for pubsub.
	// startDHT(host, bootstrapPeers, DHT_RENDEZVOUS_MAGIC, node)

	return node, nil
}

func (n *P2PNode) Start() {
	fmt.Printf("P2P listening on %s\n", n.Host.Addrs()[0])
}

// discoveryNotifee gets notified when we find a new peer
// HandlePeerFound connects to peers discovered. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *P2PNode) HandlePeerFound(peerinfo peer.AddrInfo) {
	fmt.Printf("discovered new peer %s\n", peerinfo.ID.Pretty())
	
	err := n.Host.Connect(context.Background(), peerinfo)
	// n.host.Peerstore().AddAddrs(peerinfo.ID, peerinfo.Addrs, 300 * time.Second)
	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", peerinfo.ID.Pretty(), err)
	}
}

func (n *P2PNode) GossipNewBlock(block *messages.Block) {
	buf, err := proto.Marshal(block)
	if err != nil {
		// TODO: robust error handling?
		panic(fmt.Errorf("error encoding block:", err))
	}

	n.newBlocks.Publish(n.ctx, buf)
}

func (n *P2PNode) ListenForNewBlocks(newBlockChan chan *messages.Block) {
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

		block := &messages.Block{}
		proto.Unmarshal(msg.Data, block)

		fmt.Printf("pubsub - new block: %s\n", block.PrettyHash())
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
	return n.Host.Close()
}

