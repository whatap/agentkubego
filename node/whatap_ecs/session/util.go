package session

import (
	"fmt"
	gonet "net"

	"github.com/whatap/go-api/common/io"
	"github.com/whatap/go-api/common/util/iputil"
)

var (
	myaddr    int32
	myaddrerr error
)

func getMyAddr() int32 {
	if myaddr == 0 && myaddrerr == nil {

		addrs, err := gonet.InterfaceAddrs()
		if err != nil {
			myaddrerr = err
		}

		for _, a := range addrs {
			if ipnet, ok := a.(*gonet.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					io.ToInt(iputil.ToBytes(ipnet.IP.String()), 0)
				}
			}
		}
		myaddrerr = fmt.Errorf("addr not found")
	}

	return myaddr
}
