package udp

import (
	"github.com/whatap/go-api/common/io"
)

const (
	UDP_PACK_VERSION = 20101

	TX_BLANK uint8 = 0
	TX_START uint8 = 1
	TX_END   uint8 = 255

	TX_DB_CONN   uint8 = 2
	TX_DB_FETCH  uint8 = 3
	TX_SQL       uint8 = 4
	TX_SQL_START uint8 = 5
	TX_SQL_END   uint8 = 6

	TX_HTTPC       uint8 = 7
	TX_HTTPC_START uint8 = 8
	TX_HTTPC_END   uint8 = 9

	TX_ERROR  uint8 = 10
	TX_MSG    uint8 = 11
	TX_METHOD uint8 = 12

	// secure msg
	TX_SECURE_MSG uint8 = 13

	TX_PARAM     uint8 = 30
	ACTIVE_STACK uint8 = 40
	ACTIVE_STATS uint8 = 41

	// relay pack
	RELAY_PACK uint8 = 244
)

type UdpPack interface {
	GetPackType() uint8
	Write(out *io.DataOutputX)
	Read(in *io.DataInputX)

	SetVersion(ver int32)
	GetVersion() int32

	SetFlush(flush bool)
	IsFlush() bool
}

func CreatePack(t uint8) UdpPack {
	switch t {
	case TX_START:
		return NewUdpTxStartPack()
	case TX_SQL:
		return NewUdpTxSqlPack()
	case TX_MSG:
		return NewUdpTxMessagePack()
	case TX_END:
		return NewUdpTxEndPack()
	}
	return nil
}

func WritePack(out *io.DataOutputX, p UdpPack) *io.DataOutputX {
	p.Write(out)
	return out
}
func ReadPack(t uint8, in *io.DataInputX) UdpPack {
	v := CreatePack(uint8(t))
	v.Read(in)
	return v
}

func ToBytesPack(p UdpPack) []byte {
	out := io.NewDataOutputX()
	WritePack(out, p)
	return out.ToByteArray()
}
func ToPack(t uint8, b []byte) UdpPack {
	in := io.NewDataInputX(b)
	return ReadPack(t, in)
}
