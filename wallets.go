package main

import (
	"encoding/gob"
	"crypto/elliptic"
	"log"
	"io/ioutil"
	"os"
	"bytes"
	"fmt"
)

const walletFile = "wallet_%s.dat"

type Wallets struct {
	Wallets map[string]*Wallet
}

// CreateWallet adds a Wallet to Wallets
func (ws *Wallets) createWallet(nodeID string) string {
	wallet := NewWallet(nodeID)
	address := fmt.Sprintf("%s", wallet.getAddress())
	ws.Wallets[address] = wallet
	return  address
}

// GetWallet returns a Wallet by its address
func (ws *Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

func NewWallets(nodeID string) (*Wallets, error){
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.loadFromFile(nodeID)
	if err != nil {
		fmt.Println("Wallets file doesn't exist")
	}

	return &wallets, err
}

func (ws *Wallets) getAddresses() []string {
	var addresses []string

	for address := range ws.Wallets  {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws *Wallets) saveToFile(nodeID string) {
	var content bytes.Buffer
	walletFile := fmt.Sprintf(walletFile, nodeID)

	gob.Register(elliptic.P256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}

func (ws *Wallets) loadFromFile(nodeID string) error {
	walletFile := fmt.Sprintf(walletFile, nodeID)
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}
	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}
	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}
	ws.Wallets = wallets.Wallets
	fmt.Printf("Load from local wallets\n")
	return nil
}