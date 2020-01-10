package balances

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"

	"github.com/SebastianJ/harmony-tx-sender/nodes"
	"github.com/harmony-one/go-sdk/pkg/common"
	"github.com/harmony-one/go-sdk/pkg/rpc"
	"github.com/harmony-one/go-sdk/pkg/sharding"
)

// CheckAllShardBalances - checks the balances in all shards for a given address
func CheckAllShardBalances(node string, oneAddr string) (balances map[int]float64, err error) {
	balances = make(map[int]float64)

	params := []interface{}{oneAddr, "latest"}
	s, err := sharding.Structure(node)
	if err != nil {
		return nil, err
	}

	for _, shard := range s {
		balanceRPCReply, err := rpc.Request(rpc.Method.GetBalance, shard.HTTP, params)
		if err != nil {
			if common.DebugRPC {
				fmt.Printf("NOTE: Route %s failed.", shard.HTTP)
			}
			continue
		}
		balance, _ := balanceRPCReply["result"].(string)
		bln, _ := big.NewInt(0).SetString(balance[2:], 16)

		shardID := shard.ShardID
		formattedAmount := common.ConvertBalanceIntoReadableFormat(bln)
		floatBalance, err := strconv.ParseFloat(formattedAmount, 32)

		if err != nil {
			return nil, err
		}

		balances[shardID] = floatBalance
	}

	return balances, nil
}

// OutputBalance - outputs the balance of a given address using a specified node
func OutputBalance(address string, node string) error {
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
