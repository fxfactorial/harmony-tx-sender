package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/SebastianJ/harmony-tx-sender/balances"
	"github.com/SebastianJ/harmony-tx-sender/nonces"
	"github.com/SebastianJ/harmony-tx-sender/shards"
	"github.com/SebastianJ/harmony-tx-sender/transactions"
	"github.com/SebastianJ/harmony-tx-sender/utils"
	"github.com/harmony-one/go-sdk/pkg/common"
	"github.com/harmony-one/go-sdk/pkg/rpc"
	"github.com/harmony-one/go-sdk/pkg/store"
	"github.com/harmony-one/harmony/accounts"
	"github.com/harmony-one/harmony/accounts/keystore"
	"github.com/urfave/cli"
)

var (
	helpTemplate = `NAME:
   {{.Name}} - {{.Usage}}
USAGE:
   {{.HelpName}} {{if .VisibleFlags}}[global options]{{end}}
   {{if len .Authors}}
AUTHOR:
   {{range .Authors}}{{ . }}{{end}}
   {{end}}{{if .Commands}}
GLOBAL OPTIONS:
   {{range .VisibleFlags}}{{.}}
   {{end}}
VERSION:
   {{.Version}}
   {{end}}
`
	nodeEndpointFlag = cli.StringFlag{
		Name:  "node",
		Usage: "Which node endpoint to use for API commands",
		Value: "https://api.s0.pga.hmny.io",
	}

	fromAddressFlag = cli.StringFlag{
		Name:  "from",
		Usage: "Which address to send tokens from (must exist in the keystore)",
		Value: "",
	}

	fromShardFlag = cli.IntFlag{
		Name:  "from-shard",
		Usage: "What shard to send tokens from",
		Value: 0,
	}

	passPhraseFlag = cli.StringFlag{
		Name:  "passphrase",
		Usage: "Passphrase to use for unlocking the keystore",
		Value: "",
	}

	toShardFlag = cli.IntFlag{
		Name:  "to-shard",
		Usage: "What shard to send tokens to",
		Value: 0,
	}

	amountFlag = cli.Float64Flag{
		Name:  "amount",
		Usage: "How many tokens to send per tx",
		Value: 1.0,
	}

	txCountFlag = cli.IntFlag{
		Name:  "tx-count",
		Usage: "How many transactions to send in total",
		Value: 1000,
	}

	poolSizeFlag = cli.IntFlag{
		Name:  "tx-pool-size",
		Usage: "How many transactions to send simultaneously",
		Value: 100,
	}

	disableRefreshNonceFlag = cli.BoolFlag{
		Name:  "disable-nonce-refresh",
		Usage: "Disable the nonce from getting refreshed before tx pools are executed",
	}

	receiversFileFlag = cli.StringFlag{
		Name:  "receivers",
		Usage: "Which file to use for receiver addresses",
		Value: "./data/receivers.txt",
	}

	txDataFlag = cli.StringFlag{
		Name:  "tx-data",
		Usage: "Which file to use for tx data",
		Value: "./data/tx_data.txt",
	}
)

