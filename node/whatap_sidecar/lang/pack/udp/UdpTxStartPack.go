package udp

import (
	"fmt"

	"github.com/whatap/golib/io"
)

type UdpTxStartPack struct {
	AbstractPack
	Host             string
	Uri              string
	Ipaddr           string
	UAgent           string
	Ref              string
	Userid           string
	HttpMethod       string
	IsStaticContents string
}

func NewUdpTxStartPack() *UdpTxStartPack {
	p := new(UdpTxStartPack)
	p.Ver = UDP_PACK_VERSION
	p.AbstractPack.Flush = true
	return p
}

func (this *UdpTxStartPack) GetPackType() uint8 {
	return TX_START
}

func (this *UdpTxStartPack) ToString() string {
	return fmt.Sprint(this.AbstractPack.ToString(), ",host=", this.Host, ",uri=", this.Uri)
}

func (this *UdpTxStartPack) Write(dout *io.DataOutputX) {
	this.AbstractPack.Write(dout)
	dout.WriteTextShortLength(this.Host)
	dout.WriteTextShortLength(this.Uri)
	dout.WriteTextShortLength(this.Ipaddr)
	dout.WriteTextShortLength(this.UAgent)
	dout.WriteTextShortLength(this.Ref)
	dout.WriteTextShortLength(this.Userid)
	if this.Ver >= 10103 {
		dout.WriteTextShortLength(this.HttpMethod)
	} else if this.Ver >= 20101 {
		dout.WriteTextShortLength(this.IsStaticContents)
	}
}

func (this *UdpTxStartPack) Read(din *io.DataInputX) {
	this.AbstractPack.Read(din)

	this.Host = din.ReadTextShortLength()
	this.Uri = din.ReadTextShortLength()
	this.Ipaddr = din.ReadTextShortLength()
	this.UAgent = din.ReadTextShortLength()
	this.Ref = din.ReadTextShortLength()
	this.Userid = din.ReadTextShortLength()
	if this.Ver >= 10103 {
		this.HttpMethod = din.ReadTextShortLength()
	} else if this.Ver >= 20101 {
		this.IsStaticContents = din.ReadTextShortLength()
	}
}
