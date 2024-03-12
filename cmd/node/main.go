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
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"
)

const (
	PROTOCOL_ID = "/bastion/1.0.0"
	PORT_MIN    = 51001
	PORT_MAX    = 59998
)

type Node struct {
	IsBootstrap    bool
	BootstrapPeers *[]*p2p.Peer
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context cancellation on function exit

	contentID, err := cid.V1Builder{Codec: cid.Raw, MhType: multihash.SHA2_256}.Sum([]byte("bastion"))
	if err != nil {
		log.Fatalf("Failed to create content ID: %v", err)
	}

	bootstrapPeers, err := Start(3, 0, true)
	if err != nil {
		log.Fatalf("Failed to get bootstrap peers: %v", err)
	}

	var wg sync.WaitGroup
	comms := make(chan string, 10) // Adjust buffer size as needed

	// Handle bootstrap or regular mode
	if len(os.Args) > 1 && os.Args[1] == "-b" {
		for _, provider := range bootstrapPeers {
			wg.Add(1)
			go func(provider *p2p.Peer) {
				defer wg.Done()
				ActiveLoop(ctx, provider.H, provider.DHT, &contentID, &Node{IsBootstrap: true, BootstrapPeers: &bootstrapPeers}, comms)
			}(provider)
		}
	} else {
		peers, err := Start(1, rand.Int63(), false)
		if err != nil {
			log.Fatalf("Failed to start peer: %v", err)
		}
		localPeer := peers[0]

		wg.Add(1)
		go func() {
			defer wg.Done()
			ActiveLoop(ctx, localPeer.H, localPeer.DHT, &contentID, &Node{IsBootstrap: false, BootstrapPeers: &bootstrapPeers}, comms)
		}()
	}

	// Consume comms channel in a separate goroutine
	go func() {
		for msg := range comms {
			fmt.Println(msg)
		}
	}()

	// Setup signal handling for graceful shutdown.
	setupSignalHandling(ctx, cancel, &wg, comms)

	<-ctx.Done()
}

func setupSignalHandling(ctx context.Context, cancelFunc context.CancelFunc, wg *sync.WaitGroup, comms chan string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-c:
			fmt.Println("\nReceived shutdown signal")
			cancelFunc()
		case <-ctx.Done():
			// Context canceled, possibly due to shutdown
			wg.Wait()
			fmt.Println("Shutting down gracefully...")
			close(comms) // Ensure channel is closed when all operations are done
		}
	}()
}

func ActiveLoop(ctx context.Context, h host.Host, kdht *dht.IpfsDHT, contentID *cid.Cid, node *Node, comms chan string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if node.IsBootstrap {
				log.Println("Bootstrap node")
				go Provide(ctx, kdht, contentID, comms)
				go FindProvidersAndConnect(ctx, kdht, h, contentID, comms)
				select {
				case <-ctx.Done():
					// Context canceled, possibly due to shutdown
					return
				default:
					// Non-blocking write to channel
					comms <- fmt.Sprint("Node active:", h.ID().String())

				}
			} else {
				log.Println("Regular node")
				// Regular node
				for _, p := range *node.BootstrapPeers {
					// do not attempt if already connected
					if res := h.Network().Connectedness(peer.ID(p.ID)); res == network.Connected {
						comms <- "Already connected..."
						continue
					}
					go ConnectToPeer(ctx, h, p, comms, 0)

					select {
					case <-ctx.Done():
						// Context canceled, possibly due to shutdown
						return
					default:
						// Non-blocking write to channel
						comms <- "Connecting to bootstrap node..."

					}
				}

				// Periodically provide content
				go Provide(ctx, kdht, contentID, comms)

				// Periodically find providers and connect
				go FindProvidersAndConnect(ctx, kdht, h, contentID, comms)

				select {
				case <-ctx.Done():
					// Context canceled, possibly due to shutdown
					return
				default:
					// Non-blocking write to channel
					comms <- "Regular node active... " + fmt.Sprintf("%v", kdht.RoutingTable().ListPeers())
				}
			}
		case <-ctx.Done():
			comms <- "Shutting down..."
			return
		}
	}
}

func ConnectToPeer(ctx context.Context, h host.Host, p *p2p.Peer, comms chan string, idx int) {
	err := h.Connect(ctx, peer.AddrInfo{
		ID:    p.H.ID(),
		Addrs: p.H.Addrs(),
	})

	if err != nil {
		comms <- err.Error() + "\n"
		return
	}

	// After connecting successfully
	select {
	case <-ctx.Done():
		// Context canceled, possibly due to shutdown
		return
	default:
		// Non-blocking write to channel
		comms <- "Connected to peer %s " + p.H.ID().String()
	}
}

func Provide(ctx context.Context, kdht *dht.IpfsDHT, contentID *cid.Cid, comms chan string) {
	err := kdht.Provide(ctx, *contentID, true)
	if err != nil {
		comms <- err.Error()
	} else {
		comms <- fmt.Sprintf("Provided content %s", contentID.String())
	}
}

func FindProvidersAndConnect(ctx context.Context, kdht *dht.IpfsDHT, h host.Host, contentID *cid.Cid, comms chan string) {
	provider := kdht.FindProvidersAsync(ctx, *contentID, 0)

	// Periodically provide content
	for p := range provider {
		go Provide(ctx, kdht, contentID, comms)
		if p.ID == h.ID() {
			continue
		} else {
			comms <- fmt.Sprintf("Found provider: %s", p.ID.String())
			if res := h.Network().Connectedness(p.ID); res == network.Connected {
				comms <- "Already connected..."
				continue
			} else {
				comms <- "Connecting to provider..."
				go func(comms chan string) {
					h.Connect(ctx, peer.AddrInfo{
						ID:    p.ID,
						Addrs: p.Addrs,
					})
				}(comms)

				select {
				case <-ctx.Done():
					// Context canceled, possibly due to shutdown
					return
				default:
					// Non-blocking write to channel
					comms <- "Connected to provider..."
				}
			}
		}
	}
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
