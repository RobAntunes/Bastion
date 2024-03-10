package block

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pb "bastion.network/bastion/proto/gen/chain"
	"bastion.network/bastion/pkg/system/utils"

	"golang.org/x/crypto/sha3"
)

type Header struct {
	PreviousHash string
	Height       int
	Nonce        int
	Hash         string
	Timestamp    int64
	MerkleRoot   string
}

type Body struct {
	Transactions *[]*pb.Transaction
}

type Block struct {
	Header *pb.Header
	Body   *pb.Body
}

type EncodedBlock struct {
	Header *EncodedHeader
	Body   *EncodedBody
}

type EncodedHeader struct {
	PreviousHash string
	Height       int
	Nonce        int
	Hash         string
	Timestamp    int64
	MerkleRoot   string
}

type EncodedBody struct {
	Transactions *[]*EncodedTransaction
}

type RequiredData struct {
	Entropy  []byte
	Height   uint64
	PrevHash []byte
}

func CreateBlock(ctx context.Context, requiredData RequiredData, transactions *[]*pb.Transaction) (*pb.Block, error) {
	// TODO: Use entropy to create nonce

	newBlock := &pb.Block{
		Header: &pb.Header{
			PreviousHash: requiredData.PrevHash,
			Height:       requiredData.Height + 1,
			// Nonce: "",
			Timestamp:  uint64(time.Now().Unix()),
			MerkleRoot: []byte(NewMerkleTree(transactions).GetRoot()),
		},
		Body: &pb.Body{
			Transactions: *transactions,
		},
	}
	newBlock.Header.Hash = Hash(newBlock)
	newBlock.Header.Hash = []byte(utils.EncodeHash16(Hash(newBlock)))
	return newBlock, nil
}

func BlockToBytes(b *pb.Block) []byte {
	encoded := make([]byte, 0)
	encoded = append(encoded, b.Header.PreviousHash...)
	encoded = append(encoded, b.Header.Hash...)
	encoded = append(encoded, []byte(fmt.Sprintf("%d", b.Header.Height))...)
	encoded = append(encoded, []byte(fmt.Sprintf("%d", b.Header.Nonce))...)
	encoded = append(encoded, []byte(b.Header.MerkleRoot)...)
	encoded = append(encoded, []byte(fmt.Sprintf("%d", b.Header.Timestamp))...)
	return encoded
}

func Hash(b *pb.Block) []byte {
	hasher1 := sha3.New512()
	hasher1.Write(b.Header.PreviousHash)
	hasher1.Write([]byte(fmt.Sprintf("%d", b.Header.Height)))
	hasher1.Write([]byte(fmt.Sprintf("%d", b.Header.Nonce)))
	hasher1.Write([]byte(b.Header.MerkleRoot))
	hasher1.Write([]byte(fmt.Sprintf("%d", b.Header.Timestamp)))
	hasher2 := sha3.New512()
	hasher2.Write([]byte(fmt.Sprintf("%x", hasher1.Sum([]byte{}))))
	return hasher2.Sum([]byte{})
}

func (EB *EncodedBlock) Hash() [64]byte {
	hasher := sha3.New512()
	hasher.Write([]byte(EB.Header.PreviousHash))
	hasher.Write([]byte(fmt.Sprintf("%d", EB.Header.Height)))
	hasher.Write([]byte(fmt.Sprintf("%d", EB.Header.Nonce)))
	hasher.Write([]byte(EB.Header.MerkleRoot))
	hasher.Write([]byte(fmt.Sprintf("%d", EB.Header.Timestamp)))
	return sha3.Sum512(hasher.Sum([]byte{}))
}

func (H *Header) Marshal() (string, error) {
	res, err := json.Marshal(H)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func (B *Body) Marshal() ([][]byte, error) {
	encoded := make([][]byte, len(*B.Transactions))
	for _, t := range *B.Transactions {
		etx := TxEncode16(t)
		j, err := json.Marshal(etx)
		if err != nil {
			return [][]byte{}, err
		}
		encoded = append(encoded, j)
	}
	return encoded, nil
}

func Unmarshal(s string) (*EncodedBlock, error) {
	block := &EncodedBlock{}
	err := json.Unmarshal([]byte(s), block)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func Marshal(B *EncodedBlock) (string, error) {
	res, err := json.Marshal(B)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func Encode16(b []byte) string {
	return fmt.Sprintf("%x", b)
}
