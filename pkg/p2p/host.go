package p2p

import (
	"context"
	"fmt"
	"log"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/config"
	p2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	"github.com/libp2p/go-libp2p/p2p/security/tls"

	"google.golang.org/grpc"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	GRPCProtocolID = "/grpc"
)

type Cfg struct {
	// ListenAddr is the multiaddress to listen on
	ListenAddr multiaddr.Multiaddr

	// GRPCAddr is the address to connect to
	GRPCAddr string

	// GRPCDialOptions are the grpc dial options
	GRPCDialOptions []grpc.DialOption

	// LibP2PHostOptions are the libp2p host options
	LibP2PHostOptions *config.Option

	//StreamHandler is the libp2p stream handler
	StreamHandler network.StreamHandler

	// GRPCDialer is the grpc dialer
	GRPCDialer *grpc.ClientConn

	// GRPCServer is the grpc server
	GRPCServer *grpc.Server
}

// IHost is a common interface for a libp2p host and a grpc host
type IHost interface {
	LibP2PHost
	GRPCHost
	ID() string
	Connect(ctx context.Context, cfg *Cfg, opts config.Option) error
	Terminate() error
}

type TLS struct {
}

// Host is a libp2p host
type LibP2PHost struct {
	host.Host
}

func (lp *LibP2PHost) Addrs() []multiaddr.Multiaddr {
	return lp.Host.Addrs()
}

func (h *LibP2PHost) ID() peer.ID {
	return h.Host.ID()
}

// Connect connects to a peer
func (h *LibP2PHost) Connect(ctx context.Context, addrinfo peer.AddrInfo) error {
	err := h.Host.Connect(ctx, addrinfo)
	if err != nil {
		return err
	}
	return nil
}

// Terminate closes the host
func (h *LibP2PHost) Terminate() error {
	return h.Close()
}

func EncapsulateAddrStrWithProtocol(str string, protocol string, value string, buf *[]byte) {
	encapsulated := fmt.Sprintf("%s/%s/%s", str, protocol, value)
	if buf == nil {
		return
	}
	*buf = []byte(encapsulated)
}

func CreateMultiAddrString(ip string) string {
	return fmt.Sprintf("/ip4/%s", ip)
}

// NewHost creates a new libp p2p host
func NewLibP2PHost(p *Peer, port int) (host.Host, error) {
	buf := new([]byte)
	baseStr := CreateMultiAddrString("192.168.1.107")
	EncapsulateAddrStrWithProtocol(baseStr, "tcp", fmt.Sprint(port), buf)
	// Create a new host with the specified options
	privKey, err := p2pcrypto.UnmarshalEd25519PrivateKey(p.PrivateKeyBuffer)
	if err != nil {
		log.Println("Error unmarshalling private key")
		return nil, err
	}

	opts := []config.Option{
		libp2p.ListenAddrStrings(string(*buf)),
		libp2p.EnableNATService(),
		libp2p.Identity(privKey),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
	}
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}
	// fmt.Println(h.Network().ListenAddresses())
	return h, nil
}

// GRPCHost is a grpc host
type GRPCHost struct {
	dialer *grpc.ClientConn
}

// ID returns the host's peer ID
func (h *GRPCHost) ID(opts *Cfg) string {
	return opts.GRPCAddr
}

// Connect connects to a peer
func (h *GRPCHost) Connect(ctx context.Context, cfg *Cfg) {
	grpc.Dial(cfg.GRPCAddr, cfg.GRPCDialOptions...)
}

// Close closes the host
func (h *GRPCHost) Terminate() error {
	return h.dialer.Close()
}

func NewGRPCHost(dialer *grpc.ClientConn) *GRPCHost {
	return &GRPCHost{dialer: dialer}
}
