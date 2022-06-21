package sequencer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	// "sync"
	// discovery "github.com/libp2p/go-libp2p-discovery"
	// dht "github.com/libp2p/go-libp2p-kad-dht"

	"github.com/liamzebedee/goliath/mvp/sequencer/sequencer/messages"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	libp2pHost "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-msgio/protoio"
	"github.com/whyrusleeping/timecache"
)

// The basic structure of the network.
// 10K tps
// size(tx) = 256 bytes
// rate = 2.5mb/s
// ec2 instance transmit speed of up to 1 Gigabit = ~125MB/s
// 125/2.5 = 50 outgoing streams
// ok so let's say we have one main sequencer, one replica, and then the replica streams to a bunch of peers
// makes sense.
// how can we make the ux seamless? how does it work?
// basically, each node knows which nodes have free streaming slots
// nodes can exchange this information.
// e.g.
// max_outstreams = 8

type P2PNode2 struct {
	Host libp2pHost.Host
	ctx context.Context

	protocol *P2PProtocol
	newBlocks chan messages.Block
	// newBlocks *pubsub.Topic
	// peerDiscovery *pubsub.Topic
}

// DefaultMaximumMessageSize is 1mb.
const DefaultMaxMessageSize = 1 << 20


type P2PProtocol struct {
	// atomic counter for seqnos
	// NOTE: Must be declared at the top of the struct as we perform atomic
	// operations on this field.
	//
	// See: https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	counter uint64

	host libp2pHost.Host

	// maxMessageSize is the maximum message size; it applies globally to all
	// topics.
	// maxMessageSize int

	// size of the outbound message channel that we maintain for each peer
	// peerOutboundQueueSize int

	// incoming messages from other peers
	incoming chan *messages.P2PMessage

	// a notification channel for new peer connections accumulated
	// newPeers       chan struct{}
	// newPeersPrioLk sync.RWMutex
	// newPeersMx     sync.Mutex
	// newPeersPend   map[peer.ID]struct{}

	// a notification channel for new outoging peer streams
	// newPeerStream chan network.Stream

	// // a notification channel for errors opening new peer streams
	// newPeerError chan peer.ID

	// // a notification channel for when our peers die
	// peerDead       chan struct{}
	// peerDeadPrioLk sync.RWMutex
	// peerDeadMx     sync.Mutex
	// peerDeadPend   map[peer.ID]struct{}
	// // backoff for retrying new connections to dead peers
	// deadPeerBackoff *backoff


	// sendMsg handles messages that have been validated
	sendMsg chan *messages.P2PMessage

	// peer blacklist
	// blacklist     Blacklist
	// blacklistPeer chan peer.ID

	peers []peer.ID
	// peers map[peer.ID]chan *messages.P2PMessage

	inboundStreamsMx sync.Mutex
	inboundStreams   map[peer.ID]network.Stream

	seenMessagesMx sync.Mutex
	seenMessages   *timecache.TimeCache
	seenMsgTTL     time.Duration

	// generator used to compute the ID for a message
	// idGen *msgIDGenerator

	// // key for signing messages; nil when signing is disabled
	// signKey crypto.PrivKey
	// // source ID for signed messages; corresponds to signKey, empty when signing is disabled.
	// // If empty, the author and seq-nr are completely omitted from the messages.
	// signID peer.ID
	// // strict mode rejects all unsigned messages prior to validation
	// signPolicy MessageSignaturePolicy


	newBlockHandler func(block *messages.Block)

	ctx context.Context
}

const protocolId = protocol.ID("/goliath/sequencer/v0.1.0")

func NewP2PProtocol(ctx context.Context, h libp2pHost.Host) (*P2PProtocol) {
	i := &P2PProtocol{
		host:                  h,
		ctx:                   ctx,
		incoming:              make(chan *messages.P2PMessage, 32),
		// newPeers:              make(chan struct{}, 1),
		// newPeersPend:          make(map[peer.ID]struct{}),
		// newPeerStream:         make(chan network.Stream),
		// newPeerError:          make(chan peer.ID),
		// peerDead:              make(chan struct{}, 1),
		// peerDeadPend:          make(map[peer.ID]struct{}),
		// deadPeerBackoff:       newBackoff(ctx, 1000, BackoffCleanupInterval, MaxBackoffAttempts),
		sendMsg:               make(chan *messages.P2PMessage, 32),
		// peers:                 make(map[peer.ID]chan *messages.P2PMessage),
		peers:                 []peer.ID{},
		inboundStreams:        make(map[peer.ID]network.Stream),
	}
	go i.processLoop(i.ctx)
	return i
}

func (p *P2PProtocol) handleNewStream(s network.Stream) {
	peer := s.Conn().RemotePeer()

	fmt.Println("new stream")

	p.inboundStreamsMx.Lock()
	p.peers = append(p.peers, peer)
	other, dup := p.inboundStreams[peer]
	if dup {
		// log.Debugf("duplicate inbound stream from %s; resetting other stream", peer)
		other.Reset()
	}
	p.inboundStreams[peer] = s
	p.inboundStreamsMx.Unlock()

	defer func() {
		p.inboundStreamsMx.Lock()
		if p.inboundStreams[peer] == s {
			delete(p.inboundStreams, peer)
		}
		p.inboundStreamsMx.Unlock()
	}()

	r := protoio.NewDelimitedReader(s, DefaultMaxMessageSize)
	for {
		msg := &messages.P2PMessage{}
		err := r.ReadMsg(msg)

		if err != nil {
			if err != io.EOF {
				s.Reset()
				fmt.Println("error reading rpc from %s: %s", s.Conn().RemotePeer(), err)
			} else {
				// Just be nice. They probably won't read this
				// but it doesn't hurt to send it.
				s.Close()
			}

			return
		}

		// rpc.from = peer
		select {
		case p.incoming <- msg:
		case <-p.ctx.Done():
			// Close is useless because the other side isn't reading.
			s.Reset()
			return
		}
	}
}

