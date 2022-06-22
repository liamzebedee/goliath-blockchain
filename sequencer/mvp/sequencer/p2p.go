package sequencer

import (
	"context"
	"fmt"
	"time"

	// "sync"
	// discovery "github.com/libp2p/go-libp2p-discovery"
	// dht "github.com/libp2p/go-libp2p-kad-dht"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/proto"
	"github.com/liamzebedee/goliath-blockchain/sequencer/mvp/sequencer/messages"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	libp2pHost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/ipfs/go-log/v2"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var logger = log.Logger("p2p")
const DHT_RENDEZVOUS_MAGIC = "goliath/sequencer/queen-st-hungry-jacks"
const PUBSUB_TOPIC_NEW_BLOCKS = "NewBlocks"
const PUBSUB_TOPIC_PEER_DISCOVERY = "PeerDiscovery"

type P2PNode struct {
	Host libp2pHost.Host
	ctx context.Context
	newBlocks *pubsub.Topic
	peerDiscovery *pubsub.Topic
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

	// Gossip Sub.
	// 
	// gossipSubParams := pubsub.DefaultGossipSubParams()
	// gossipSubParams.D = 2
	// gossipSubParams.MaxIHaveLength = 30000
	
	// pubsub, err := pubsub.NewGossipSub(
	// 	ctx, 
	// 	host,
	// 	pubsub.WithPeerExchange(true),
	// 	pubsub.WithDirectPeers(bootstrapPeerInfos),
	// 	// pubsub.WithGossipSubParams(gossipSubParams),
	// 	// pubsub.WithFloodPublish(true),
	// )

	pubsub, err := pubsub.NewFloodSub(
		ctx,
		host,
	)
	if err != nil {
		panic(err)
	}

	// Connect to bootstrap peers.
	for _, peerinfo := range(bootstrapPeerInfos) {
		err = host.Connect(context.Background(), peerinfo)
		if err != nil {
			fmt.Printf("error connecting to peer %s: %s\n", peerinfo.ID.Pretty(), err)
		}
	}

	// Join the pubsub topics.
	newBlocks, err := pubsub.Join(topicName(PUBSUB_TOPIC_NEW_BLOCKS))
	if err != nil {
		return nil, err
	}

	peerDiscovery, err := pubsub.Join(topicName(PUBSUB_TOPIC_PEER_DISCOVERY))
	if err != nil {
		return nil, err
	}

	node := &P2PNode{
		Host: host,
		ctx: ctx,
		newBlocks: newBlocks,
		peerDiscovery: peerDiscovery,
	}
	
	return node, nil
}

func (n *P2PNode) Start() {
	fmt.Printf("P2P listening on %s\n", n.Host.Addrs()[0])

	// go n.BroadcastPresenceRoutine()
	// go n.ListenForNewPeers()
}

// discoveryNotifee gets notified when we find a new peer
// HandlePeerFound connects to peers discovered. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *P2PNode) HandlePeerFound(peerinfo peer.AddrInfo) {
	isConnectedToPeer := len(n.Host.Peerstore().Addrs(peerinfo.ID)) > 0
	if isConnectedToPeer {
		return
	}

	fmt.Printf("connecting to new peer %s\n", peerinfo.ID.Pretty())
	err := n.Host.Connect(context.Background(), peerinfo)

	if err != nil {
		fmt.Printf("error connecting to peer %s: %s\n", peerinfo.ID.Pretty(), err)
	}
}

func (n *P2PNode) GossipNewBlock(block *messages.Block) {
	buf, err := proto.Marshal(block)
	if err != nil {
		// TODO: robust error handling?
		panic(fmt.Errorf("error encoding block: %s", err))
	}

	fmt.Println("pubsub - gossip block:", block.PrettyHash())
	err = n.newBlocks.Publish(n.ctx, buf)
	if err != nil {
		fmt.Println(fmt.Errorf("error gossipping new block: %s", err))
	}
}

func (n *P2PNode) ListenForNewBlocks(handler func(block *messages.Block)) {
	sub, err := n.newBlocks.Subscribe(
		pubsub.WithBufferSize(10000),
	)
	if err != nil {
		panic(err)
	}

	for {
		msg, err := sub.Next(n.ctx)
		if err != nil {
			// close(newBlockChan)
			return
		}

		block := &messages.Block{}
		proto.Unmarshal(msg.Data, block)

		fmt.Printf("pubsub - new block: %s\n", block.PrettyString())
		handler(block)
	}
}

func (n *P2PNode) BroadcastPresenceRoutine() {
	peerinfo := &peer.AddrInfo{
		ID: n.Host.ID(),
		Addrs: n.Host.Addrs(),
	}
	buf, err := peerinfo.MarshalJSON()
	if err != nil {
		panic(fmt.Errorf("error serialising my peerinfo: %s", err))
	}

	for {
		n.peerDiscovery.Publish(n.ctx, buf)

		// TODO, only republish info when numpeers falls below level or at random interval,
		time.Sleep(5 * time.Second)
	}
}

func (n *P2PNode) ListenForNewPeers() {
	sub, err := n.peerDiscovery.Subscribe()
	if err != nil {
		panic(err)
	}

	for {
		msg, err := sub.Next(n.ctx)
		if err != nil {
			panic(fmt.Errorf("error in ListenForNewPeers: %s", err))
			return
		}

		peerinfo := &peer.AddrInfo{}
		err = peerinfo.UnmarshalJSON(msg.Data)
		if err != nil {
			fmt.Println(fmt.Errorf("error reading peerinfo gossip: %s", err))
			continue
		}

		fmt.Printf("pubsub - peerinfo gossip: %s\n", peerinfo.String())
		n.HandlePeerFound(*peerinfo)
	}
}

func (n *P2PNode) Close() (error) {
	return n.Host.Close()
}

