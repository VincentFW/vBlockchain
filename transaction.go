package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/elliptic"
	"math/big"
)

const subsidy = 2100

type Transaction struct {
	ID   []byte
	Vin  []TXInput
	Vout []TXOutput
}

func (tx *Transaction) isCoinbase() bool {
	return len(tx.Vin) == 1 && len(tx.Vin[0].Txid) == 0 && tx.Vin[0].Vout == -1
}

// Serialize returns a serialized Transaction
func (tx *Transaction) serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	if err != nil {
		log.Panic(err)
	}

	return encoded.Bytes()
}

// Hash returns the hash of the Transaction
func (tx *Transaction) hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.serialize())

	return hash[:]
}

// String returns a human-readable representation of a transaction
func (tx *Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("Transaction %x:", tx.ID))

	for i, input := range tx.Vin {
		lines = append(lines, fmt.Sprintf("  Input %d:", i))
		lines = append(lines, fmt.Sprintf("    TXID:      %x", input.Txid))
		lines = append(lines, fmt.Sprintf("    Out:       %d", input.Vout))
		lines = append(lines, fmt.Sprintf("    Signature: %x", input.Signature))
		lines = append(lines, fmt.Sprintf("    PubKey:    %x", input.PubKey))

	}

	for i, output := range tx.Vout {
		lines = append(lines, fmt.Sprintf("  Output %d:", i))
		lines = append(lines, fmt.Sprintf("    Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("    Script: %x", output.PubKeyHash))
	}

	return strings.Join(lines, "\n")
}

func (tx *Transaction) trimmedCopy() Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	for _, in := range tx.Vin {
		inputs = append(inputs, TXInput{in.Txid, in.Vout, nil, nil})
	}

	for _, out := range tx.Vout {
		outputs = append(outputs,TXOutput{out.Value,out.PubKeyHash})
	}

	txCopy := Transaction{tx.ID,inputs,outputs}

	return txCopy
}

func (tx *Transaction) sign(privkey ecdsa.PrivateKey, prevTXs map[string]Transaction) {
	if tx.isCoinbase() {
		return
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.trimmedCopy()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash

		//txCopy.hash()
		//txCopy.Vin[inID].PubKey = nil
		dataToSign := fmt.Sprintf("%x\n", txCopy)
		r, s, err := ecdsa.Sign(rand.Reader, &privkey, []byte(dataToSign))
		if err != nil {
			log.Panic(err)
		}
		signature := append(r.Bytes(), s.Bytes()...)

		tx.Vin[inID].Signature = signature
		txCopy.Vin[inID].PubKey = nil
	}

}

func (tx *Transaction) verify(prevTXs map[string]Transaction) bool {
	if tx.isCoinbase() {
		return true
	}

	for _, vin := range tx.Vin {
		if prevTXs[hex.EncodeToString(vin.Txid)].ID == nil {
			log.Panic("ERROR: Previous transaction is not correct")
		}
	}

	txCopy := tx.trimmedCopy()
	curve := elliptic.P256()

	for inID, vin := range tx.Vin {
		prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
		txCopy.Vin[inID].Signature = nil
		txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash

		//txCopy.hash()
		//txCopy.Vin[inID].PubKey = nil


		r := big.Int{}
		s := big.Int{}
		sigLen := len(vin.Signature)
		r.SetBytes(vin.Signature[:(sigLen / 2)])
		s.SetBytes(vin.Signature[(sigLen / 2):])


		x := big.Int{}
		y := big.Int{}
		keyLen := len(vin.PubKey)
		x.SetBytes(vin.PubKey[:(keyLen / 2)])
		y.SetBytes(vin.PubKey[(keyLen / 2):])
		dataToVerify := fmt.Sprintf("%x\n", txCopy)


		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, []byte(dataToVerify), &r, &s) == false {
			return false
		}
		txCopy.Vin[inID].PubKey = nil
	}

	return true
}

func NewCoinbaseTransaction(to, sig string) *Transaction {
	if sig == "" {
		randData := make([]byte, 20)
		_, err := rand.Read(randData)
		if err != nil {
			log.Panic(err)
		}

		sig = fmt.Sprintf("%x", randData)
	}

	txin := TXInput{[]byte{}, -1, nil, []byte(sig)}
	txout := NewTXOutput(subsidy, to)
	tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}}
	tx.ID = tx.hash()

	return &tx
}

func (bc *Blockchain) NewUTXOTransaction(wallet *Wallet, to string, amount int, UTXOSet UTXOSet) *Transaction {
	var inputs []TXInput
	var outputs []TXOutput

	//wallet := wallets.GetWallet(from)
	pubKeyHash := HashPubKey(wallet.PublicKey)
	acc, validOutputs := UTXOSet.findSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		log.Panic("ERROR: Not enough funds")
	}

	// Build a list of inputs
	from := fmt.Sprintf("%s", wallet.getAddress())
	for txid, outs := range validOutputs {
		txID, err := hex.DecodeString(txid)
		if err != nil {
			log.Panic(err)
		}

		for _, out := range outs {
			input := TXInput{txID, out, nil, wallet.PublicKey}
			inputs = append(inputs, input)
		}
	}

	// Build a list of outputs
	outputs = append(outputs, *NewTXOutput(amount, to))
	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from)) // a change
	}

	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.hash()
	UTXOSet.Blockchain.signTransaction(&tx, wallet.PrivateKey)

	return &tx
}

// DeserializeTransaction deserializes a transaction
func DeserializeTransaction(data []byte) Transaction {
	var transaction Transaction

	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&transaction)
	if err != nil {
		log.Panic(err)
	}

	return transaction
}