func main() {
	app := cli.NewApp()
	cli.AppHelpTemplate = helpTemplate
	app.Name = "Harmony Tx Sender CLI App"
	app.Version = fmt.Sprintf("%s/%s-%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	app.Usage = "This is the entry point for starting a new Harmony tx sender"
	app.Flags = []cli.Flag{
		nodeEndpointFlag,
		fromAddressFlag,
		fromShardFlag,
		passPhraseFlag,
		toShardFlag,
		amountFlag,
		txCountFlag,
		poolSizeFlag,
		disableRefreshNonceFlag,
		receiversFileFlag,
		txDataFlag,
	}
	app.Authors = []cli.Author{
		{
			Name:  "Sebastian Johnsson",
			Email: "",
		},
	}

	app.Action = func(context *cli.Context) error {
		return startSender(context)
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func startSender(context *cli.Context) error {
	node := context.GlobalString(nodeEndpointFlag.Name)

	if node == "" {
		return errors.New("you need to specify a node to use for the API calls")
	}

	fromAddress := context.GlobalString(fromAddressFlag.Name)

	if fromAddress == "" {
		return errors.New("you need to specify the sender address")
	}

	fromShardID := uint32(context.GlobalInt(fromShardFlag.Name))
	toShardID := uint32(context.GlobalInt(toShardFlag.Name))

	balances.CheckBalance(fromAddress, node)

	txDataFilePath := context.GlobalString(txDataFlag.Name)
	txData, err := utils.ReadFileToString(txDataFilePath)
	if txData != "" {
		txData = base64.StdEncoding.EncodeToString([]byte(txData))
	}

	txData = ``

	passPhrase := context.GlobalString(passPhraseFlag.Name)
	amount := context.GlobalFloat64(amountFlag.Name)
	gasPrice := int64(1)

	receiversPath, _ := filepath.Abs(context.GlobalString(receiversFileFlag.Name))
	receivers, _ := utils.FetchReceivers(receiversPath)

	if len(receivers) == 0 {
		return fmt.Errorf("you need to create %s and add at least one receiver address to it", receiversPath)
	}

	txCount := context.GlobalInt(txCountFlag.Name)
	maximumPoolSize := context.GlobalInt(poolSizeFlag.Name)

	networkHandler, err := shards.HandlerForShard(fromShardID, node)
	if err != nil {
		log.Fatal(err)
	}

	chain := &common.Chain.DevNet

	currentNonce, err := nonces.GetNonceFromInput(fromAddress, "", networkHandler)

	if err != nil {
		log.Fatal(err)
	}

	keystore, account, err := store.UnlockedKeystore(fromAddress, passPhrase)
	if err != nil {
		log.Fatal(err)
	}

	disableNonceRefresh := context.GlobalBool(disableRefreshNonceFlag.Name)

	/*err = bulkSendTransactions(keystore, account, networkHandler, fromAddress, fromShardID, toAddress, toShardID, amount, gasPrice, currentNonce, txData, passPhrase, node, txCount)

	if err != nil {
		log.Fatal(err)
	}*/

	asyncBulkSendTransactions(keystore, account, networkHandler, chain, fromAddress, fromShardID, receivers, toShardID, amount, gasPrice, currentNonce, txData, passPhrase, node, txCount, maximumPoolSize, disableNonceRefresh)

	return nil
}

func bulkSendTransactions(keystore *keystore.KeyStore, account *accounts.Account, networkHandler *rpc.HTTPMessenger, chain *common.ChainID, fromAddress string, fromShardID uint32, receivers []string, toShardID uint32, amount float64, gasPrice int64, currentNonce uint64, txData string, passPhrase string, node string, txCount int) error {
	randomReceiver := utils.RandomReceiver(receivers)

	for i := 0; i < txCount; i++ {
		err := sendTransaction(keystore, account, networkHandler, chain, fromAddress, fromShardID, randomReceiver, toShardID, amount, gasPrice, currentNonce, txData, passPhrase, node)

		if err != nil {
			return err
		}

		currentNonce++
	}

	return nil
}

func sendTransaction(keystore *keystore.KeyStore, account *accounts.Account, networkHandler *rpc.HTTPMessenger, chain *common.ChainID, fromAddress string, fromShardID uint32, toAddress string, toShardID uint32, amount float64, gasPrice int64, currentNonce uint64, txData string, passPhrase string, node string) error {
	fmt.Println(fmt.Sprintf(`currentNonce is now: %d`, currentNonce))

	txReceipt, err := transactions.SendTransaction(keystore, account, networkHandler, chain, fromAddress, fromShardID, toAddress, toShardID, amount, gasPrice, currentNonce, txData, passPhrase, node)

	if err != nil {
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, err))
		return err
	}

	fmt.Println(fmt.Sprintf(`Receipt hash: %s`, *txReceipt))

	return nil
}

func asyncBulkSendTransactions(keystore *keystore.KeyStore, account *accounts.Account, networkHandler *rpc.HTTPMessenger, chain *common.ChainID, fromAddress string, fromShardID uint32, receivers []string, toShardID uint32, amount float64, gasPrice int64, currentNonce uint64, txData string, passPhrase string, node string, txCount int, maximumPoolSize int, disableNonceRefresh bool) {
	pools := 1

	if disableNonceRefresh {
		os.Exit(1)
	}

	if !disableNonceRefresh && txCount > maximumPoolSize {
		pools = int(math.RoundToEven(float64(txCount) / float64(maximumPoolSize)))
		fmt.Println(fmt.Sprintf(`Number of goroutine pools: %d`, pools))
	}

	for poolIndex := 0; poolIndex < pools; poolIndex++ {
		var waitGroup sync.WaitGroup

		if poolIndex > 1 {
			currentNonce, _ = nonces.GetNonceFromInput(fromAddress, "", networkHandler)
			fmt.Println(fmt.Sprintf(`Nonce refreshed! Nonce is now: %d`, currentNonce))
		}

		for i := 0; i < maximumPoolSize; i++ {
			randomReceiver := utils.RandomReceiver(receivers)
			waitGroup.Add(1)
			go asyncSendTransaction(keystore, account, networkHandler, chain, fromAddress, fromShardID, randomReceiver, toShardID, amount, gasPrice, currentNonce, txData, passPhrase, node, &waitGroup)
			currentNonce++
		}

		waitGroup.Wait()
	}
}

func asyncSendTransaction(keystore *keystore.KeyStore, account *accounts.Account, networkHandler *rpc.HTTPMessenger, chain *common.ChainID, fromAddress string, fromShardID uint32, toAddress string, toShardID uint32, amount float64, gasPrice int64, currentNonce uint64, txData string, passPhrase string, node string, waitGroup *sync.WaitGroup) {
	fmt.Println(fmt.Sprintf(`Sending tx - From: %s, From Shard: %d, To: %s, To Shard: %d, Amount: %f, Nonce: %d`, fromAddress, fromShardID, toAddress, toShardID, amount, currentNonce))

	txReceipt, err := transactions.SendTransaction(keystore, account, networkHandler, chain, fromAddress, fromShardID, toAddress, toShardID, amount, gasPrice, currentNonce, txData, passPhrase, node)

	if err == nil {
		fmt.Println(fmt.Sprintf(`Receipt hash: %s`, *txReceipt))
	} else {
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, err))
	}

	defer waitGroup.Done()
}
