package block

import (
	sha3 "golang.org/x/crypto/sha3"

	"bastion.network/bastion/pkg/system/utils"
	pb "bastion.network/bastion/proto/gen/chain"
)

type MerkleTree struct {
	Root *MerkleNode

}

type MerkleNode struct {
	Left *MerkleNode
	Right *MerkleNode
	Hash []byte
}

func NewMerkleTree(transactions *[]*pb.Transaction) *MerkleTree {
	merkleTree := &MerkleTree{}
	merkleTree.Root = BuildMerkleTree(transactions)
	return merkleTree
}

func BuildMerkleTree(transactions *[]*pb.Transaction) *MerkleNode {
	var nodes []*MerkleNode
	if len(*transactions) % 2 != 0 {
		*transactions = append(*transactions, (*transactions)[len(*transactions)-1])
	}
	for _, t := range *transactions {
		nodes = append(nodes, &MerkleNode{nil, nil, t.Id})
	}
	for len(nodes) > 1 {
		var newLevel []*MerkleNode
		for i := 0; i < len(nodes); i += 2 {
			if i+1 == len(nodes) {
				newLevel = append(newLevel, &MerkleNode{
					nodes[i], nil, HashPair(nodes[i], nodes[i]),
				})

			} else {
				newLevel = append(newLevel, &MerkleNode{nodes[i], nodes[i+1], HashPair(nodes[i], nodes[i+1])})
			}
		}
		nodes = newLevel
	}
	return nodes[0]
}

func HashPair(n1 *MerkleNode, n2 *MerkleNode) []byte {
	buf := append(n1.Hash[:], n2.Hash[:]...)
	hasher := sha3.New512()
	hasher.Write(buf)
	return hasher.Sum(nil)
}

func (m *MerkleTree) GetRoot() string {
	return utils.EncodeHash16(m.Root.Hash)
}

func (m *MerkleTree) GetProof(t *pb.Transaction) []string {
	return m.GetProofRecursive(t, m.Root)
}

// GetProofRecursive returns the proof of a transaction in the merkle tree
// by recursively traversing the tree
// If the transaction is found, the function returns the proof
// If the transaction is not found, the function returns an empty array
// The proof is an array of hashes, where each hash is the hash of the concatenation
// of the two hashes of the children of the node
// The proof is used to verify the authenticity of a transaction
// in the merkle tree
func (m *MerkleTree) GetProofRecursive(t *pb.Transaction, n *MerkleNode) []string {
	if n.Left == nil && n.Right == nil {
		if utils.EncodeHash16(t.Id) == utils.EncodeHash16(n.Hash) {
			return []string{utils.EncodeHash16(n.Hash)}
		} else {
			return []string{}
		}
	}
	if n.Left != nil && utils.EncodeHash16(n.Left.Hash) == utils.EncodeHash16(t.Id) {
		return append(m.GetProofRecursive(t, n.Right), utils.EncodeHash16(n.Hash))
	}
	if n.Right != nil && utils.EncodeHash16(n.Right.Hash) == utils.EncodeHash16(t.Id) {
		return append(m.GetProofRecursive(t, n.Left), utils.EncodeHash16(n.Hash))
	}
	return []string{}
}
