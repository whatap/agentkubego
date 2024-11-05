package hmap

import (
	"fmt"
)
//LongLinkedEntry LongLinkedEntry
type LongLinkedEntry struct {
	key       int64
	hash_next *LongLinkedEntry
	link_next *LongLinkedEntry
	link_prev *LongLinkedEntry
}
//Get Get
func (this *LongLinkedEntry) Get() int64 {
	return this.key
}
//Equals Equals
func (this *LongLinkedEntry) Equals(o *LongLinkedEntry) bool {
	return this.key == o.key
}
//HashCode HashCode
func (this *LongLinkedEntry) HashCode() uint {
	return uint(this.key)
}
//ToString ToString
func (this *LongLinkedEntry) ToString() string {
	return fmt.Sprintf("%d", this.key)
}
