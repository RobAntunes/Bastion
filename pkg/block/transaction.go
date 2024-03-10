package block

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"time"
	"unicode/utf8"

	pb "bastion.network/bastion/proto/gen/chain"

	sha3 "golang.org/x/crypto/sha3"
)

type BaseTransaction struct {
	From      [64]byte
	To        [64]byte
	Value     *big.Int
	Nonce     [64]byte
	Timestamp uint64
}

func TxToBytes(T *pb.BaseTransaction) []byte {
	buf := make([]byte, 192)
	buf = append(buf, T.From...)
	buf = append(buf, T.To...)
	buf = append(buf, T.Nonce...)

	contentBuf := make([]byte, 256)
	// return new slice with content + T.Value.ToBytes()
	binary.NativeEndian.PutUint64(contentBuf, T.Value)
	return append(buf, contentBuf...)
}

type Transaction struct {
	Base *BaseTransaction
	Id   [64]byte // sha3 hash
	// Signature [32]byte
	Encoded *EncodedTransaction
}

type EncodedTransaction struct {
	Id    string `json:"id"`
	From  string `json:"from"`
	To    string `json:"to"`
	Value string `json:"value"`
	Nonce string `json:"nonce"`
	// Signature string
}

func CreateTransaction() *pb.Transaction {
	var toBuf []byte
	var fromBuf []byte

	_, err := rand.Read(toBuf[:])
	if err != nil {
		panic(err)
	}

	_, err = rand.Read(fromBuf[:])
	if err != nil {
		panic(err)
	}

	// TODO: Value can be negative right now which is wrong
	// TODO check if 20 decimal places is enough
	tx := &pb.Transaction{
		Id: []byte{},
		BaseTransaction: &pb.BaseTransaction{
			From:  []byte(fmt.Sprintf("%v", sha3.Sum512(fromBuf))),
			To:    []byte(fmt.Sprintf("%v", sha3.Sum512(toBuf))),
			Value: big.NewInt(int64(rand.Int63())).Uint64(),
			//Nonce: [32]byte,
			Timestamp: uint64(time.Now().Unix()),
		}, // Signature: "0x"
	}
	return tx
}

func (T *Transaction) Marshal() string {
	res, err := json.Marshal(T.Encoded)
	if err != nil {
		return ""
	}
	return string(res)
}

// Hash the transaction without the id
func HashTransaction(tx *pb.Transaction) []byte {
	hasher := sha3.New512()
	hasher.Write(TxToBytes(tx.BaseTransaction))
	return hasher.Sum(nil)
}

// Solidify the transaction by encoding the fields in hex
func TxEncode16(tx *pb.Transaction) *EncodedTransaction {
	return &EncodedTransaction{
		Id:    fmt.Sprintf("%x", tx.Id),
		From:  fmt.Sprintf("%x", tx.BaseTransaction.From),
		To:    fmt.Sprintf("%x", tx.BaseTransaction.To),
		Value: fmt.Sprint(tx.BaseTransaction.Value),
		Nonce: fmt.Sprintf("%x", tx.BaseTransaction.Nonce),
		// Signature: fmt.Sprintf("%x", tx.Signature),
	}
}

// TODO: Add signature above and below
func MarshalText(tx *pb.Transaction) ([]byte, error) {
	str := string(tx.Id)
	for _, i := range [][]byte{tx.BaseTransaction.From[:], tx.BaseTransaction.To[:], tx.BaseTransaction.Nonce[:], []byte(fmt.Sprintf("%d", tx.BaseTransaction.Value))} {
		dr, _ := utf8.DecodeRune(i)
		str += string(dr)
	}
	return []byte(str), nil
}
