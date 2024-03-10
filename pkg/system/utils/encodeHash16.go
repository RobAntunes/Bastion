package utils

import "fmt"

func EncodeHash16(hash []byte) string {
	return fmt.Sprintf("%x", hash)
}
