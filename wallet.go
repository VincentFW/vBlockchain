package main

import (
	"crypto/sha256"
	"golang.org/x/crypto/ripemd160"
	"log"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/ecdsa"
	"bytes"
)

const version = byte(0x00)
const addressChecksumLen = 4

type Wallet struct {
	PrivateKey ecdsa.PrivateKey
	PublicKey  []byte
}


func (w *Wallet) getAddress() []byte {
	pubKeyHash := HashPubKey(w.PublicKey)
	versionedPayload := append([]byte{version}, pubKeyHash...)

	checksum := checksum(versionedPayload)
	fullPayload := append(versionedPayload, checksum...)

	address := Base58Encode(fullPayload)

	return address
}

// ValidateAddress check if address if valid
func ValidateAddress(address string) bool {
	pubKeyHash := Base58Decode([]byte(address))
	actualChecksum := pubKeyHash[len(pubKeyHash)-addressChecksumLen:]
	version := pubKeyHash[0]
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-addressChecksumLen]
	targetChecksum := checksum(append([]byte{version}, pubKeyHash...))

	return bytes.Compare(actualChecksum, targetChecksum) == 0
}


func newKeyPair() (ecdsa.PrivateKey, []byte) {
	curve := elliptic.P256()
	private, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Panic(err)
	}

	pubKey := append(private.PublicKey.X.Bytes(), private.PublicKey.Y.Bytes()...)

	return *private, pubKey
}

func checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])
	return secondSHA[:addressChecksumLen]// get first 4 bytes
}

// NewWallet creates and returns a Wallet
func NewWallet(nodeID string) *Wallet {
	private, public := newKeyPair()
	wallet := Wallet{private,public}

	return &wallet
}

// HashPubKey hashes public key
func HashPubKey(pubKey []byte) []byte {
	publicSHA256 := sha256.Sum256(pubKey)

	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	if err != nil {
		log.Panic(err)
	}
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160
}

