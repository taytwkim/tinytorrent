package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// this is the name of the topic nodes can subscribe to.
// 1.0.0 is usually used to indicate versions in case we change how
// announcements are structured.
// if a node is outdated, it will simply subscribe to the outdated topic
// instead of crashing the new topic.
const announceTopic = "p2pfs/announce/1.0.0"

// Announcement is the JSON payload sent over GossipSub
type Announcement struct {
	PeerID     string   `json:"peer_id"`
	Multiaddrs []string `json:"multiaddrs"`
	Files      []string `json:"files"`
}

func (n *Node) setupDiscovery() error {
	// Join "joins" the topic
	topic, err := n.PubSub.Join(announceTopic)
	if err != nil {
		return err
	}

	// Subscribe sets up a queue that collects new messages from the topic
	// You subscribe only when you are trying to listen.
	// if a node just wants to broadcast, it doesn't have to subscribe.
	sub, err := topic.Subscribe()
	if err != nil {
		return err
	}

	// start running these as go routines in the background
	go n.announceLoop(topic) // I have files x, y, z!
	go n.listenLoop(sub)

	return nil
}

// broadcasts which files this node has
// wrapper for broadcastAnnounce
func (n *Node) announceLoop(topic *pubsub.Topic) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Wait a bit before first announce to let connections establish
	time.Sleep(2 * time.Second)
	n.broadcastAnnounce(topic)

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			n.broadcastAnnounce(topic)
		}
	}
}

func (n *Node) broadcastAnnounce(topic *pubsub.Topic) {
	n.localFilesLock.RLock()
	var files []string
	for f := range n.LocalFiles {
		files = append(files, f)
	}
	n.localFilesLock.RUnlock()

	var addrs []string
	for _, addr := range n.Host.Addrs() {
		addrs = append(addrs, addr.String())
	}

	msg := Announcement{
		PeerID:     n.Host.ID().String(),
		Multiaddrs: addrs,
		Files:      files,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling announcement: %v", err)
		return
	}

	if err := topic.Publish(n.ctx, payload); err != nil {
		log.Printf("Error publishing announcement: %v", err)
	} else {
		log.Printf("Announced %d files", len(files))
	}
}

func (n *Node) listenLoop(sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(n.ctx)
		if err != nil {
			if n.ctx.Err() != nil {
				return // Context cancelled
			}
			log.Printf("Error reading from sub: %v", err)
			continue
		}

		// Don't process our own messages
		if msg.ReceivedFrom == n.Host.ID() {
			continue
		}

		var ann Announcement
		if err := json.Unmarshal(msg.Data, &ann); err != nil {
			log.Printf("Failed to unmarshal announcement from %s: %v", msg.ReceivedFrom, err)
			continue
		}

		pid, err := peer.Decode(ann.PeerID)
		if err != nil {
			continue
		}

		var maddrList []multiaddr.Multiaddr
		for _, a := range ann.Multiaddrs {
			m, err := multiaddr.NewMultiaddr(a)
			if err == nil {
				maddrList = append(maddrList, m)
			}
		}

		// Update providers index
		n.providersLock.Lock()
		for _, file := range ann.Files {
			if n.Providers[file] == nil {
				n.Providers[file] = make(map[peer.ID]PeerProvider)
			}

			// We only need the peer ID for routing in libp2p usually,
			// but storing Full AddrInfo is good given we have Multiaddrs.
			n.Providers[file][pid] = PeerProvider{
				Info:     peer.AddrInfo{ID: pid, Addrs: maddrList}, // Simplified for MVP; real app would parse Multiaddrs into Info.Addrs
				LastSeen: time.Now(),
			}
		}
		n.providersLock.Unlock()
		log.Printf("Updated provider index from %s", pid.String()[:8])
	}
}

// ---------------------------------------------------------
// Daemon-side RPC Handlers for CLI Commands
// ---------------------------------------------------------

type P2PFSAPI struct {
	node *Node
}

type WhohasArgs struct {
	Filename string
}

type WhohasReply struct {
	Peers []string
}

func (api *P2PFSAPI) Whohas(args *WhohasArgs, reply *WhohasReply) error {
	// Note that for the MVP, the node can just check its provider map because
	// the node should already have this info from the subscribed topic

	api.node.providersLock.RLock()
	defer api.node.providersLock.RUnlock()

	peers := api.node.Providers[args.Filename]
	for pid := range peers {
		reply.Peers = append(reply.Peers, pid.String())
	}
	return nil
}

type FetchArgs struct {
	Filename string
	PeerID   string
}

type FetchReply struct {
	Success bool
}

func (api *P2PFSAPI) Fetch(args *FetchArgs, reply *FetchReply) error {
	// The node implements the actual protocol call
	err := api.node.doFetch(args.Filename, args.PeerID)
	if err == nil {
		reply.Success = true
	}
	return err
}

type ListArgs struct {
	TargetAddr string
}

type ListReply struct {
	Files []string
}

func (api *P2PFSAPI) List(args *ListArgs, reply *ListReply) error {
	files, err := api.node.doList(args.TargetAddr)
	if err == nil {
		reply.Files = files
	}
	return err
}

func (n *Node) startRPCServer() error {
	api := &P2PFSAPI{node: n}
	rpcServer := rpc.NewServer()
	rpcServer.Register(api)

	// Clean up old socket if it exists
	os.Remove(n.RpcSocket)

	l, err := net.Listen("unix", n.RpcSocket)
	if err != nil {
		return err
	}
	n.rpcListener = l

	go func() {
		defer l.Close()
		http.Serve(l, rpcServer)
	}()

	log.Printf("RPC Server listening on %s", n.RpcSocket)
	return nil
}

// ---------------------------------------------------------
// CLI Client-side wrapper for connecting to Daemon RPC
// ---------------------------------------------------------

type Client struct {
	rpcClient *rpc.Client
}

func NewClient(rpcSocket string) *Client {
	c, err := rpc.DialHTTP("unix", rpcSocket)
	if err != nil {
		log.Fatalf("Daemon not running or could not connect: %v", err)
	}
	return &Client{rpcClient: c}
}

func (c *Client) Whohas(filename string) ([]string, error) {
	args := &WhohasArgs{Filename: filename}
	var reply WhohasReply
	err := c.rpcClient.Call("P2PFSAPI.Whohas", args, &reply)
	return reply.Peers, err
}

func (c *Client) Fetch(filename, peerID string) error {
	args := &FetchArgs{Filename: filename, PeerID: peerID}
	var reply FetchReply
	return c.rpcClient.Call("P2PFSAPI.Fetch", args, &reply)
}

func (c *Client) List(targetAddr string) ([]string, error) {
	args := &ListArgs{TargetAddr: targetAddr}
	var reply ListReply
	err := c.rpcClient.Call("P2PFSAPI.List", args, &reply)
	return reply.Files, err
}
