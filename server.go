package main

import (
	"fmt"
	"net"
	"io"
	"log"
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"encoding/hex"
)

const protocol = "tcp"
//const dnsNodeID = "3000"
const nodeVersion = 1
const commandLength = 12

var nodeAddress string
var miningAddress string
var knownNodes = []string{"172.17.0.2:3000"}
var blocksInTransit = [][]byte{}
var mempool = make(map[string]Transaction)

type addr struct {
	AddrList []string
}

type tx struct {
	AddFrom     string
	Transaction []byte
}

type verzion struct {
	Version  int
	BestHeight int
	AddrFrom string
}

type getblocks struct {
	AddrFrom string
}

type inv struct {
	AddrFrom string
	Type  string
	Items [][]byte
}

type block struct {
	AddrFrom string
	Block    []byte
}

type getdata struct {
	AddrFrom string
	Type     string
	ID       []byte
}

func CommandToBytes(command string) []byte {
	var bytes [commandLength]byte
	for i, c := range command {
		bytes[i] = byte(c)
	}
	return bytes[:]
}

func BytesToCommand(bytes []byte) string {
	var command []byte
	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

func ExtractCommand(request []byte) []byte {
	return request[:commandLength]
}

func RequestBlocks() {
	for _, node := range knownNodes {
		SendGetBlocks(node)
	}
}

func SendAddr(address string) {
	nodes := addr{knownNodes}
	nodes.AddrList = append(nodes.AddrList,nodeAddress)
	payload := GobEncode(nodes)
	request := append(CommandToBytes("addr"),payload...)
	SendData(address,request)
}

func SendData(addr string, data []byte) {
	fmt.Printf("SendData addr is %s \n", addr)
	conn, err := net.Dial(protocol,addr)
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string

		for _, node := range knownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		knownNodes = updatedNodes

		return
	}
	defer conn.Close()
	fmt.Printf("send data: %x\n", data)
	_, err = io.Copy(conn,bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func SendVersion(addr string, bc *Blockchain) {
	bestHeight := bc.getBestHeight()
	payload := GobEncode(verzion{nodeVersion, bestHeight,nodeAddress})

	request := append(CommandToBytes("version"), payload...)

	SendData(addr, request)

}

//func SendVrack(addr string) {
//	payload := GobEncode(verack{})
//	request := append(CommandToBytes("verack"),payload...)
//	SendData(addr,request)
//}

func SendInv(address, kind string, items [][]byte) {
	inventory := inv{nodeAddress, kind,items}
	payload := GobEncode(inventory)

	request := append(CommandToBytes("inv"),payload...)

	SendData(address,request)
}

func SendBlock(address string, b *Block) {
	data := block{nodeAddress, b.Serialize()}
	payload := GobEncode(data)
	request := append(CommandToBytes("block"),payload...)

	SendData(address,request)
}

func SendTx(addr string, tnx *Transaction) {
	data := tx{nodeAddress,tnx.serialize()}
	payload := GobEncode(data)
	request := append(CommandToBytes("tx"),payload...)

	SendData(addr,request)
}

func SendGetData(address, kind string, id []byte) {
	payload := GobEncode(getdata{nodeAddress,kind,id})
	request := append(CommandToBytes("getdata"),payload...)

	SendData(address,request)
}

func SendGetBlocks(address string) {
	payload := GobEncode(getblocks{nodeAddress})
	fmt.Printf("SendGetBlocks is %s\n",payload)
	request := append(CommandToBytes("getblocks"),payload...)
	SendData(address,request)
}

func HandleConnection(conn net.Conn, bc *Blockchain) {
	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	command := BytesToCommand(request[:commandLength])
	fmt.Printf("Received %s command\n", command)
	switch command {
	case "addr":
		HandleAddr(request)
	case "version":
		// send verack
		// send addr
		fmt.Printf("Received %s command\n", command)
		HandleVersion(request, bc)
	case "inv":
		HandleInv(request, bc)
	case "getblocks":
		HandleGetBlocks(request, bc)
	case "block":
		HandleBlock(request, bc)
	case "getdata":
		HandleGetData(request, bc)
	case "tx":
		HandleTx(request, bc)
	default:
		fmt.Println("Unknown command received!")
	}
	conn.Close()

}

func HandleVersion(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload verzion

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("HandleVersion payload is %s\n",payload)

	myBestHeight := bc.getBestHeight()
	foreignerBestHeight := payload.BestHeight

	if myBestHeight < foreignerBestHeight {
		SendGetBlocks(payload.AddrFrom)
	} else if myBestHeight > foreignerBestHeight {
		SendVersion(payload.AddrFrom, bc)
	}

	// sendAddr(payload.AddrFrom)
	if !nodeIsKnown(payload.AddrFrom) {
		knownNodes = append(knownNodes, payload.AddrFrom)
	}

}

func HandleAddr(request []byte) {
	var buff bytes.Buffer
	var payload addr

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	knownNodes = append(knownNodes,payload.AddrList...)
	fmt.Printf("There are %d known nodes now!\n", len(knownNodes))
	RequestBlocks()
}

func HandleInv(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("Recevied inventory with %d %s\n", len(payload.Items), payload.Type)
	if payload.Type == "block" {
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		SendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]
		if mempool[hex.EncodeToString(txID)].ID == nil {
			SendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}

//
func HandleGetBlocks(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getblocks

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := bc.getBlockHashes()

	fmt.Printf("HandleGetBlocks request is %s\n",string(request))
	fmt.Printf("HandleGetBlocks payload is %s\n",payload)
	SendInv(payload.AddrFrom,"block",blocks)
}

func HandleBlock(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload block

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	blockData := payload.Block
	block := DeserializeBlock(blockData)
	fmt.Println("Recevied a new block!")
	bc.addBlock(block)


	fmt.Printf("Added block %x\n", block.Hash)
	fmt.Printf("Added block %d\n", block.Height)
	//UTXOSet := UTXOSet{bc}
	//fmt.Println(blocksInTransit)
	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		SendGetData(payload.AddrFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	} else {
		UTXOSet := UTXOSet{bc}
		UTXOSet.update(block)
		UTXOSet.reindex()
	}
}

func HandleGetData(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getdata

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Type == "block" {
		block, err := bc.getBlock([]byte(payload.ID))
		if err != nil {
			return
		}

		SendBlock(payload.AddrFrom,&block)
	}

	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := mempool[txID]

		SendTx(payload.AddrFrom, &tx)
		// delete(mempool, txID)
	}
}

func HandleTx(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := DeserializeTransaction(txData)
	mempool[hex.EncodeToString(tx.ID)] = tx

	if nodeAddress == knownNodes[0] {
		for _, node := range knownNodes {
			if node != nodeAddress && node != payload.AddFrom {
				SendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		if len(mempool) >= 2 && len(miningAddress) > 0 {
		MineTransactions:
			var txs []*Transaction

			for id := range mempool {
				tx := mempool[id]
				if bc.verifyTransaction(&tx) {
					txs = append(txs, &tx)
				}
			}

			if len(txs) == 0 {
				fmt.Println("All transactions are invalid! Waiting for new ones...")
				return
			}

			cbTx := NewCoinbaseTransaction(miningAddress, "")
			txs = append(txs, cbTx)

			newBlock := bc.MineBlock(txs)
			UTXOSet := UTXOSet{bc}
			UTXOSet.reindex()

			fmt.Println("New block is mined!")

			for _, tx := range txs {
				txID := hex.EncodeToString(tx.ID)
				delete(mempool, txID)
			}

			for _, node := range knownNodes {
				if node != nodeAddress {
					SendInv(node, "block", [][]byte{newBlock.Hash})
				}
			}

			if len(mempool) > 0 {
				goto MineTransactions
			}
		}
	}
}

func StartServer(nodeID, minerAddress string) {

	nodeAddress = fmt.Sprintf("%s:%s", GetLocalIP(),nodeID)
	fmt.Printf("nodeAddress is %s\n", nodeAddress)
	miningAddress = minerAddress
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()

	bc := NewBlockChain(nodeID)

	if nodeAddress != knownNodes[0] {
		SendVersion(knownNodes[0], bc)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}

		go HandleConnection(conn,bc)
	}
}

func GobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func nodeIsKnown(addr string) bool {
	for _, node := range knownNodes {
		if node == addr {
			return true
		}
	}

	return false
}
