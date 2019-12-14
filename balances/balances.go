package balances

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/SebastianJ/harmony-tx-sender/nodes"
	"github.com/harmony-one/go-sdk/pkg/common"
	"github.com/harmony-one/go-sdk/pkg/rpc"
	"github.com/harmony-one/go-sdk/pkg/sharding"
)

// CheckBalance - check the balance of a given address using a specified node
func CheckBalance(address string, node string) error {
	if nodes.CheckNodeInput(node) {
		balanceRPCReply, err := rpc.Request(rpc.Method.GetBalance, node, []interface{}{address, "latest"})
		if err != nil {
			return err
		}
		nodeRPCReply, err := rpc.Request(rpc.Method.GetNodeMetadata, node, []interface{}{})
		if err != nil {
			return err
		}
		balance, _ := balanceRPCReply["result"].(string)
		bln, _ := big.NewInt(0).SetString(balance[2:], 16)
		var out bytes.Buffer
		out.WriteString("[")
		out.WriteString(fmt.Sprintf(`{"shard":%d, "amount":%s}`,
			uint64(nodeRPCReply["result"].(map[string]interface{})["shard-id"].(float64)),
			common.ConvertBalanceIntoReadableFormat(bln),
		))
		out.WriteString("]")
		fmt.Println(common.JSONPrettyFormat(out.String()))
		return nil
	}
	r, err := sharding.CheckAllShards(node, address, false)
	if err != nil {
		return err
	}
	fmt.Println(r)
	return nil
}
