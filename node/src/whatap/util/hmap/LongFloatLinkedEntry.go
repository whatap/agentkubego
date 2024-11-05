package hmap

import (
	"fmt"
	"math"
)

//LongFloatLinkedEntry LongFloatLinkedEntry
type LongFloatLinkedEntry struct {
	key       int64
	value     float32
	hash_next *LongFloatLinkedEntry
	link_next *LongFloatLinkedEntry
	link_prev *LongFloatLinkedEntry
}

//GetKey GetKey
func (this *LongFloatLinkedEntry) GetKey() int64 {
	return this.key
}
//GetValue GetValue
func (this *LongFloatLinkedEntry) GetValue() float32 {
	return this.value
}

//SetValue SetValue
func (this *LongFloatLinkedEntry) SetValue(v float32) float32 {
	old := this.value
	this.value = v
	return old
}

//Equals Equals
func (this *LongFloatLinkedEntry) Equals(o *LongFloatLinkedEntry) bool {
	return this.key == o.key && this.value == o.value
}

//HashCode HashCode
func (this *LongFloatLinkedEntry) HashCode() uint {
	return uint(this.key) ^ uint(math.Float32bits(this.value))
}
//ToString ToString
func (this *LongFloatLinkedEntry) ToString() string {
	return fmt.Sprintf("%d=%f", this.key, this.value)
}
