package sequencer

import (
	"context"
	"fmt"

	// "sync"
	// discovery "github.com/libp2p/go-libp2p-discovery"
	// dht "github.com/libp2p/go-libp2p-kad-dht"

	"github.com/ethereum/go-ethereum/common/hexutil"
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



func NewP2PNode(multiaddr string, privateKeyRaw string, bootstrapPeers AddrList) (*P2PNode, error) {
	var privateKey crypto.PrivKey
	var err error

	if privateKeyRaw == "" {
		privateKey, _, err = crypto.GenerateKeyPair(
			crypto.Ed25519, // Select your key type. Ed25519 are nice short
			-1,             // Select key length when possible (i.e. RSA).
		)
		if err != nil {
			panic(err)
		}

	} else {
		privateKeyBytes, err := hexutil.Decode(privateKeyRaw)
		if err != nil {
			panic(err)
		}

		privateKey, err = crypto.UnmarshalPrivateKey(privateKeyBytes)
		if err != nil {
			panic(err)
		}
	}

	host, err := libp2p.New(
		libp2p.ListenAddrStrings(multiaddr),
		libp2p.Identity(privateKey),
	)
	if err != nil {
		panic(err)
	}

	// Debug log on first setup.
	init := true
	if init {
		rawkey, err := crypto.MarshalPrivateKey(privateKey)
		if err != nil {
			panic(err)
		}

		fmt.Printf("P2P multiaddr: %s/p2p/%s\n", host.Addrs()[0], host.ID())
		fmt.Printf("P2P private key: %s\n", hexutil.Encode(rawkey))
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

// func startDHT(host libp2pHost.Host, bootstrapPeers AddrList, rendezVousString string, peerDiscoveryNotifee PeerDiscoveryNotifee) {
// 	// Start a DHT, for use in peer discovery. We can't just make a new DHT
// 	// client because we want each peer to maintain its own local copy of the
// 	// DHT, so that the bootstrapping node of the DHT can go down without
// 	// inhibiting future peer discovery.
// 	ctx := context.Background()
// 	kademliaDHT, err := dht.New(ctx, host)
// 	if err != nil {
// 		panic(err)
// 	}

// 	logger.Debug("Bootstrapping the DHT")
// 	if err = kademliaDHT.Bootstrap(ctx); err != nil {
// 		panic(err)
// 	}

// 	// Let's connect to the bootstrap nodes first. They will tell us about the
// 	// other nodes in the network.
// 	var wg sync.WaitGroup
// 	for i, peerAddr := range bootstrapPeers {
// 		logger.Debugf("connecting to bootstrap peer #%d on %s", i, peerAddr)

// 		peerinfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
// 		if err != nil {
// 			logger.Errorf("error with bootstrap peer: %s", err)
// 		}

// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()

// 			// host.Peerstore().AddAddrs(peerinfo.ID, peerinfo.Addrs, 300 * time.Second)
// 			if err := host.Connect(ctx, *peerinfo); err != nil {
// 				logger.Warning(err)
// 			} else {
// 				logger.Info("Connection established with bootstrap node:", *peerinfo)
// 			}
// 		}()
// 	}
// 	wg.Wait()

// 	// We use a rendezvous point "meet me here" to announce our location.
// 	// This is like telling your friends to meet you at the Eiffel Tower.
// 	logger.Info("Announcing ourselves...")
// 	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
// 	discovery.Advertise(ctx, routingDiscovery, rendezVousString)
// 	logger.Debug("Successfully announced!")

// 	// Now, look for others who have announced
// 	// This is like your friend telling you the location to meet you.
// 	logger.Debug("Searching for other peers...")
// 	peerChan, err := routingDiscovery.FindPeers(ctx, rendezVousString)
// 	if err != nil {
// 		panic(err)
// 	}

// 	go func() {
// 		for peer := range peerChan {
// 			if peer.ID == host.ID() {
// 				continue
// 			}
// 			fmt.Println("Found peer:", peer)

// 			fmt.Println("Connecting to:", peer)
// 			peerDiscoveryNotifee.HandlePeerFound(peer)
// 		}
// 	}()
// }

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

func (n *P2PNode) GossipNewBlock(block Block) {
	n.newBlocks.Publish(n.ctx, block.sequenceMsg)
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
			sequenceMsg: msg.Data,
		}

		fmt.Printf("pubsub - new block: %s\n", block)
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

