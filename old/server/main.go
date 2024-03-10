package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"net"

	// "fmt"

	"log"

	"bastion.network/bastion/block"
	pb "bastion.network/bastion/proto/gen/chain"
	"github.com/libp2p/go-libp2p/core/host"

	"google.golang.org/grpc"
)

const (
	EntropyServiceName = "bastion.network.entropy"
	EntropySourcesMin  = 2
	EntropySourcesMax  = 8
)

// server is used to implement blockchain.BlockchainServiceServer.
type BastionServiceServer struct {
	pb.UnimplementedBastionServiceServer // must embed the unimplemented server
}

func (s *BastionServiceServer) GetBlockchainHeight(ctx context.Context, req *pb.GetBlockchainHeightRequest) (*pb.GetBlockchainHeightResponse, error) {
	return &pb.GetBlockchainHeightResponse{
		Height: uint64(len(Blockchain.Blocks)),
	}, nil
}

type BlockServiceServer struct {
	pb.UnimplementedBlockServiceServer // must embed the unimplemented server
}

type TransactionServiceServer struct {
	pb.UnimplementedTransactionServiceServer // must embed the unimplemented server
}

type EntropyServiceServer struct {
	pb.UnimplementedEntropyServiceServer // must embed the unimplemented server
}

func (s *BastionServiceServer) GetEntropy(ctx context.Context, entropyReq *pb.GetEntropyRequest) (*pb.GetEntropyResponse, error) {
	// get cryptographically secure entropy

	entropy := make([]byte, entropyReq.Length)
	_, err := rand.Read(entropy)
	if err != nil {
		return nil, err
	}
	return &pb.GetEntropyResponse{
		Entropy: entropy,
	}, nil
}

func (s *BastionServiceServer) AddBlock(
	ctx context.Context,
	blockReq *pb.AddBlockRequest,
) (*pb.AddBlockResponse, error) {
	// TODO: get entropy for nonce, select a number of nodes between 3 and 10 and get their entropy

	newBlock, err := block.CreateBlock(context.Background(), block.RequiredData{
		Height:   uint64(len(Blockchain.Blocks)),
		PrevHash: []byte(Blockchain.Blocks[len(Blockchain.Blocks)-1].Header.Hash),
	}, &blockReq.Transactions)
	if err != nil {
		return nil, err
	}
	Blockchain.Blocks = append(Blockchain.Blocks, newBlock)

	return &pb.AddBlockResponse{
		Block:   newBlock,
		Success: true,
	}, nil
}

func HashBlock(ctx context.Context, req *pb.HashBlockRequest) (*pb.HashBlockResponse, error) {
	// server side implementation does not require grpc.Dial
	// instead, we can call the service directly

	// hash a block
	// return the hash
	// return an error if the block could not be hashed
	return &pb.HashBlockResponse{}, nil
}

func CreateGenesis() *pb.Block {
	g := block.NewGenesis()
	return &pb.Block{
		Header: &pb.Header{
			PreviousHash: g.Header.PreviousHash,
			Height:       g.Header.Height,
			Nonce:        g.Header.Nonce,
			Hash:         g.Header.Hash,
			Timestamp:    g.Header.Timestamp,
			MerkleRoot:   g.Header.MerkleRoot,
		},
		Body: &pb.Body{
			Transactions: g.Body.Transactions,
		},
	}
}

var blockchain *pb.Blockchain = &pb.Blockchain{Blocks: []*pb.Block{block.NewGenesis()}}

func (s *BastionServiceServer) GetBlockchain(ctx context.Context, req *pb.GetBlockchainRequest) (*pb.GetBlockchainResponse, error) {
	return &pb.GetBlockchainResponse{
		Blockchain: blockchain,
		Success:    true,
	}, nil
}

func AddTransaction(tx *pb.Transaction, b *pb.Block) (*pb.AddTransactionResponse, error) {
	// server side implementation does not require grpc.Dial
	// instead, we can call the service directly

	// add a transaction to a block
	// return the block
	// return an error if the transaction could not be added
	return &pb.AddTransactionResponse{}, nil
}

