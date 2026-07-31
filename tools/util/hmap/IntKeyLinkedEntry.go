package hmap

import (
	"fmt"
)

type IntKeyLinkedEntry struct {
	key       int32
	keyHash   uint
	value     interface{}
	next      *IntKeyLinkedEntry
	link_next *IntKeyLinkedEntry
	link_prev *IntKeyLinkedEntry
}

func NewIntKeyLinkedEntry(key int32, value interface{}, next *IntKeyLinkedEntry) *IntKeyLinkedEntry {
	p := new(IntKeyLinkedEntry)
	p.key = key
	p.value = value
	p.next = next

	return p
}

func (this *IntKeyLinkedEntry) GetKey() int32 {
	return this.key
}
func (this *IntKeyLinkedEntry) GetValue() interface{} {
	return this.value
}
func (this *IntKeyLinkedEntry) SetValue(v interface{}) interface{} {
	old := this.value
	this.value = v
	return old
}
func (this *IntKeyLinkedEntry) Equals(o *IntKeyLinkedEntry) bool {
	return this.key == o.key
}

func (this *IntKeyLinkedEntry) HashCode() uint {
	// key is int32; the Java-derived `key ^ (key >>> 32)` fold is meaningless
	// for 32-bit keys (and flagged by go vet) — use the key directly.
	return uint(uint32(this.key))
}

func (this *IntKeyLinkedEntry) ToString() string {
	return fmt.Sprintf("%d=%v", this.key, this.value)
}
