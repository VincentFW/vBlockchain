package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
	"strconv"
)

var (
	maxNonce = math.MaxInt64
)

const targetBits = 15 // target diffculty

type ProofOfWork struct {
	block  *Block
	target *big.Int
}

func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	pow := &ProofOfWork{b, target}

	return pow
}

func (pow *ProofOfWork) prepareData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.block.PrevBlockHash,
			pow.block.hashTransactions(),
			IntToHex(pow.block.Timestamp),
			IntToHex(int64(targetBits)),
			IntToHex(int64(nonce))},
		[]byte{},
	)
	return data
}

func (pow *ProofOfWork) run() (int, []byte) {
	var hashValue big.Int
	var hash [32]byte
	nonce := 0

	fmt.Printf("Mining the block\n")
	for nonce < maxNonce {
		data := pow.prepareData(nonce)

		hash = sha256.Sum256(data)
		fmt.Printf("\r%x", hash)
		hashValue.SetBytes(hash[:])

		if hashValue.Cmp(pow.target) == -1 {
			break
		} else {
			nonce++
		}
	}
	fmt.Printf("Finish mining. Get block and  the nonce is: %s\n", strconv.FormatInt(int64(nonce), 16))
	return nonce, hash[:]
}

func (pow *ProofOfWork) validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.target) == -1

	return isValid
}
