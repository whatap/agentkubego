package value

import (
	"bytes"
	"fmt"

	"github.com/whatap/kube/node/src/whatap/io"
	"github.com/whatap/kube/node/src/whatap/util/hmap"
)

const (
	MAX_CHILD = 1024
)

type MapValue struct {
	table *hmap.StringKeyLinkedMap
}

func NewMapValue() *MapValue {
	v := new(MapValue)
	v.table = hmap.NewStringKeyLinkedMap()
	return v
}

func (this *MapValue) CompareTo(o Value) int {
	if o == nil {
		return 0
	}
	if o.GetValueType() != this.GetValueType() {
		return int(this.GetValueType() - o.GetValueType())
	}
	that := o.(*MapValue)
	if this.table.Size() != that.table.Size() {
		return this.table.Size() - that.table.Size()
	}
	keys := this.Keys()
	for keys.HasMoreElements() {
		key := keys.NextString()
		v1 := this.table.Get(key).(Value)
		v2 := that.table.Get(key).(Value)
		if v2 == nil {
			return 1
		}
		c := v1.CompareTo(v2)
		if c != 0 {
			return c
		}
	}
	return 0

}

func (this *MapValue) Equals(o Value) bool {
	if o == nil || o.GetValueType() != this.GetValueType() {
		return false
	}
	that := o.(*MapValue)
	if this.table.Size() != that.table.Size() {
		return false
	}
	keys := this.Keys()
	for keys.HasMoreElements() {
		key := keys.NextString()
		v1 := this.table.Get(key).(Value)
		v2 := that.table.Get(key).(Value)
		if v2 == nil {
			return false
		}
		if v1.Equals(v2) == false {
			return false
		}
	}
	return true
}

func (this *MapValue) Get(key string) Value {
	o := this.table.Get(key)
	if o == nil {
		return nil
	}
	return o.(Value)
}
func (this *MapValue) GetString(key string) string {
	o := this.table.Get(key)
	if o != nil && o.(Value).GetValueType() == VALUE_TEXT {
		t := o.(*TextValue)
		return t.Val
	}
	return ""
}
func (this *MapValue) GetBool(key string) bool {
	o := this.table.Get(key)
	if o != nil && o.(Value).GetValueType() == VALUE_BOOLEAN {
		t := o.(*BoolValue)
		return t.Val
	}
	return false
}

func (this *MapValue) GetInt(key string) int64 {
	o := this.table.Get(key)
	if o != nil && o.(Value).GetValueType() == VALUE_DECIMAL_INT {
		t := o.(*IntValue)
		return int64(t.Val)
	} else if o != nil && o.(Value).GetValueType() == VALUE_DECIMAL {
		t := o.(*DecimalValue)
		return t.Val
	}

	return int64(0)
}

func (this *MapValue) GetMap(key string) *MapValue {
	o := this.table.Get(key)
	if val, ok := o.(*MapValue); ok {
		return val
	}
	return nil
}

func (this *MapValue) GetList(key string) *ListValue {
	o := this.table.Get(key)
	if val, ok := o.(*ListValue); ok {
		return val
	}
	return nil
}

func (this *MapValue) GetRaw(key string) interface{} {
	return this.table.Get(key)
}

func (this *MapValue) PutString(key string, value string) {
	this.Put(key, NewTextValue(value))
}
func (this *MapValue) PutLong(key string, value int64) {
	this.Put(key, NewDecimalValue(value))
}
func (this *MapValue) Put(key string, value Value) {
	this.table.Put(key, value)
}
func (this *MapValue) PutList(key string, value ListValue) {
	this.table.Put(key, value)
}
func (this *MapValue) PutMapValue(key string, value *MapValue) {
	this.table.Put(key, value)
}
func (this *MapValue) NewList(key string) *ListValue {
	list := NewListValue(nil)
	this.table.Put(key, list)
	return list
}

