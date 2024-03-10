package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"

	"log"

	"bastion.network/bastion/pkg/block"
	pb "bastion.network/bastion/proto/gen/chain"

	kyberk2so "github.com/symbolicsoft/kyber-k2so"
	"golang.org/x/crypto/chacha20poly1305"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type BastionServiceClient struct {
	Client pb.BastionServiceClient
}

func (bsc *BastionServiceClient) GetBlockchain(ctx context.Context, req *pb.GetBlockchainRequest) *pb.GetBlockchainResponse {
	res, err := bsc.Client.GetBlockchain(ctx, &pb.GetBlockchainRequest{})
	if err != nil {
		log.Printf("Could not get blockchain: %v", err)
	}
	return res
}

func (bsc *BastionServiceClient) GetBlockchainHeight(ctx context.Context, req *pb.GetBlockchainHeightRequest) *pb.GetBlockchainHeightResponse {
	res, err := bsc.Client.GetBlockchainHeight(ctx, &pb.GetBlockchainHeightRequest{})
	if err != nil {
		log.Printf("Could not get blockchain height: %v", err)
	}
	return res
}

func (bsc *BastionServiceClient) AddBlock(txs []*pb.Transaction) (*pb.AddBlockResponse, error) {
	res, err := bsc.Client.AddBlock(context.Background(), &pb.AddBlockRequest{
		Transactions: txs,
	})
	if err != nil {
		log.Printf("Could not add block: %v", err)
	}
	return res, nil
}

var tlsOpts = &tls.Config{
	InsecureSkipVerify: true,
	MinVersion:         tls.VersionTLS12,
	SessionTicketKey:   [32]uint8{0x00},
	// CipherSuites:       []uint16{tls.CH},
}

func main() {
	ctx := context.Background()

	addr := "127.0.0.1:50051"

	_, publicKey, _ := kyberk2so.KemKeypair1024()
	_, ssA, err := kyberk2so.KemEncrypt1024(publicKey)
	if err != nil {
		log.Fatalf("Failed to encrypt: %v", err)
	}

	// Encrypt the message with the shared secret
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		log.Fatalf("Failed to read random: %v", err)
	}
	aead, err := chacha20poly1305.NewX(ssA[:])
	if err != nil {
		log.Fatalf("Failed to create AEAD: %v", err)
	}

	client, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsOpts)))
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}

	bsc := &BastionServiceClient{
		Client: pb.NewBastionServiceClient(client),
	}

	tx := block.CreateTransaction()
	tx.Id = block.HashTransaction(tx)

	blk, err := bsc.AddBlock([]*pb.Transaction{tx})
	if err != nil {
		log.Printf("Could not add block: %v", err)
	}

	buf := make([]byte, 64)
	sb := aead.Seal(buf, nonce, block.BlockToBytes(blk.Block), nil)

	log.Printf("Hash: %v", string(sb))
}
