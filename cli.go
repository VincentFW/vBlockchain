package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

type CLI struct {
	// bc *Blockchain
}

func (cli CLI) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  printchain - print all the blocks of the blockchain")
	fmt.Println("  listaddresses - Lists all addresses from the wallet file")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT -mine - Send AMOUNT of coins from FROM address to TO. Mine on the same node, when -mine is set.")
	fmt.Println("  createwallet - Generates a new key-pair and saves it into the wallet file")
	fmt.Println("  createblockchain -address ADDRESS - Create a blockchain and send genesis block reward to ADDRESS")
	fmt.Println("  getbalance -address ADDRESS - Get balance of ADDRESS")
	fmt.Println("  startnode -miner ADDRESS - Start a node with ID specified in NODE_ID env. var. -miner enables mining")
	fmt.Println("  reindexutxo - Rebuilds the UTXO set")

}

func (cli CLI) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		os.Exit(1)
	}
}

func (cli *CLI) startNode(nodeID, minerAddress string) {
	fmt.Printf("Starting node %s\n", nodeID)
	if len(minerAddress) > 0 {
		if ValidateAddress(minerAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minerAddress)
		} else {
			log.Panic("Wrong miner address!")
		}
	}
	StartServer(nodeID, minerAddress)

}

func (cli *CLI) reindexUTXO(nodeID string) {
	bc := NewBlockChain(nodeID)
	utxoSet := UTXOSet{bc}
	utxoSet.reindex()
	fmt.Printf("add utxo into chainstate")
	count := utxoSet.countTransactions()
	fmt.Printf("Done! There are %d transactions in the UTXO set.\n", count)

}

func (cli *CLI) createBlockchain(address string, nodeID string) {
	if !ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := CreateBlockchain(address,nodeID)
	defer bc.db.Close()

	UTXOSet := UTXOSet{bc}
	UTXOSet.reindex()

	fmt.Println("Create Blockchain successfully!")
}

func (cli *CLI) createWallet(nodeID string) {
	wallets, _ := NewWallets(nodeID)

	address := wallets.createWallet(nodeID)
	wallets.saveToFile(nodeID)
	fmt.Printf("Your new address: %s\n", address)
}

func (cli CLI) listAddresses(nodeID string) {
	wallets, _ := NewWallets(nodeID)

	addresses := wallets.getAddresses()
	for _, address := range addresses {
		fmt.Println(address)
	}

}

func (cli CLI) getBalance(address string, nodeID string) {
	if !ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}

	bc := NewBlockChain(nodeID)
	utxoSet := UTXOSet{bc}
	defer bc.db.Close()

	balance := 0
	//get all unspentTXs by address
	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	//utxos := bc.findUTXOs(pubKeyHash)
	utxos := utxoSet.findUTXO(pubKeyHash)
	for _, out := range utxos {
		balance += out.Value
	}
	fmt.Printf("Balance of '%s': %d\n", address, balance)
}

func (cli *CLI) send(from, to string, value int, nodeID string, mineNow bool) {
	if !ValidateAddress(from) {
		log.Panic("ERROR: From address is not valid")
	}

	if !ValidateAddress(to) {
		log.Panic("ERROR: To address is not valid")
	}

	bc := NewBlockChain(nodeID)
	utxoSet := UTXOSet{bc}
	defer bc.db.Close()

	wallets, err := NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)


	tx := bc.NewUTXOTransaction(&wallet, to, value, utxoSet)


	if mineNow {
		coinbase := NewCoinbaseTransaction(from,"")   // add coinbase reward to tx sender
		txs := []*Transaction{coinbase,tx}
		newBlock := bc.MineBlock(txs)
		utxoSet.update(newBlock)
	} else {
		SendTx(knownNodes[0], tx)
	}

	fmt.Printf("Send amount successfuly!")
}

func (cli CLI) printChain(nodeID string) {
	bc := NewBlockChain(nodeID)
	defer bc.db.Close()
	bci := bc.Iterator()

	for {
		block := bci.next()
		fmt.Printf("============ Block %x ============\n", block.Hash)
		fmt.Printf("Prev. block: %x\n", block.PrevBlockHash)
		fmt.Printf("Height: %d\n", block.Height)

		pow := NewProofOfWork(block)
		fmt.Printf("Target: %v\n", pow.target)
		fmt.Printf("IsVerified: %s\n\n", strconv.FormatBool(pow.validate()))
		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}
		fmt.Println()

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
}

func (cli CLI) run() {
	cli.validateArgs()

	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env. var is not set!")
		os.Exit(1)
	}

	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("printchain", flag.ExitOnError)
	getBalanceCmd := flag.NewFlagSet("getbalance", flag.ExitOnError)
	listAddressesCmd := flag.NewFlagSet("listaddresses", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("createblockchain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("createwallet", flag.ExitOnError)
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	sendMine := sendCmd.Bool("mine", false, "Mine immediately on the same node")
	reindexUTXOCmd := flag.NewFlagSet("reindexutxo", flag.ExitOnError)
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	startNodeCmd := flag.NewFlagSet("startnode", flag.ExitOnError)
	startNodeMiner := startNodeCmd.String("miner", "", "Enable mining mode and send reward to ADDRESS")


	switch os.Args[1] {
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "printchain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "getbalance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createblockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "createwallet":
		err := createWalletCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "listaddresses":
		err := listAddressesCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "reindexutxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "startnode":
		err := startNodeCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		os.Exit(1)
	}

	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			os.Exit(1)
		}
		cli.send(*sendFrom, *sendTo, *sendAmount, nodeID, *sendMine)
	}

	if printChainCmd.Parsed() {
		cli.printChain(nodeID)
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			os.Exit(1)
		}
		cli.getBalance(*getBalanceAddress, nodeID)
	}
	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			os.Exit(1)
		}
		cli.createBlockchain(*createBlockchainAddress,nodeID)
	}
	if createWalletCmd.Parsed() {
		cli.createWallet(nodeID)
	}
	if listAddressesCmd.Parsed() {
		cli.listAddresses(nodeID)
	}
	if reindexUTXOCmd.Parsed() {
		cli.reindexUTXO(nodeID)
	}
	if startNodeCmd.Parsed() {
		cli.startNode(nodeID, *startNodeMiner)
	}
}
