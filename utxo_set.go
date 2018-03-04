package main

import (
	"github.com/boltdb/bolt"
	"log"
	"encoding/hex"
	"fmt"
)

const utxoBucket = "chainstate"

// UTXOSet represents UTXO set
type UTXOSet struct{
	Blockchain *Blockchain
}

// Reindex rebuilds the UTXO set
func (u *UTXOSet) reindex() {
	db := u.Blockchain.db
	bucketName := []byte(utxoBucket)

	err := db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket(bucketName)
		if err != nil && err != bolt.ErrBucketNotFound {
			log.Panic(err)
		}

		_, err = tx.CreateBucket(bucketName)
		if err != nil {
			log.Panic(err)
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	UTXO := u.Blockchain.findUTXO()
	fmt.Printf("find all UTXO set. length is %d \n",len(UTXO))
	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)

		for txID, outs := range UTXO {
			key, err := hex.DecodeString(txID)
			if err != nil {
				log.Panic(err)
			}

			err = b.Put(key, outs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
}

// FindUTXO finds UTXO for a public key hash
func (u *UTXOSet) findUTXO(keyhash []byte) []TXOutput {
	var utxos []TXOutput

	db := u.Blockchain.db
	
	err := db.View(func(tx *bolt.Tx) error {
		bucketName := []byte(utxoBucket)
		b := tx.Bucket(bucketName)
		
		b.ForEach(func(k, v []byte) error {
			outs := DeserializeOutputs(v)
			for _, out := range outs.Outputs {
				if out.canUnlockedWith(keyhash) {
					utxos = append(utxos, out)
				}
			}
			return nil
		})

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return utxos
}

// FindSpendableOutputs finds and returns unspent outputs to reference in inputs
func (u *UTXOSet) findSpendableOutputs(keyhash []byte, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	accumulated := 0
	db := u.Blockchain.db

	err := db.View(func(tx *bolt.Tx) error {
		bucketName := []byte(utxoBucket)
		b := tx.Bucket(bucketName)
		b.ForEach(func(k, v []byte) error {
			txID := hex.EncodeToString(k)
			outs := DeserializeOutputs(v)
			for outIdx, out := range outs.Outputs {
				if out.canUnlockedWith(keyhash) && accumulated < amount{
					accumulated += out.Value
					unspentOutputs[txID] = append(unspentOutputs[txID],outIdx)
				}
			}
			return nil
		})

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	return accumulated, unspentOutputs

}

// GetCount returns the number of transactions in the UTXO set
func (u *UTXOSet) countTransactions() int {
	counter := 0
	db := u.Blockchain.db

	err := db.View(func(tx *bolt.Tx) error {
		bucketName := []byte(utxoBucket)
		b := tx.Bucket(bucketName)
		b.ForEach(func(k, v []byte) error {
			counter++
			return nil
		})
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return counter
}

// Update updates the UTXO set with transactions from the Block
// The Block is considered to be the tip of a blockchain
func (u *UTXOSet) update(block *Block) {
	db := u.Blockchain.db

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(utxoBucket))

		for _, tx := range block.Transactions {
			if tx.isCoinbase() == false {
				for _, vin := range tx.Vin {
					updatedOuts := TXOutputs{}
					outsBytes := b.Get(vin.Txid)
					outs := DeserializeOutputs(outsBytes)

					for outIdx, out := range outs.Outputs {
						if outIdx != vin.Vout {
							updatedOuts.Outputs = append(updatedOuts.Outputs, out)
						}
					}

					if len(updatedOuts.Outputs) == 0 {
						err := b.Delete(vin.Txid)
						if err != nil {
							log.Panic(err)
						}
					} else {
						err := b.Put(vin.Txid, updatedOuts.Serialize())
						if err != nil {
							log.Panic(err)
						}
					}

				}
			}

			newOutputs := TXOutputs{}
			for _, out := range tx.Vout {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			err := b.Put(tx.ID, newOutputs.Serialize())
			if err != nil {
				log.Panic(err)
			}
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}