// func GetEntropy(ctx context.Context, req *pb.GetEntropyRequest) (*pb.GetEntropyResponse, error) {
// 	nodes := req.Nodes
// 	length := req.Length

// }

var Blockchain = &pb.Blockchain{
	Blocks: []*pb.Block{CreateGenesis()},
}

type EntropyService struct {
	Host host.Host
}

//func NewEntropyService(h host.Host) *EntropyService {
//	ps := &EntropyService{h}
//	h.SetStreamHandler(EntropyServiceName, ps.NonceStreamHandler)
//	return ps
//}

// func GetRandomPeers(node host.Host) []*peer.ID {
// 	rPeers := []*peer.ID{}
// 	peers := node.Peerstore().Peers()
// 	peerNumber, err := rand.Int(rand.Reader, big.NewInt(EntropySourcesMax))
// 	if big.NewInt(EntropySourcesMin).Cmp(peerNumber) == -1 {
// 		peerNumber = big.NewInt(EntropySourcesMin)
// 	}
// 	if err != nil {
// 		log.Printf("error generating random number: %s", err)
// 	}
// 	for i := 0; i < int(peerNumber.Int64()); i++ {
// 		rPeer, err := rand.Int(rand.Reader, big.NewInt(int64(len(peers))))
// 		if err != nil {
// 			log.Printf("error generating random number: %s", err)
// 		}
// 		rPeers = append(rPeers, &peers[rPeer.Uint64()])
// 	}
// 	return rPeers
// }

//func (s *EntropyService) NonceStreamHandler(stream network.Stream) {
//	host := s.Host
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
//	defer cancel()
//	// set the service name
//	if err := stream.Scope().SetService(EntropyServiceName); err != nil {
//		log.Printf("error attaching stream to ping service: %s", err)
//		stream.Reset()
//	}
//
//	entropySize, err := rand.Int(rand.Reader, big.NewInt(256/EntropySourcesMax/EntropySourcesMin))
//	if err != nil {
//		log.Printf("error generating random number: %s", err)
//		stream.Reset()
//	}
//
//	if err := stream.Scope().ReserveMemory(int(entropySize.Int64()), network.ReservationPriorityAlways); err != nil {
//		log.Printf("error reserving memory for entropy stream: %s", err)
//		stream.Reset()
//	}
//	defer stream.Scope().ReleaseMemory(int(entropySize.Int64()))
//
//	// get entropy from random peers
//	addr, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/0")
//	if err != nil {
//		panic(err)
//	}
//	peer, err := peerstore.AddrInfoFromP2pAddr(addr)
//	if err != nil {
//		panic(err)
//	}
//
//	rPeers := GetRandomPeers(s.Host)
//	for _, p := range rPeers {
//		go func() {
//			if err := host.Connect(ctx, *peer); err != nil {
//				log.Printf("error connecting to peer: %s", err)
//				err := stream.Reset()
//				if err != nil {
//					return
//				}
//			} else {
//				log.Printf("connected to peer: %s", peer)
//				defer func(host host.Host) {
//					err := host.Close()
//					if err != nil {
//
//					}
//				}(host)
//
//				stream.Write([]byte(p.ShortString()))
//
//			}
//		}()
//	}
//}

func main() {
	cert, err := tls.LoadX509KeyPair("ec_certificate.crt", "ec_private.key")
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	s := grpc.NewServer()
	pb.RegisterBastionServiceServer(s, &BastionServiceServer{})
	pb.RegisterBlockServiceServer(s, &BlockServiceServer{})
	pb.RegisterTransactionServiceServer(s, &TransactionServiceServer{})
	pb.RegisterEntropyServiceServer(s, &EntropyServiceServer{})

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	l := tls.NewListener(listener, tlsConfig)

	log.Printf("Server listening on %s", l.Addr().String())
	s.Serve(l)
	defer l.Close()
	defer s.Stop()
}
