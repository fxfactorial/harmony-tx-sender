package nodes

import (
	"net"
	"strings"
)

// CheckNodeInput - check what node to use
func CheckNodeInput(node string) bool {
	removePrefix := strings.TrimPrefix(node, "http://")
	removePrefix = strings.TrimPrefix(removePrefix, "https://")
	possibleIP := strings.Split(removePrefix, ":")[0]
	return net.ParseIP(string(possibleIP)) != nil
}