func (this *MapValue) PutRaw(name string, v interface{}) {
	switch v.(type) {
	case Value:
		this.Put(name, v.(Value))
	case ListValue:
		this.PutList(name, v.(ListValue))
	case int:
		this.Put(name, NewDecimalValue(int64(v.(int))))
	case int16:
		this.Put(name, NewDecimalValue(int64(v.(int16))))
	case uint16:
		this.Put(name, NewDecimalValue(int64(v.(uint16))))
	case int32:
		this.Put(name, NewDecimalValue(int64(v.(int32))))
	case uint32:
		this.Put(name, NewDecimalValue(int64(v.(uint32))))
	case int64:
		this.Put(name, NewDecimalValue(v.(int64)))
	case uint64:
		this.Put(name, NewDecimalValue(int64(v.(uint64))))
	case float32:
		this.Put(name, NewFloatValue(v.(float32)))
	case float64:
		this.Put(name, NewDoubleValue(v.(float64)))
	case string:
		this.Put(name, NewTextValue(v.(string)))
	case []string:
		vlist := this.NewList(name)
		for _, k := range v.([]string) {
			vlist.Add(NewTextValue(k))
		}
	case bool:
		this.Put(name, NewBoolValue(v.(bool)))
	default:
		fmt.Printf("Panic, Not supported type %T. available type: Value, int, int32, int64, float32, float64, string ", name, v)
		panic(fmt.Sprintf("Panic, Not supported type %T. available type: Value, int, int32, int64, float32, float64, string ", v))
	}
}

func (this *MapValue) Clear() {
	this.Clear()
}
func (this *MapValue) Size() int {
	return this.table.Size()
}
func (this *MapValue) Keys() hmap.StringEnumer {
	return this.table.Keys()
}
func (this *MapValue) GetValueType() byte {
	return VALUE_MAP
}
func (this *MapValue) Write(dout *io.DataOutputX) {
	dout.WriteDecimal(int64(this.table.Size()))
	keys := this.Keys()
	for keys.HasMoreElements() {
		key := keys.NextString()
		v := this.table.Get(key)
		if val, ok := v.(Value); ok {
			dout.WriteText(key)
			WriteValue(dout, val)
		} else if val, ok := v.(MapValue); ok {
			dout.WriteText(key)
			WriteMapValue(dout, &val)
		} else if val, ok := v.(ListValue); ok {
			dout.WriteText(key)
			WriteListValue(dout, &val)
		}
	}
}
func (this *MapValue) Read(din *io.DataInputX) {
	count := int(din.ReadDecimal())
	for t := 0; t < count && count < MAX_CHILD; t++ {
		key := din.ReadText()
		value := ReadValue(din)
		this.table.Put(key, value)
	}
}

func (this *MapValue) ToByte() []byte {
	dout := io.NewDataOutputX()
	this.Write(dout)
	return dout.ToByteArray()
}

func (this *MapValue) ToString() string {
	var buffer bytes.Buffer
	x := this.table.Entries()
	buffer.WriteString("{")
	for i := 0; x.HasMoreElements(); i++ {
		if i > 0 {
			buffer.WriteString(", ")
		}
		e := x.NextElement().(*hmap.StringKeyLinkedEntry)
		buffer.WriteString(e.GetKey())
		buffer.WriteString("=")
		v := e.GetValue()
		if val, ok := v.(Value); ok {
			buffer.WriteString(val.(Value).ToString())
		} else if val, ok := v.(*MapValue); ok {
			buffer.WriteString(val.ToString())
		} else if val, ok := v.(*ListValue); ok {
			buffer.WriteString(val.ToString())
		}

	}
	buffer.WriteString("}")
	return buffer.String()
}
func (this *MapValue) ToFmtString() string {
	var buffer bytes.Buffer
	x := this.table.Entries()
	buffer.WriteString("{")
	for i := 0; x.HasMoreElements(); i++ {
		if i > 0 {
			buffer.WriteString("\n, ")
		}
		e := x.NextElement().(*hmap.StringKeyLinkedEntry)
		buffer.WriteString(e.GetKey())
		buffer.WriteString("=")
		buffer.WriteString(e.GetValue().(Value).ToString())

	}
	buffer.WriteString("}")
	return buffer.String()
}

func (this *MapValue) IsEmpty() bool {
	return this.table.IsEmpty()
}

func (this *MapValue) IterateString(callback func(key string, val string)) {
	x := this.table.Entries()
	for x.HasMoreElements() {
		e := x.NextElement().(*hmap.StringKeyLinkedEntry)

		callback(e.GetKey(), e.GetValue().(Value).ToString())
	}

	return
}
