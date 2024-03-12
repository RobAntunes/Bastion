package main

import (
	"crypto/rand"

	"github.com/cloudflare/circl/sign/eddilithium3"
	kyberk2so "github.com/symbolicsoft/kyber-k2so"
	blake3 "github.com/zeebo/blake3"
	sha3 "golang.org/x/crypto/sha3"
)

type HashFunction interface {
	Hash256(data []byte) [32]byte
	Hash512(data []byte) [64]byte
	Reset()
}

type SHA3 struct {
	hasher sha3.ShakeHash
}

func (s *SHA3) Hash256(data []byte) [32]byte {
	return sha3.Sum256(data)
}

func (s *SHA3) Hash512(data []byte) [64]byte {
	return sha3.Sum512(data)
}

func (s *SHA3) Hash(data []byte) []byte {
	s.hasher.Write(data)
	return s.hasher.Sum(nil)
}

func (s *SHA3) Reset() {
	s.hasher.Reset()
}

func ToBytes256(data [32]byte) []byte {
	return data[:]
}

func ToBytes512(data [64]byte) []byte {
	return data[:]
}

func (s3 *SHA3) Wrap256(data []byte, count uint8, hashFunc HashFunction) ([32]byte, HashFunction) {
	hashFunc.Reset()
	var buf [32]byte
	for range count {
		buf = hashFunc.Hash256(ToBytes256(s3.Hash256(data)))
	}
	return buf, hashFunc
}

func (s3 *SHA3) Wrap512(data []byte, count uint8, hashFunc HashFunction) ([64]byte, HashFunction) {
	hashFunc.Reset()
	var buf [64]byte
	for range count {
		buf = hashFunc.Hash512(ToBytes512(s3.Hash512(data)))
	}
	return buf, hashFunc
}

type Blake3 struct {
	hasher blake3.Hasher
}

func (b *Blake3) Hash256(data []byte) [32]byte {
	return blake3.Sum256(data)
}

func (b *Blake3) Hash512(data []byte) [64]byte {
	return blake3.Sum512(data)
}

func (b *Blake3) Reset() {
	b.hasher.Reset()
}

func (b3 *Blake3) Wrap(data []byte, count uint8, hashFunc HashFunction) ([64]byte, HashFunction) {
	hashFunc.Reset()
	buf := [64]byte{}
	for range count {
		buf = hashFunc.Hash512(ToBytes512(b3.Hash512(data)))
	}
	return buf, hashFunc
}

type K2SO struct {
	publicKey  [1568]byte
	privateKey [3168]byte
}

func NewSHA3() *SHA3 {
	return &SHA3{sha3.NewShake256()}
}

func NewBlake3() *Blake3 {
	return &Blake3{*blake3.New()}
}

func NewK2SO() (*K2SO, error) {
	prvk, pk, err := kyberk2so.KemKeypair1024()
	if err != nil {
		return nil, err
	}
	return &K2SO{
		privateKey: prvk,
		publicKey:  pk,
	}, nil
}

// HashDynamic returns a hash of the data using either SHA3 or Blake3
// depending on the size of the data and the highSecurity flag.
// If highSecurity is true, the data is hashed using SHA3-256 and then Blake3.
// If highSecurity is false and data is less than 4KiB, the data is hashed using SHA3 only.
// If data is 4KiB or more, the data is hashed using Blake3 only.
// TODO: Implement a more secure method for hashing data
// such as using a keypair to sign the data before hashing.
// This will prevent data tampering and ensure data integrity.
// This can be done by generating a keypair
// and then using the public key to verify the data.
// The private key can be used to sign the data before hashing.
// The signature can then be verified using the public key.
// This will ensure that the data has not been tampered with.
// The keypair should be generated using a secure method
// and stored securely.
func HashDynamic(data []byte, highSecurity bool) [64]byte {
	if highSecurity {
		data, _ := NewBlake3().Wrap(data, 1, NewSHA3())
		return data
	} else if len(data) >= 4096 { // 4 KiB threshold for Blake3
		return NewBlake3().Hash512(data)
	} else {
		return NewSHA3().Hash512(data)
	}
}

// SignData signs the data using the eddilithium3 signature scheme
// and returns the signature.
func SignData(data []byte) (*eddilithium3.PublicKey, []byte, error) {
	pk, prk, err := eddilithium3.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	buf := make([]byte, eddilithium3.SignatureSize)
	eddilithium3.SignTo(prk, buf, data)

	return pk, buf, nil
}
