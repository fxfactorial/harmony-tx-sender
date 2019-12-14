package shards

import (
	"github.com/SebastianJ/harmony-tx-sender/nodes"
	"github.com/harmony-one/go-sdk/pkg/rpc"
	"github.com/harmony-one/go-sdk/pkg/sharding"
)

// HandlerForShard - get a specific handler for a given shard and node combination
func HandlerForShard(senderShard uint32, node string) (*rpc.HTTPMessenger, error) {
	if nodes.CheckNodeInput(node) {
		return rpc.NewHTTPHandler(node), nil
	}
	s, err := sharding.Structure(node)
	if err != nil {
		return nil, err
	}

	for _, shard := range s {
		if uint32(shard.ShardID) == senderShard {
			return rpc.NewHTTPHandler(shard.HTTP), nil
		}
	}

	return nil, nil
}
