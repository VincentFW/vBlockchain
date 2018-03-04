package main

import "bytes"

type TXInput struct {
	Txid      []byte
	Vout      int
	Signature []byte
	PubKey    []byte
}

func (input *TXInput) canUnlockedWith(pubKeyHash []byte) bool {

	lockingHash := HashPubKey(input.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}
