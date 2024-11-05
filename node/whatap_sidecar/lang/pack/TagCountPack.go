package pack

import (
	"fmt"

	"github.com/whatap/golib/io"
	"github.com/whatap/golib/lang/value"
	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/hash"
)

type TagCountPack struct {
	AbstractPack
	Category string
	tagHash  int64
	Tags     *value.MapValue
	Fields   *value.MapValue
}

func NewTagCountPack() *TagCountPack {
	p := new(TagCountPack)
	p.Tags = value.NewMapValue()
	p.Fields = value.NewMapValue()
	return p
}

func (this *TagCountPack) GetPackType() int16 {
	return TAG_COUNT
}

func (this *TagCountPack) ToString() string {
	return fmt.Sprint(this.AbstractPack.ToString(), ",category=", this.Category, ",tags=", this.Tags.ToString(), ",fields=", this.Fields.ToString())
}

func (this *TagCountPack) GetTag(name string) string {
	return this.Tags.GetString(name)
}
func (this *TagCountPack) PutTag(name, val string) {
	this.Tags.PutString(name, val)
}
func (this *TagCountPack) PutTagLong(name string, val int64) {
	this.Tags.PutLong(name, val)
}
func (this *TagCountPack) Put(name string, v interface{}) {
	switch v.(type) {
	case value.Value:
		this.Fields.Put(name, v.(value.Value))
	case bool:
		this.Fields.Put(name, value.NewBoolValue(v.(bool)))
	case []byte:
		this.Fields.Put(name, value.NewTextValue(string(v.([]byte))))
	case int:
		this.Fields.Put(name, value.NewDecimalValue(int64(v.(int))))
	case int32:
		this.Fields.Put(name, value.NewDecimalValue(int64(v.(int32))))
	case uint32:
		this.Fields.Put(name, value.NewDecimalValue(int64(v.(uint32))))
	case int64:
		this.Fields.Put(name, value.NewDecimalValue(v.(int64)))
	case uint64:
		this.Fields.Put(name, value.NewDecimalValue(int64(v.(uint64))))
	case float32:
		this.Fields.Put(name, value.NewFloatValue(v.(float32)))
	case float64:
		this.Fields.Put(name, value.NewDoubleValue(v.(float64)))
	case string:
		this.Fields.Put(name, value.NewTextValue(v.(string)))
	default:
		panic(fmt.Sprintf("Panic, Not supported type %T. available type: value.Value, int, int32, int64, float32, float64, string ", v))
	}
}

func (this *TagCountPack) Get(name string) value.Value {
	return this.Fields.Get(name)
}

func (this *TagCountPack) GetFloat(name string) float64 {
	val := this.Fields.Get(name)
	if val == nil {
		return 0
	}

	switch val.GetValueType() {
	case value.VALUE_DOUBLE_SUMMARY, value.VALUE_LONG_SUMMARY:
		return (val.(value.SummaryValue)).DoubleAvg()
	case value.VALUE_DECIMAL:
		return float64((val.(*value.DecimalValue)).Val)
	case value.VALUE_DECIMAL_INT:
		return float64((val.(*value.IntValue)).Val)
	case value.VALUE_DECIMAL_LONG:
		return float64((val.(*value.LongValue)).Val)
	case value.VALUE_FLOAT:
		return float64((val.(*value.FloatValue)).Val)
	case value.VALUE_DOUBLE:
		return float64((val.(*value.DoubleValue)).Val)
	default:
	}

	return 0
}

func (this *TagCountPack) GetLong(name string) int64 {
	val := this.Fields.Get(name)
	if val == nil {
		return 0
	}

	switch val.GetValueType() {
	case value.VALUE_DOUBLE_SUMMARY, value.VALUE_LONG_SUMMARY:
		return (val.(value.SummaryValue)).LongAvg()
	case value.VALUE_DECIMAL:
		return int64((val.(*value.DecimalValue)).Val)
	case value.VALUE_DECIMAL_INT:
		return int64((val.(*value.IntValue)).Val)
	case value.VALUE_DECIMAL_LONG:
		return int64((val.(*value.LongValue)).Val)
	case value.VALUE_FLOAT:
		return int64((val.(*value.FloatValue)).Val)
	case value.VALUE_DOUBLE:
		return int64((val.(*value.DoubleValue)).Val)
	default:

		//	default:
		//						if (val instanceof Number) {
		//							return ((Number) val).longValue();
		//						}

	}
	return 0
}

func (this *TagCountPack) Write(dout *io.DataOutputX) {
	this.AbstractPack.Write(dout)
	dout.WriteByte(0)
	dout.WriteText(this.Category)
	if this.tagHash == 0 && this.Tags.Size() > 0 {
		tagIO := io.NewDataOutputX()
		value.WriteValue(tagIO, this.Tags)
		tagBytes := tagIO.ToByteArray()
		this.tagHash = hash.Hash64(tagBytes)
		dout.WriteDecimal(this.tagHash)
		dout.WriteBytes(tagBytes)
	} else {
		dout.WriteDecimal(this.tagHash)
		value.WriteValue(dout, this.Tags)
	}
	value.WriteValue(dout, this.Fields)
}

func (this *TagCountPack) Read(din *io.DataInputX) {
	this.AbstractPack.Read(din)
	//ver := din.ReadByte()
	din.ReadByte()
	this.Category = din.ReadText()
	this.tagHash = din.ReadDecimal()
	this.Tags = value.ReadValue(din).(*value.MapValue)
	this.Fields = value.ReadValue(din).(*value.MapValue)
}

func (this *TagCountPack) IsEmpty() bool {
	return this.Fields.IsEmpty()
}
func (this *TagCountPack) Size() int {
	return this.Fields.Size()
}

func (this *TagCountPack) Clear() {
	this.Fields.Clear()
}
