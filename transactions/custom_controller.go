package transactions

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/harmony-one/go-sdk/pkg/address"
	"github.com/harmony-one/go-sdk/pkg/common"
	"github.com/harmony-one/go-sdk/pkg/ledger"
	"github.com/harmony-one/go-sdk/pkg/rpc"
	"github.com/harmony-one/go-sdk/pkg/transaction"
	"github.com/harmony-one/harmony/accounts"
	"github.com/harmony-one/harmony/accounts/keystore"
	"github.com/harmony-one/harmony/common/denominations"
	"github.com/harmony-one/harmony/core"
)

type p []interface{}

type transactionForRPC struct {
	params      map[string]interface{}
	transaction *transaction.Transaction
	// Hex encoded
	signature   *string
	receiptHash *string
	receipt     rpc.Reply
}

type sender struct {
	ks      *keystore.KeyStore
	account *accounts.Account
}

// CustomController drives the transaction signing process
type CustomController struct {
	failure           error
	messenger         rpc.T
	sender            sender
	transactionForRPC transactionForRPC
	chain             common.ChainID
	Behavior          behavior
}

type behavior struct {
	DryRun               bool
	SigningImpl          transaction.SignerImpl
	ConfirmationWaitTime uint32
}

// NewCustomController initializes a Controller, caller can control behavior via options
func NewCustomController(
	handler rpc.T,
	senderKs *keystore.KeyStore,
	senderAcct *accounts.Account,
	chain common.ChainID,
	options ...func(*CustomController)) *CustomController {

	txParams := make(map[string]interface{})
	ctrlr := &CustomController{
		failure:   nil,
		messenger: handler,
		sender: sender{
			ks:      senderKs,
			account: senderAcct,
		},
		transactionForRPC: transactionForRPC{
			params:      txParams,
			signature:   nil,
			receiptHash: nil,
			receipt:     nil,
		},
		chain:    chain,
		Behavior: behavior{false, transaction.Software, 0},
	}
	for _, option := range options {
		option(ctrlr)
	}
	return ctrlr
}

func (C *CustomController) verifyBalance(amount float64) {
	if C.failure != nil {
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, C.failure))
		return
	}
	balanceRPCReply, err := C.messenger.SendRPC(
		rpc.Method.GetBalance,
		p{address.ToBech32(C.sender.account.Address), "latest"},
	)
	if err != nil {
		C.failure = err
		return
	}
	currentBalance, _ := balanceRPCReply["result"].(string)
	balance, _ := big.NewInt(0).SetString(currentBalance[2:], 16)
	balance = common.NormalizeAmount(balance)
	transfer := big.NewInt(int64(amount * denominations.Nano))

	tns := float64(transfer.Uint64()) / denominations.Nano
	bln := float64(balance.Uint64()) / denominations.Nano

	if tns > bln {
		C.failure = fmt.Errorf(
			"current balance of %.6f is not enough for the requested transfer %.6f", bln, tns,
		)
	}
}

func (C *CustomController) sendSignedTx() {
	if C.failure != nil || C.Behavior.DryRun {
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, C.failure))
		return
	}

	fmt.Println("transactionForRPC.params:", C.transactionForRPC.params)
	fmt.Println("transactionForRPC.signature:", C.transactionForRPC.signature)

	reply, err := C.messenger.SendRPC(rpc.Method.SendRawTransaction, p{C.transactionForRPC.signature})
	if err != nil {
		C.failure = err
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, C.failure))
		return
	}
	r, _ := reply["result"].(string)
	C.transactionForRPC.receiptHash = &r
}

func (C *CustomController) setIntrinsicGas(rawInput string) {
	if C.failure != nil {
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, C.failure))
		return
	}
	inputData, _ := base64.StdEncoding.DecodeString(rawInput)
	// NOTE Need to add more intrisicGas
	// in order to include more data in inputData
	gas, _ := core.IntrinsicGas(inputData, false, true)
	C.transactionForRPC.params["gas"] = gas
}

func (C *CustomController) setGasPrice() {
	if C.failure != nil {
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, C.failure))
		return
	}
	C.transactionForRPC.params["gas-price"] = nil
}

func (C *CustomController) setAmount(amount float64) {
	amountBigInt := big.NewInt(int64(amount * denominations.Nano))
	amt := amountBigInt.Mul(amountBigInt, big.NewInt(denominations.Nano))
	C.transactionForRPC.params["transfer-amount"] = amt
}

func (C *CustomController) setReceiver(receiver string) {
	C.transactionForRPC.params["receiver"] = address.Parse(receiver)
}

