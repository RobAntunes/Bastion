package crypto

import "crypto/rand"

// GenerateNonce generates a random 32 byte nonce
// and returns it as a byte array.
// It returns an empty byte array and an error if the random number generator fails
func GenerateNonce() ([32]byte, error) {
	var nonce [32]byte
	_, err := rand.Read(nonce[:])
	if err != nil {
		return [32]byte{}, err
	}
	return nonce, err
}
