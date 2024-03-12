package p2p

import (
	"context"
	"crypto/ed25519"
	"encoding/binary"
	"sync"

	"log"
	mrand "math/rand"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multihash"
)

// PSRNG is a non-cryptographically secure pseudo-random number generator that implements the io.Reader interface.
type PSRNG struct {
	value  int64
	seed   int64
	source mrand.Source
	mu     sync.Mutex
}

// NewPSRNG creates a new PSRNG instance with a given seed.
func NewPSRNG(seed int64) *PSRNG {
	s := mrand.NewSource(seed)
	return &PSRNG{
		value:  mrand.New(s).Int63(),
		seed:   seed,
		source: s,
	}
}

// Read fills the provided byte slice with pseudo-random data.
func (r *PSRNG) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Generate random numbers using the seeded source
	for i := range p {
		// Use Int63() of source and take the lower byte
		rnd := r.value
		p[i] = byte(rnd & 0xff)
	}
	return len(p), nil
}

func (r *PSRNG) Update(seed int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.seed = seed
	r.source.Seed(seed)
	r.value = mrand.New(r.source).Int63()
}

// Peer represents a single peer in the network.
type Peer struct {
	ID               string
	PrivateKeyBuffer []byte
	H			  host.Host
	DHT 		  *dht.IpfsDHT
}

// GenPeerID generates a new peer ID and private key buffer.
func GetPeers(qty int, seed int64) ([]*Peer, error) {
	psrng := NewPSRNG(seed)
	peers := make([]*Peer, 0)
	for range qty {
		peerID, buf, err := GenPeerID(psrng)
		if err != nil {
			log.Printf("Error generating peer ID: %v", err)
		} else {
			peers = append(peers, &Peer{ID: peerID, PrivateKeyBuffer: buf})
		}
	}
	return peers, nil
}

// NewKDHT creates a new Kademlia DHT instance with the provided host.
func NewKDHT(ctx context.Context, h host.Host) (*dht.IpfsDHT, error) {
	// Initialize a new DHT instance with the host, setting it to automatic mode.
	// This allows the DHT to operate in client or server mode based on the network environment.
	kdht, err := dht.New(ctx, h, dht.Mode(dht.ModeAutoServer))
	if err != nil {
		return nil, err
	}

	// If bootstrap peers are provided, use them to bootstrap the DHT.
	err = kdht.Bootstrap(ctx)
	if err != nil {
		return nil, err
	}
	return kdht, nil
}

// TODO: change to EDDSA with Curve448
func GenPeerID(psrng *PSRNG) (string, []byte, error) {
	psrng.Update(psrng.seed + 1)
	buf := make([]byte, 64)
	binary.NativeEndian.PutUint64(buf, uint64(psrng.value))
	pbK, prvK, err := ed25519.GenerateKey(psrng)
	if err != nil {
		log.Printf("Error generating key pair: %v", err)
		return "", nil, err
	}
	publicKeyBytes := []byte(ed25519.PublicKey(pbK))
	privateKeyBytes := []byte(ed25519.PrivateKey(prvK))

	buf = make([]byte, 64)
	copy(buf, privateKeyBytes)
	copy(buf[32:], publicKeyBytes)

	hash, err := multihash.Sum(publicKeyBytes, multihash.IDENTITY, -1)
	if err != nil {
		log.Printf("Error hashing peer ID: %v", err)
		return "", nil, err
	}
	encoded := hash.B58String()

	return string(encoded), buf, nil
}
