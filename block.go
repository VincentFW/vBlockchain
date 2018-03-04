package main

import (
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

type Block struct {
	Timestamp     int64
	Transactions  []*Transaction
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
	Height	 	  int
}

func NewBlock(transactions []*Transaction, prevBlockHash []byte, height int) *Block {
	block := &Block{time.Now().Unix(), transactions, prevBlockHash, []byte{}, 0,height}
	pow := NewProofOfWork(block)
	nonce, hash := pow.run()
	block.Hash = hash[:]
	block.Nonce = nonce

	return block
}

func NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{},0)
}

func (block *Block) Serialize() []byte {
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)
	err := encoder.Encode(&block)
	if err != nil {
		log.Panic(err)
	}
	return result.Bytes()
}

func DeserializeBlock(b []byte) *Block {
	var block *Block
	decoder := gob.NewDecoder(bytes.NewReader(b))
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}
	return block
}

// HashTransactions returns a hash of the transactions in the block
func (block *Block) hashTransactions() []byte {
	var transactions [][]byte

	//loop transactions of current block
	for _, tx := range block.Transactions {
		transactions = append(transactions, tx.serialize())
	}
	mTree := NewMerkleTree(transactions)
	return mTree.RootNode.Data
}
