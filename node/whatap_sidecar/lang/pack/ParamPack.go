package pack

import (
	"bytes"
	"fmt"

	"github.com/whatap/golib/io"
	val "github.com/whatap/golib/lang/value"
	"github.com/whatap/golib/util/hmap"
)

const (
	MAX_CHILD = 1024
)

type ParamPack struct {
	AbstractPack
	Id       int32
	table    *hmap.StringKeyLinkedMap
	Request  int64
	Response int64
}

func NewParamPack() *ParamPack {
	p := new(ParamPack)
	p.table = hmap.NewStringKeyLinkedMap()
	return p
}

func (this *ParamPack) ContainsKey(key string) bool {
	return this.table.ContainsKey(key)
}

func (this *ParamPack) Get(key string) val.Value {
	o := this.table.Get(key)
	if o == nil {
		return val.NewNullValue()
	}
	return o.(val.Value)
}

func (this *ParamPack) GetLong(key string) int64 {
	o := this.table.Get(key)
	if o != nil {
		v := o.(val.Value)
		if v.GetValueType() == val.VALUE_DECIMAL {
			return o.(*val.DecimalValue).Val
		}
	}

	return 0
}

func (this *ParamPack) GetMap(key string) *val.MapValue {
	o := this.table.Get(key)
	if o == nil {
		return nil
	}
	return o.(*val.MapValue)
}

func (this *ParamPack) GetList(key string) *val.ListValue {
	o := this.table.Get(key)
	if o == nil {
		return nil
	}
	if val, ok := o.(*val.ListValue); ok {
		return val
	}
	return nil
}

func (this *ParamPack) GetString(key string) string {
	o := this.Get(key)
	if o.GetValueType() == val.VALUE_TEXT {
		t := o.(*val.TextValue)
		return t.Val
	}
	return ""
}

func (this *ParamPack) PutString(key string, value string) {
	this.Put(key, val.NewTextValue(value))
}
func (this *ParamPack) PutLong(key string, value int64) {
	this.Put(key, val.NewDecimalValue(value))
}
func (this *ParamPack) Put(key string, value val.Value) {
	this.table.Put(key, value)
}
func (this *ParamPack) PutList(key string, value *val.ListValue) {
	this.table.Put(key, value)
}
func (this *ParamPack) PutMap(key string, value *val.MapValue) {
	this.table.Put(key, value)
}
func (this *ParamPack) Clear() {
	this.Clear()
}
func (this *ParamPack) Size() {
	this.table.Size()
}
func (this *ParamPack) Keys() hmap.StringEnumer {
	return this.table.Keys()
}

func (this *ParamPack) GetPackType() int16 {
	return PACK_PARAMETER
}
func (this *ParamPack) Write(dout *io.DataOutputX) {
	this.AbstractPack.Write(dout)
	dout.WriteInt(this.Id)
	dout.WriteDecimal(this.Request)
	dout.WriteDecimal(this.Response)
	dout.WriteDecimal(int64(this.table.Size()))

	keys := this.Keys()
	for keys.HasMoreElements() {
		key := keys.NextString()
		value := this.table.Get(key).(val.Value)

		dout.WriteText(key)
		val.WriteValue(dout, value)
	}
}
func (this *ParamPack) Read(din *io.DataInputX) {
	this.AbstractPack.Read(din)
	this.Id = din.ReadInt()
	this.Request = din.ReadDecimal()
	this.Response = din.ReadDecimal()
	count := int(din.ReadDecimal())
	for t := 0; t < count && count < MAX_CHILD; t++ {
		key := din.ReadText()
		value := val.ReadValue(din)
		this.table.Put(key, value)
	}
}
func (this *ParamPack) ToResponse() *ParamPack {
	if this.Request == 0 {
		return this
	}
	this.Response = this.Request
	this.Request = 0
	return this
}
func (this *ParamPack) ToString() string {
	var buffer bytes.Buffer
	buffer.WriteString("ParamPack")
	buffer.WriteString("\nId=")
	buffer.WriteString(fmt.Sprintf("%d", this.Id))
	buffer.WriteString("\nRequest=")
	buffer.WriteString(fmt.Sprintf("%d", this.Request))
	buffer.WriteString("\nResponse=")
	buffer.WriteString(fmt.Sprintf("%d", this.Response))
	buffer.WriteString("\n")
	buffer.WriteString(this.table.ToString())
	return buffer.String()
}
