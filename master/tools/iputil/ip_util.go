package iputil

import (
	"net"
)

func IsIPv6(address string) bool {
	ip := net.ParseIP(address)
	return ip != nil && ip.To4() == nil
}
