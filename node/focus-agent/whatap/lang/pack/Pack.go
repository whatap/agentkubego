package pack

import (
	"github.com/whatap/go-api/common/io"
)

const (
	PACK_PARAMETER = 0x0100

	TAG_COUNT = 0x1601
	TAG_LOG   = 0x1602

	// 5120
	PACK_EVENT = 0x1400

	PACK_SM_BASE_1     = 0x3008
	PACK_SM_DISK_QUATA = 0x3001
	PACK_SM_NET_PERF   = 0x3002
	PACK_TEXT          = 0x0700
	PACK_KUBE_NODE     = 0x1706
	PACK_LOGSINK       = 0x170a
	PACK_ZIP           = 0x170b

	OS_LINUX       = 1
	OS_WINDOW      = 2
	OS_OSX         = 3
	OS_HPUX        = 4
	OS_AIX         = 5
	OS_SUNOS       = 6
	OS_OPENBSD     = 7
	OS_FREEBSD     = 8
	OS_KUBE_NODE   = 9
	OS_KUBE_MASTER = 10
)

type Pack interface {
	GetPackType() int16
	Write(out *io.DataOutputX)
	Read(in *io.DataInputX)

	// OID, PCODE 설정을 위한 함수
	SetOID(oid int32)
	SetPCODE(pcode int64)
	SetOKIND(okind int32)
	SetONODE(onode int32)
}

func CreatePack(t int16) Pack {
	switch t {
	case TAG_COUNT:
		return NewTagCountPack()
	case TAG_LOG:
		return NewTagLogPack()
	case PACK_EVENT:
		return NewEventPack()
	case PACK_PARAMETER:
		return NewParamPack()
	case PACK_TEXT:
		return NewTextPack()
	case PACK_LOGSINK:
		return NewLogSinkPack()
	case PACK_ZIP:
		return NewZipPack()
	}
	return nil
}

func WritePack(out *io.DataOutputX, p Pack) *io.DataOutputX {
	out.WriteShort(int16(p.GetPackType()))
	p.Write(out)
	return out
}
func ReadPack(in *io.DataInputX) Pack {
	t := in.ReadShort()
	v := CreatePack(t)
	if v != nil {
		v.Read(in)
	}

	return v
}

func ToBytesPack(p Pack) []byte {
	out := io.NewDataOutputX()
	WritePack(out, p)
	return out.ToByteArray()
}
func ToPack(b []byte) Pack {
	in := io.NewDataInputX(b)
	return ReadPack(in)
}
