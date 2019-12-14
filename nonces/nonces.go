package nonces

import (
	"strconv"

	"github.com/harmony-one/go-sdk/pkg/rpc"
	"github.com/harmony-one/go-sdk/pkg/transaction"
)

// GetNonceFromInput - get a specific nonce from input or form the network
func GetNonceFromInput(addr, inputNonce string, messenger rpc.T) (uint64, error) {
	if inputNonce != "" {
		nonce, err := strconv.ParseUint(inputNonce, 10, 64)
		if err != nil {
			return 0, err
		} else {
			return nonce, nil
		}
	} else {
		return transaction.GetNextNonce(addr, messenger), nil
	}
}
