package udp

import (
	"fmt"

	"github.com/whatap/go-api/common/io"
)

type UdpTxSqlPack struct {
	AbstractPack
	Dbc string
	Sql string
}

func NewUdpTxSqlPack() *UdpTxSqlPack {
	p := new(UdpTxSqlPack)
	p.Ver = UDP_PACK_VERSION
	p.AbstractPack.Flush = false
	return p
}

func (this *UdpTxSqlPack) GetPackType() uint8 {
	return TX_SQL
}

func (this *UdpTxSqlPack) ToString() string {
	return fmt.Sprint(this.AbstractPack.ToString(), ",dbc=", this.Dbc, ",sql=", this.Sql, ",desc=")
}

func (this *UdpTxSqlPack) Write(dout *io.DataOutputX) {
	this.AbstractPack.Write(dout)
	dout.WriteTextShortLength(this.Dbc)
	dout.WriteTextShortLength(this.Sql)
}

func (this *UdpTxSqlPack) Read(din *io.DataInputX) {
	this.AbstractPack.Read(din)

	this.Dbc = din.ReadTextShortLength()
	this.Sql = din.ReadTextShortLength()
}
