package main

import (
	"github.com/boltdb/bolt"
	"log"
)

type BlockchainIterator struct {
	currentHash []byte
	db          *bolt.DB
}

func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.tip, bc.db}

	return bci
}

func (bci *BlockchainIterator) next() *Block {
	var block *Block

	err := bci.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(bci.currentHash)
		block = DeserializeBlock(encodedBlock)
		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bci.currentHash = block.PrevBlockHash

	return block
}