func (C *CustomController) setNewTransactionWithDataAndGas(i string, amount float64, gasPrice int64) {
	if C.failure != nil {
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, C.failure))
		return
	}
	amountBigInt := big.NewInt(int64(amount * denominations.Nano))
	amt := amountBigInt.Mul(amountBigInt, big.NewInt(denominations.Nano))
	gPrice := big.NewInt(gasPrice)
	gPrice = gPrice.Mul(gPrice, big.NewInt(denominations.Nano))

	tx := transaction.NewTransaction(
		C.transactionForRPC.params["nonce"].(uint64),
		C.transactionForRPC.params["gas"].(uint64),
		C.transactionForRPC.params["receiver"].(address.T),
		C.transactionForRPC.params["from-shard"].(uint32),
		C.transactionForRPC.params["to-shard"].(uint32),
		amt,
		gPrice,
		[]byte(i),
	)
	C.transactionForRPC.transaction = tx
}

// TransactionToJSON dumps JSON rep
func (C *CustomController) TransactionToJSON(pretty bool) string {
	r, _ := C.transactionForRPC.transaction.MarshalJSON()
	if pretty {
		return common.JSONPrettyFormat(string(r))
	}
	return string(r)
}

// RawTransaction dumps the signature as string
func (C *CustomController) RawTransaction() string {
	return *C.transactionForRPC.signature
}

func (C *CustomController) signAndPrepareTxEncodedForSending() {
	if C.failure != nil {
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, C.failure))
		return
	}
	signedTransaction, err :=
		C.sender.ks.SignTx(*C.sender.account, C.transactionForRPC.transaction, C.chain.Value)
	if err != nil {
		C.failure = err
		return
	}
	C.transactionForRPC.transaction = signedTransaction
	enc, _ := rlp.EncodeToBytes(signedTransaction)
	hexSignature := hexutil.Encode(enc)
	C.transactionForRPC.signature = &hexSignature
	if common.DebugTransaction {
		r, _ := signedTransaction.MarshalJSON()
		fmt.Println("Signed with ChainID:", C.transactionForRPC.transaction.ChainID())
		fmt.Println(common.JSONPrettyFormat(string(r)))
	}
}

func (C *CustomController) setShardIDs(fromShard, toShard int) {
	if C.failure != nil {
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, C.failure))
		return
	}
	C.transactionForRPC.params["from-shard"] = uint32(fromShard)
	C.transactionForRPC.params["to-shard"] = uint32(toShard)
}

func (C *CustomController) ReceiptHash() *string {
	return C.transactionForRPC.receiptHash
}

func (C *CustomController) Receipt() rpc.Reply {
	return C.transactionForRPC.receipt
}

func (C *CustomController) hardwareSignAndPrepareTxEncodedForSending() {
	if C.failure != nil {
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, C.failure))
		return
	}
	enc, signerAddr, err := ledger.SignTx(C.transactionForRPC.transaction, C.chain.Value)
	if err != nil {
		C.failure = err
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, C.failure))
		return
	}
	if strings.Compare(signerAddr, address.ToBech32(C.sender.account.Address)) != 0 {
		C.failure = errors.New("signature verification failed : sender address doesn't match with ledger hardware addresss")
		return
	}
	hexSignature := hexutil.Encode(enc)
	C.transactionForRPC.signature = &hexSignature
}

func (C *CustomController) txConfirmation() {
	if C.failure != nil {
		fmt.Println(fmt.Sprintf(`Error occurred: %s`, C.failure))
		return
	}
	if C.Behavior.ConfirmationWaitTime > 0 {
		receipt := *C.ReceiptHash()
		start := int(C.Behavior.ConfirmationWaitTime)
		for {
			if start < 0 {
				return
			}
			r, _ := C.messenger.SendRPC(rpc.Method.GetTransactionReceipt, p{receipt})
			if r["result"] != nil {
				C.transactionForRPC.receipt = r
				return
			}
			time.Sleep(time.Second * 2)
			start = start - 2
		}
	}
}

// ExecuteTransaction is the single entrypoint to execute a transaction.
// Each step in transaction creation, execution probably includes a mutation
// Each becomes a no-op if failure occured in any previous step
func (C *CustomController) ExecuteTransaction(
	to, inputData string,
	amount float64, gPrice int64, nonce uint64,
	fromShard, toShard int,
) error {
	// WARNING Order of execution matters

	fmt.Println("setShardIDs")

	C.setShardIDs(fromShard, toShard)

	fmt.Println("setIntrinsicGas")

	C.setIntrinsicGas(inputData)

	fmt.Println("setAmount")
	C.setAmount(amount)

	/*fmt.Println("verifyBalance")
	C.verifyBalance(amount)*/

	fmt.Println("setReceiver")
	C.setReceiver(to)

	fmt.Println("setGasPrice")
	C.setGasPrice()

	fmt.Println(`C.transactionForRPC.params["nonce"] = nonce`)
	C.transactionForRPC.params["nonce"] = nonce

	fmt.Println("setNewTransactionWithDataAndGas")
	C.setNewTransactionWithDataAndGas(inputData, amount, gPrice)

	fmt.Println("signAndPrepareTxEncodedForSending")
	C.signAndPrepareTxEncodedForSending()

	fmt.Println("sendSignedTx")
	C.sendSignedTx()

	fmt.Println("txConfirmation")
	C.txConfirmation()

	return C.failure
}