// processLoop handles all inputs arriving on the channels
func (p *P2PProtocol) processLoop(ctx context.Context) {
	defer func() {
		// Clean up go routines.
		// for _, ch := range p.peers {
		// 	close(ch)
		// }
		// p.peers = nil
		// p.topics = nil
	}()

	for {
		select {
		case msg := <-p.incoming:
			p.recvMessage(msg)
		case msg := <-p.sendMsg:
			p.sendMessage(msg)
		case <-ctx.Done():
			// log.Info("pubsub processloop shutting down")
			return
		}
	}
}

func (p *P2PProtocol) recvMessage(msg *messages.P2PMessage) {
	if block := msg.GetBlock(); block != nil {
		fmt.Printf("pubsub - new block: %s\n", block.PrettyHash())
		if p.newBlockHandler != nil {
			p.newBlockHandler(block)
		}
	}
}

func (proto *P2PProtocol) AddPeer(pid peer.ID) {
	proto.peers = append(proto.peers, pid)
}

func (proto *P2PProtocol) sendMessage(msg *messages.P2PMessage) {
	for _, peer := range proto.peers {
		stream, err := proto.host.NewStream(proto.ctx, peer, protocolId)
		if err != nil {
			fmt.Println("error opening stream:", err)
			continue
		}

		// var buf []byte
		// buff := bytes.NewBuffer(buf)
		
		// w := protoio.NewDelimitedWriter(bufw)
		// err = w.WriteMsg(msg)
		
		// if DefaultMaxMessageSize < buff.Len() {
		// 	// message too big
		// 	// panic
		// 	panic("msg too big")
		// }

		bufw := bufio.NewWriter(stream)
		w := protoio.NewDelimitedWriter(bufw)
		err = w.WriteMsg(msg)
		if err != nil {
			fmt.Println("error writing to stream:", err)
			continue
		}

		bufw.Flush()
		if err != nil {
			fmt.Println("error writing to stream:", err)
			continue
		}
	}

	// from := msg.ReceivedFrom
	// topic := msg.GetTopic()
	// out := rpcWithMessages(msg.Message)
	// for pid := range fs.p.topics[topic] {
	// 	if pid == from || pid == peer.ID(msg.GetFrom()) {
	// 		continue
	// 	}

	// 	mch, ok := fs.p.peers[pid]
	// 	if !ok {
	// 		continue
	// 	}

	// 	select {
	// 	case mch <- out:
	// 		fs.tracer.SendRPC(out, pid)
	// 	default:
	// 		log.Infof("dropping message to peer %s: queue full", pid)
	// 		fs.tracer.DropRPC(out, pid)
	// 		// Drop it. The peer is too slow.
	// 	}
	// }

	// Really we just want to multicast this to all of our peers.
}

func (proto *P2PProtocol) publishBlock(block *messages.Block) (error) {
	proto.sendMsg <- &messages.P2PMessage{
		Block: block,
	}
	return nil
}

func NewP2PNode2(multiaddr string, privateKey crypto.PrivKey, bootstrapPeers AddrList) (*P2PNode2, error) {
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

	// Setup protocol.
	protocol := NewP2PProtocol(ctx, host)
	host.SetStreamHandler(protocolId, protocol.handleNewStream)

	// Parse bootstrap peers.
	bootstrapPeerInfos := []peer.AddrInfo{}
	for _, addr := range(bootstrapPeers) {
		peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		bootstrapPeerInfos = append(bootstrapPeerInfos, *peerinfo)
	}

	// Connect to bootstrap peers.
	for _, peerinfo := range(bootstrapPeerInfos) {
		err = host.Connect(context.Background(), peerinfo)
		if err != nil {
			fmt.Printf("error connecting to peer %s: %s\n", peerinfo.ID.Pretty(), err)
		}

		protocol.AddPeer(peerinfo.ID)
	}


	node := &P2PNode2{
		Host: host,
		ctx: ctx,
		protocol: protocol,
	}
	
	return node, nil
}

func (n *P2PNode2) Start() {
	fmt.Printf("P2P listening on %s\n", n.Host.Addrs()[0])
}

// discoveryNotifee gets notified when we find a new peer
// HandlePeerFound connects to peers discovered. Once they're connected,
// the PubSub system will automatically start interacting with them if they also
// support PubSub.
func (n *P2PNode2) HandlePeerFound(peerinfo peer.AddrInfo) {
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

func (n *P2PNode2) GossipNewBlock(block *messages.Block) {
	fmt.Println("pubsub - gossip block:", block.PrettyHash())
	
	err := n.protocol.publishBlock(block)

	if err != nil {
		fmt.Println(fmt.Errorf("error gossipping new block: %s", err))
	}
}



func (n *P2PNode2) ListenForNewBlocks(handler func(*messages.Block)) {
	n.protocol.newBlockHandler = func(block *messages.Block) {
		fmt.Printf("pubsub - new block: %s\n", block.PrettyHash())
		handler(block)
	}
}



func (n *P2PNode2) Close() (error) {
	return n.Host.Close()
}

