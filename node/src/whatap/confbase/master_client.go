package confbase

import (
	"fmt"
	"net"

	"github.com/whatap/kube/node/src/whatap/io"
	"github.com/whatap/kube/node/src/whatap/lang/value"
)

var (
	READ_MAX int32 = 0x800000
)

func sendmap(host string, port int, m *value.MapValue) (ret *value.MapValue, err error) {
	doutx := io.NewDataOutputX()
	value.WriteMapValue(doutx, m)
	dout := io.NewDataOutputX()
	dout.WriteUshort(0xCAFE)
	dout.WriteIntBytes(doutx.ToByteArray())

	proxyAddr := fmt.Sprintf("%s:%d", host, port)
	client, err := net.Dial("tcp", proxyAddr)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	sendbuf := dout.ToByteArray()
	nbyteleft := len(sendbuf)
	buflen := nbyteleft
	for 0 < nbyteleft {
		nbytethistime, err := client.Write(sendbuf[buflen-nbyteleft : buflen])
		if err != nil {
			return nil, err
		}
		nbyteleft -= nbytethistime
	}

	din := io.NewNetReadHelper(client)

	if v, e := din.ReadUnsignedShort(); e == nil && v == 0xCAFE {
		buf, e := din.ReadIntBytesLimit(READ_MAX)

		if e == nil {
			dinx := io.NewDataInputX(buf)
			ret = value.ReadMapValue(dinx)

			return
		} else {
			err = e
		}

	} else if e != nil {
		err = e
	} else {
		err = fmt.Errorf("confbase resposne parse failed ")
	}

	return
}
