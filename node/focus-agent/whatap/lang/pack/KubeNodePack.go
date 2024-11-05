package pack

import "github.com/whatap/go-api/common/io"

type KubeNodePack struct {
	AbstractPack
	HostIp       int32
	ListenPort   int32
	Starttime    int64
	SysCores     int32
	SysMem       int64
	ContainerKey int32
}

func NewKubeNodePack() *KubeNodePack {
	p := new(KubeNodePack)
	return p
}

func (this *KubeNodePack) GetPackType() int16 {
	return PACK_KUBE_NODE
}

func (this *KubeNodePack) Write(doutx *io.DataOutputX) {
	this.AbstractPack.Write(doutx)
	doutx.WriteByte(1)

	dout := io.NewDataOutputX()
	dout.WriteInt(this.HostIp)
	dout.WriteLong(this.Starttime)
	dout.WriteDecimal(int64(this.SysCores))
	dout.WriteDecimal(this.SysMem)
	dout.WriteDecimal(int64(this.ListenPort))
	dout.WriteDecimal(int64(this.ContainerKey))
	dout.WriteDecimal(0)
	doutx.WriteBlob(dout.ToByteArray())
}

func (this *KubeNodePack) Read(dinx *io.DataInputX) {
	din := io.NewDataInputX(dinx.ReadBlob())
	this.AbstractPack.Read(din)
	this.HostIp = din.ReadInt()
	this.Starttime = din.ReadLong()
	this.SysCores = int32(din.ReadDecimal())
	this.SysMem = din.ReadDecimal()
	this.ListenPort = int32(din.ReadDecimal())
	this.ContainerKey = int32(din.ReadDecimal())
	din.ReadDecimal()
}
