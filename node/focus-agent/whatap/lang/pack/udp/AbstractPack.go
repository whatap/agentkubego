package udp

import (
	"fmt"

	"github.com/whatap/go-api/common/io"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/dateutil"
)

const ()

type AbstractPack struct {
	Ver      int32
	Txid     int64
	Time     int64
	Elapsed  int32
	Cpu      int64
	Mem      int64
	Pid      int32
	ThreadId int64

	Flush bool
}

func (this *AbstractPack) Write(dout *io.DataOutputX) {
	dout.WriteLong(this.Txid)
	dout.WriteLong(this.Time)
	dout.WriteInt(this.Elapsed)
	dout.WriteLong(this.Cpu)
	dout.WriteLong(this.Mem)
	if this.Ver >= 10101 {
		dout.WriteInt(this.Pid)
	}
	if this.Ver >= 10104 {
		dout.WriteLong(this.ThreadId)
	}
}
func (this *AbstractPack) Read(din *io.DataInputX) {
	this.Txid = din.ReadLong()
	this.Time = din.ReadLong()
	this.Elapsed = din.ReadInt()
	this.Cpu = din.ReadLong()
	this.Mem = din.ReadLong()
	if this.Ver >= 10101 {
		this.Pid = din.ReadInt()
	}
	if this.Ver >= 10104 {
		this.ThreadId = din.ReadLong()
	}
}

// oid 설정   pack interface
func (this *AbstractPack) SetVersion(ver int32) {
	this.Ver = ver
}

// oid 설정   pack interface
func (this *AbstractPack) GetVersion() int32 {
	return this.Ver
}

func (this *AbstractPack) SetFlush(flush bool) {
	this.Flush = flush
}
func (this *AbstractPack) IsFlush() bool {
	return this.Flush
}

func (this *AbstractPack) ToString() string {
	return fmt.Sprint(dateutil.TimeStamp(this.Time), " Txid=", this.Txid, ",ver=", this.Ver)
}
