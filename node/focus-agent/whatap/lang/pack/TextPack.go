package pack

import (
	"github.com/whatap/go-api/common/io"
)

const (
	TEXT_SERVICE        = 1
	TEXT_SQL            = 2
	TEXT_DB_URL         = 3
	TEXT_HTTPC_URL      = 4
	TEXT_ERROR          = 5
	TEXT_METHOD         = 6
	TEXT_STACK_ELEMENTS = 7
	TEXT_REFERER        = 8
	TEXT_USER_AGENT     = 9
	TEXT_HTTPC_HOST     = 10
	TEXT_MESSAGE        = 11
	TEXT_CRUD           = 12
	TEXT_ONAME          = 13
	TEXT_COMMAND        = 14

	TEXT_SYS_DEVICE_ID     = 21
	TEXT_SYS_MOUNT_POINT   = 22
	TEXT_SYS_FILE_SYSTEM   = 23
	TEXT_SYS_NET_DESC      = 24
	TEXT_SYS_PROC_CMD1     = 26
	TEXT_SYS_PROC_CMD2     = 27
	TEXT_SYS_PROC_USER     = 28
	TEXT_SYS_PROC_STATE    = 29
	TEXT_SYS_PROC_FILENAME = 30
	TEXT_SM_LOG_FILE       = 31
	TEXT_EXT_META          = 32
	TEXT_SYS_MOUNT_OPTION  = 33

	TEXT_DB_COUNTER_NAME   = 41
	TEXT_DB_COUNTER_UNIT   = 42
	TEXT_CW_AGENT_IP       = 51
	TEXT_CW_MXID           = 52
	TEXT_MTRACE_SPEC       = 53
	TEXT_MTRACE_CALLER_URL = 54
	TEXT_ADDIN_AID_NAME    = 55
	TEXT_ADDIN_CID_NAME    = 56
	TEXT_CONTAINER_ID      = 60
	TEXT_ONODE_NAME        = 63
)

type TextRec struct {
	Div  byte
	Hash int32
	Text string
}
type TextPack struct {
	AbstractPack
	records []TextRec
}

func NewTextPack() *TextPack {
	p := new(TextPack)
	p.records = make([]TextRec, 0, 32)
	return p
}
func (this *TextPack) GetPackType() int16 {
	return PACK_TEXT
}
func (this *TextPack) Write(dout *io.DataOutputX) {
	this.AbstractPack.Write(dout)
	dout.WriteDecimal(int64(len(this.records)))
	for i := 0; i < len(this.records); i++ {
		r := this.records[i]
		dout.WriteByte(r.Div)
		dout.WriteInt(r.Hash)
		dout.WriteText(r.Text)
	}
}
func (this *TextPack) Read(din *io.DataInputX) {
	this.AbstractPack.Read(din)
	size := int(din.ReadDecimal())
	this.records = make([]TextRec, size)
	for i := 0; i < size; i++ {
		div := din.ReadByte()
		hash := din.ReadInt()
		text := din.ReadText()
		this.records[i] = TextRec{div, hash, text}
	}
}
func (this *TextPack) AddTexts(texts []TextRec) {
	if len(this.records) == 0 {
		this.records = texts
	} else {
		this.records = append(this.records, texts...)
	}
}
