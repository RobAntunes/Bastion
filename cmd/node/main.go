package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"bastion.network/bastion/pkg/p2p"
	"bastion.network/bastion/pkg/system/utils"
	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

const (
	PROTOCOL_ID = "/bastion/1.0.0"
	PORT_MIN    = 51001
	PORT_MAX    = 59998
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// Setup host and DHT
	contentID, err := cid.V1Builder{Codec: cid.Raw, MhType: multihash.SHA2_256}.Sum([]byte("bastion"))
	if err != nil {
		log.Fatalf("Failed to create content ID: %v", err)
	}

	bootstrapPeers, err := Start(3, 0, true)
	if err != nil {
		log.Fatalf("Failed to get bootstrap peers: %v", err)
	}
	bspch := make(chan string, len(bootstrapPeers)) // Buffer to number of peers for non-blocking writes.

	// Use WaitGroup to wait for all bootstrap node goroutines to finish.
	var wg sync.WaitGroup
	var connectWg sync.WaitGroup

	if len(os.Args) > 1 && os.Args[1] == "-b" {
		if err != nil {
			log.Fatalf("Failed to get peers: %v", err)
		}
		for i, provider := range bootstrapPeers {
			wg.Add(1) // Increment WaitGroup counter before starting a goroutine.

			// Put content on the DHT
			provider.DHT.PutValue(ctx, contentID.String(), []byte("I...AM...ALIVE!"))
			go func(ctx context.Context, provider *p2p.Peer, index int) {
				defer wg.Done()
				ActiveLoop(ctx, provider.H, provider.DHT, &contentID, true)
			}(ctx, provider, i)
		}
	} else {
		peers, err := Start(1, rand.Int63(), false)
		if err != nil {
			log.Fatalf("Failed to start peer: %v", err)
		}
		localPeer := peers[0]

		// Use WaitGroup to wait for all connection attempts to finish.

		for i, p := range bootstrapPeers {
			connectWg.Add(1) // Increment WaitGroup counter before starting a goroutine.

			// Put content on the DHT
			localPeer.DHT.PutValue(ctx, contentID.String(), []byte("I...AM...ALIVE!"))

			go func(ctx context.Context, p *p2p.Peer, idx int) {
				defer connectWg.Done() // Decrement counter when goroutine completes.
				ConnectToPeer(ctx, localPeer.H, p, bspch, idx)
				ActiveLoop(ctx, localPeer.H, localPeer.DHT, &contentID, false)
			}(ctx, p, i)
		}
	}

	go func() {
		for msg := range bspch {
			fmt.Println(msg) // Consumes messages as they arrive.
		}
	}()

	// Setup signal handling for graceful shutdown.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-c:
			fmt.Println("\nReceived shutdown signal")
			cancel() // Cancel the context.
		case <-ctx.Done():
			// Handle context cancellation.
		}
		wg.Wait()        // Wait for all bootstrap node goroutines to finish.
		connectWg.Wait() // Wait for all connection attempts to finish.
		// !!!
		// TODO: Consider waiting for other goroutines you might have.
		// !!!
		fmt.Println("Shutting down gracefully...")
		close(bspch) // Close channel after all writes are done and before program exit.
	}()

	<-ctx.Done() // Ensures main does not exit prematurely, waiting for signal handling or explicit cancellation.
}

func ActiveLoop(ctx context.Context, h host.Host, kdht *dht.IpfsDHT, contentID *cid.Cid, isBootstrap bool) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Println("Node active:", h.ID().String())

			// Bootstrap nodes don't need to connect to other peers.
			if !isBootstrap {
				// Advertise content
				if err := kdht.Provide(ctx, *contentID, true); err != nil {
					log.Printf("Error advertising content: %v", err)
				} else {
					log.Printf("Content advertised: %s", contentID.String())
				}

				if err := FindProvidersAndConnect(ctx, h, kdht, contentID); err != nil {
					log.Printf("Error finding providers and connecting: %v", err)
				}
			} else {
				// Bootstrap nodes only need to advertise content.
				if err := kdht.Provide(ctx, *contentID, true); err != nil {
					log.Printf("Error advertising content: %v", err)
				} else {
					log.Printf("Content advertised: %s", contentID.String())
				}
				fmt.Println("Peerlist:", kdht.RoutingTable().ListPeers())
			}
		case <-ctx.Done():
			fmt.Println("Node shutting down...")
			return

		}
	}
}

func ConnectToPeer(ctx context.Context, h host.Host, p *p2p.Peer, pch chan string, idx int) {
	err := h.Connect(ctx, peer.AddrInfo{
		ID:    p.H.ID(),
		Addrs: p.H.Addrs(),
	})

	if err != nil {
		fmt.Println("Error connecting to peer:", err)
		pch <- err.Error()
		return
	}

	// After connecting successfully
	select {
	case <-pch:
		return
	case <-ctx.Done():
		// Context canceled, possibly due to shutdown
		return
	default:
		// Non-blocking write to channel
		pch <- fmt.Sprintf("Connected to bootstrap peer %s", p.H.ID().String())
	}
}

// FindProvidersAndConnect searches for providers for the given content and attempts to connect to them.
func FindProvidersAndConnect(ctx context.Context, h host.Host, kdht *dht.IpfsDHT, contentID *cid.Cid) error {
	providers, err := kdht.FindProviders(ctx, *contentID)
	if err != nil {
		log.Printf("Error finding providers: %v", err)
		return err
	}
	fmt.Println("Found providers:", providers)
	for _, provider := range providers {
		if provider.ID == h.ID() {
			fmt.Println("skipping self...")
			continue // skip self
		}
		if err := h.Connect(ctx, provider); err != nil {
			log.Printf("Failed to connect to provider %s: %v", provider.ID, err)
		} else {
			log.Printf("Connected to provider %s", provider.ID)
			kdht.GetValue(ctx, contentID.String())
		}
	}
	return nil
}

// Creates a new host and DHT instance
func setupHosts(ctx context.Context, p *p2p.Peer, port int) (host.Host, *dht.IpfsDHT, error) {
	h, err := p2p.NewLibP2PHost(p, port)
	for err != nil {
		log.Printf("Failed to create libp2p host: %v", err)
		h, err = p2p.NewLibP2PHost(p, utils.RangeIn(PORT_MIN, PORT_MAX))
	}
	kdht, err := p2p.NewKDHT(ctx, h)
	if err != nil {
		log.Printf("Failed to create DHT: %v", err)
		return nil, nil, errors.New("failed to create DHT")
	}
	return h, kdht, nil
}

func Start(num int, seed int64, isBootstrap bool) ([]*p2p.Peer, error) {
	peers, err := p2p.GetPeers(num, seed)
	if err != nil {
		log.Printf("Failed to get peers: %v", err)
		return nil, err
	}
	peerList := make([]*p2p.Peer, num)
	var port int
	if isBootstrap {
		port = 4001
	} else {
		port = utils.RangeIn(PORT_MIN, PORT_MAX)
	}
	for i, peer := range peers {
		h, kdht, err := setupHosts(context.Background(), peer, port+i)
		if err != nil {
			log.Printf("Failed to set up host and DHT: %v", err)
			return nil, err
		}
		peerList[i] = &p2p.Peer{ID: peer.ID, PrivateKeyBuffer: peer.PrivateKeyBuffer, H: h, DHT: kdht}
	}
	return peerList, nil
}
