package udp

import (
	"fmt"

	"github.com/whatap/go-api/common/io"
)

type UdpTxEndPack struct {
	AbstractPack
	Host    string
	Uri     string
	Mtid    string
	Mdepth  string
	Mcaller string

	McallerTxid    string
	McallerPcode   string
	McallerSpec    string
	McallerUrl     string
	McallerPoidKey string
}

func NewUdpTxEndPack() *UdpTxEndPack {
	p := new(UdpTxEndPack)
	p.Ver = UDP_PACK_VERSION
	p.AbstractPack.Flush = true
	return p
}

func (this *UdpTxEndPack) GetPackType() uint8 {
	return TX_END
}

func (this *UdpTxEndPack) ToString() string {
	return fmt.Sprint(this.AbstractPack.ToString(), ",host=", this.Host, ",uri=", this.Uri, ",elapsed=", this.Elapsed)
}

func (this *UdpTxEndPack) Write(dout *io.DataOutputX) {
	this.AbstractPack.Write(dout)
	if this.Ver >= 20101 {
		dout.WriteTextShortLength(this.Host)
		dout.WriteTextShortLength(this.Uri)
		dout.WriteTextShortLength(this.Mtid)
		dout.WriteTextShortLength(this.Mdepth)
		dout.WriteTextShortLength(this.Mcaller)
	} else if this.Ver >= 10102 {
		dout.WriteTextShortLength(this.Host)
		dout.WriteTextShortLength(this.Uri)
		dout.WriteTextShortLength(this.Mtid)
		dout.WriteTextShortLength(this.Mdepth)
		dout.WriteTextShortLength(this.McallerTxid)
		dout.WriteTextShortLength(this.McallerPcode)
		dout.WriteTextShortLength(this.McallerSpec)
		dout.WriteTextShortLength(this.McallerUrl)
		dout.WriteTextShortLength(this.McallerPoidKey)
	}
}

func (this *UdpTxEndPack) Read(din *io.DataInputX) {
	this.AbstractPack.Read(din)
	if this.Ver >= 20101 {
		this.Host = din.ReadTextShortLength()
		this.Uri = din.ReadTextShortLength()
		this.Mtid = din.ReadTextShortLength()
		this.Mdepth = din.ReadTextShortLength()
		this.Mcaller = din.ReadTextShortLength()
	} else if this.Ver >= 10102 {
		this.Host = din.ReadTextShortLength()
		this.Uri = din.ReadTextShortLength()
		this.Mtid = din.ReadTextShortLength()
		this.Mdepth = din.ReadTextShortLength()
		this.McallerTxid = din.ReadTextShortLength()
		this.McallerPcode = din.ReadTextShortLength()
		this.McallerSpec = din.ReadTextShortLength()
		this.McallerUrl = din.ReadTextShortLength()
		this.McallerPoidKey = din.ReadTextShortLength()
	}
}
