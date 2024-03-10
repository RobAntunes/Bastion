package chain

import (
	"bastion.network/bastion/pkg/block"
	"bastion.network/bastion/pkg/system/utils"
	
	pb "bastion.network/bastion/proto/gen/chain"
)

type Chain struct {
	Blocks *[]*block.Block
}

func NewChain() *Chain {
	genesis := block.NewGenesis()

	blocks := make([]*block.Block, 0)
	blocks = append(blocks, &block.Block{
		Header: genesis.Header,
		Body: genesis.Body,
	})
	return &Chain{Blocks: &blocks}
}

// TODO: Add recovering from panic
// TODO: Add error handling
// TODO: Generate nonce for block and transactions from randomly selected external source

func (c *Chain) GetBlockByHeight(height int) *block.Block {
	return (*c.Blocks)[height]
}

func (c *Chain) GetBlockByHash(hash string) *block.Block {
	for _, b := range *c.Blocks {
		if utils.EncodeHash16(b.Header.Hash) == hash {
			return b
		}
	}
	return nil
}

func (c *Chain) GetBlockByMerkleRoot(merkleRoot string) *block.Block {
	for _, b := range *c.Blocks {
		if utils.EncodeHash16(b.Header.MerkleRoot) == merkleRoot {
			return b
		}
	}
	return nil
}

func (c *Chain) GetBlockByTimestamp(timestamp int64) *block.Block {
	for _, b := range *c.Blocks {
		if b.Header.Timestamp == uint64(timestamp) {
			return b
		}
	}
	return nil
}

// This is trash
func (c *Chain) GetTransactionById(id string) *pb.Transaction {
	for _, b := range *c.Blocks {
		for _, t := range b.Body.Transactions {
			if utils.EncodeHash16(t.Id) == id {
				return t
			}
		}
	}
	return nil
}
