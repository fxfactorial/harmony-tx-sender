package transactions

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/harmony-one/go-sdk/pkg/address"
	"github.com/harmony-one/go-sdk/pkg/common"
	"github.com/harmony-one/go-sdk/pkg/rpc"
	"github.com/harmony-one/go-sdk/pkg/transaction"
	"github.com/harmony-one/harmony/accounts"
	"github.com/harmony-one/harmony/accounts/keystore"
	"github.com/harmony-one/harmony/common/denominations"
	"github.com/harmony-one/harmony/core"
)

var (
	debug bool = false
)

type interfaceWrapper []interface{}

// SendTransaction - send transactions
func SendTransaction(keystore *keystore.KeyStore, account *accounts.Account, networkHandler *rpc.HTTPMessenger, chain *common.ChainID, fromAddress string, fromShardID uint32, toAddress string, toShardID uint32, amount float64, gasPrice int64, nonce uint64, inputData string, keystorePassphrase string, node string) (*string, error) {
	params, err := generateTransactionParams(keystore, account, networkHandler, fromAddress, fromShardID, toAddress, toShardID, amount, gasPrice, nonce, inputData)

	if err != nil {
		return nil, err
	}

	tx := generateTransaction(inputData, amount, gasPrice, params)

	signature, err := generateSignature(keystore, account, tx, chain.Value)

	if err != nil {
		return nil, err
	}

	receiptHash, err := sendRPCTransaction(networkHandler, signature)

	if err != nil {
		return nil, err
	}

	return receiptHash, nil
}

func generateTransaction(inputData string, amount float64, gasPrice int64, params map[string]interface{}) *transaction.Transaction {
	amountBigInt := big.NewInt(int64(amount * denominations.Nano))
	amt := amountBigInt.Mul(amountBigInt, big.NewInt(denominations.Nano))
	gPrice := big.NewInt(gasPrice)
	gPrice = gPrice.Mul(gPrice, big.NewInt(denominations.Nano))

	tx := transaction.NewTransaction(
		params["nonce"].(uint64),
		params["gas"].(uint64),
		params["receiver"].(address.T),
		params["from-shard"].(uint32),
		params["to-shard"].(uint32),
		amt,
		gPrice,
		[]byte(inputData),
	)

	return tx
}

func generateTransactionParams(keystore *keystore.KeyStore, account *accounts.Account, networkHandler *rpc.HTTPMessenger, fromAddress string, fromShardID uint32, toAddress string, toShardID uint32, amount float64, gasPrice int64, nonce uint64, inputData string) (map[string]interface{}, error) {
	var params map[string]interface{}
	params = make(map[string]interface{})

	params["from-shard"] = uint32(fromShardID)
	params["to-shard"] = uint32(toShardID)

	base64InputData, err := base64.StdEncoding.DecodeString(inputData)

	if err != nil {
		return nil, err
	}

	gas, err := core.IntrinsicGas(base64InputData, true, true)

	if err != nil {
		return nil, err
	}

	if gas == 0 {
		return nil, errors.New("calculated gas is 0 - this shouldn't be possible")
	}

	params["gas"] = gas
	params["gas-price"] = nil

	amountBigInt := big.NewInt(int64(amount * denominations.Nano))
	amt := amountBigInt.Mul(amountBigInt, big.NewInt(denominations.Nano))
	params["transfer-amount"] = amt

	params["receiver"] = address.Parse(toAddress)

	params["nonce"] = nonce

	if debug {
		fmt.Println("")
		data, _ := json.MarshalIndent(params, "", "  ")
		fmt.Print(string(data))
		fmt.Println("")
	}

	return params, nil
}

func generateSignature(keystore *keystore.KeyStore, account *accounts.Account, tx *transaction.Transaction, chainID *big.Int) (*string, error) {
	signedTransaction, err := keystore.SignTx(*account, tx, chainID)

	if err != nil {
		return nil, err
	}

	enc, err := rlp.EncodeToBytes(signedTransaction)

	if err != nil {
		return nil, err
	}

	hexSignature := hexutil.Encode(enc)
	signature := &hexSignature

	return signature, nil
}

func sendRPCTransaction(networkHandler *rpc.HTTPMessenger, signature *string) (*string, error) {
	if debug {
		fmt.Println("")
		fmt.Println(fmt.Sprintf(`Sending rawTx - %s - to endpoint %s`, *signature, rpc.Method.SendRawTransaction))
		fmt.Println("")
	}

	reply, err := networkHandler.SendRPC(rpc.Method.SendRawTransaction, interfaceWrapper{signature})
	if err != nil {
		return nil, err
	}

	r, _ := reply["result"].(string)
	receiptHash := &r

	return receiptHash, nil
}
