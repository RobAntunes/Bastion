package block

import (
	"fmt"

	pb "bastion.network/bastion/proto/gen/chain"
	"bastion.network/bastion/pkg/system/utils"
)

func Validate(old *pb.Block, new *Block) bool {
	oldHash := fmt.Sprintf("%x", old.Header.Hash)
	newHash := fmt.Sprintf("%x", new.Header.PreviousHash)
	if old.Header.Height == 0 {
		return true
	}
	if old.Header.Height+1 != new.Header.Height {
		return false
	}
	if oldHash != newHash {
		return false
	}
	if utils.EncodeHash16(old.Header.Hash) != utils.EncodeHash16([]byte(new.Header.PreviousHash)) {
		return false
	}
	if old.Header.Timestamp > new.Header.Timestamp {
		return false
	}
	return true
}
