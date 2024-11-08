package udp

import (
	"fmt"

	"github.com/whatap/go-api/common/io"
)

type UdpTxMethodPack struct {
	AbstractPack
	Method string
	Stack  string
}

func NewUdpTxMethodPack() *UdpTxMethodPack {
	p := new(UdpTxMethodPack)
	p.Ver = UDP_PACK_VERSION
	p.AbstractPack.Flush = false
	return p
}

func (this *UdpTxMethodPack) GetPackType() uint8 {
	return TX_METHOD
}

func (this *UdpTxMethodPack) ToString() string {
	return fmt.Sprint(this.AbstractPack.ToString(), ",method=", this.Method, ",stack=", this.Stack)
}

func (this *UdpTxMethodPack) Write(dout *io.DataOutputX) {
	this.AbstractPack.Write(dout)
	dout.WriteTextShortLength(this.Method)
	dout.WriteTextShortLength(this.Stack)
}

func (this *UdpTxMethodPack) Read(din *io.DataInputX) {
	this.AbstractPack.Read(din)

	this.Method = din.ReadTextShortLength()
	this.Stack = din.ReadTextShortLength()
}
