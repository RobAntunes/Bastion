package block

import (
	"math/big"
	"time"

	pb "bastion.network/bastion/proto/gen/chain"

	sha3 "golang.org/x/crypto/sha3"
)

// Leave out the fields that must be zeroed out.

func NewGenesis() *pb.Block {
	height := 0

	zeroTx := &pb.Transaction{
		BaseTransaction: &pb.BaseTransaction{
			Value: big.NewInt(0).Uint64(),
		},
		// Signature: "0x00000000000000000000000000000000",
	}
	zeroTx.Id = HashTransaction(zeroTx)

	zeroTxHash := sha3.Sum512(TxToBytes(zeroTx.BaseTransaction))
	hash := sha3.Sum512(zeroTxHash[:])

	merkleRoot := hash

	header := &pb.Header{
		Height: uint64(height),
		// Nonce: "0x00",
		Hash:         TxToBytes(zeroTx.BaseTransaction),
		PreviousHash: TxToBytes(zeroTx.BaseTransaction),
		Timestamp:    uint64(time.Now().Unix()),
		MerkleRoot:   merkleRoot[:],
	}

	body := &pb.Body{
		Transactions: []*pb.Transaction{zeroTx},
	}

	return &pb.Block{
		Header: header,
		Body:   body,
	}
}
