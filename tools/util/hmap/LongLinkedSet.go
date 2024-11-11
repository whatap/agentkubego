package hmap

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
)
//LongLinkedSet LongLinkedSet
type LongLinkedSet struct {
	table      []*LongLinkedEntry
	header     *LongLinkedEntry
	count      int
	threshold  int
	loadFactor float32
	lock       sync.Mutex
	max        int
}

//NewLongLinkedSet NewLongLinkedSet
func NewLongLinkedSet() *LongLinkedSet {

	initCapacity := DEFAULT_CAPACITY
	loadFactor := DEFAULT_LOAD_FACTOR

	this := new(LongLinkedSet)
	this.loadFactor = float32(loadFactor)
	this.table = make([]*LongLinkedEntry, initCapacity)
	this.header = &LongLinkedEntry{}
	this.header.link_next = this.header
	this.header.link_prev = this.header
	this.threshold = (int)(float64(initCapacity) * loadFactor)
	return this
}

//Size Size
func (this *LongLinkedSet) Size() int {
	return this.count
}

//KeyArray KeyArray
func (this *LongLinkedSet) KeyArray() []int64 {
	this.lock.Lock()
	defer this.lock.Unlock()

	_keys := make([]int64, this.Size())
	en := this.Keys()
	for i := 0; i < len(_keys); i++ {
		_keys[i] = en.NextLong()
	}
	return _keys
}

//LongEnumerSetImpl LongEnumerSetImpl
type LongEnumerSetImpl struct {
	parent *LongLinkedSet
	entry  *LongLinkedEntry
	rtype  int
}

//HasMoreElements HasMoreElements
func (this *LongEnumerSetImpl) HasMoreElements() bool {
	return this.entry != nil && this.parent.header != this.entry
}

//NextLong NextLong
func (this *LongEnumerSetImpl) NextLong() int64 {
	if this.HasMoreElements() {
		e := this.entry
		this.entry = e.link_next
		return e.Get()
	}
	return 0
}

//Keys Keys
func (this *LongLinkedSet) Keys() LongEnumer {
	return &LongEnumerSetImpl{parent: this, entry: this.header.link_next}
}

//Contains Contains
func (this *LongLinkedSet) Contains(key int64) bool {
	this.lock.Lock()
	defer this.lock.Unlock()

	tab := this.table
	index := this.hash(key) % uint(len(tab))
	for e := tab[index]; e != nil; e = e.hash_next {
		if e.key==key {
			return true
		}
	}
	return false

}

//GetFirst GetFirst
func (this *LongLinkedSet) GetFirst() int64 {
	this.lock.Lock()
	defer this.lock.Unlock()

	return this.header.link_next.key
}

//GetLast GetLast
func (this *LongLinkedSet) GetLast() int64 {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.header.link_prev.key
}
func (this *LongLinkedSet) hash(key int64) uint {
	return uint(key)
}

func (this *LongLinkedSet) rehash() {
	oldCapacity := len(this.table)
	oldMap := this.table
	newCapacity := oldCapacity*2 + 1
	newMap := make([]*LongLinkedEntry, newCapacity)
	this.threshold = int(float32(newCapacity) * this.loadFactor)
	this.table = newMap
	for i := oldCapacity; i > 0; i-- {
		for old := oldMap[i-1]; old != nil; {
			e := old
			old = old.hash_next
			index := uint(this.hash(e.key) % uint(newCapacity))
			e.hash_next = newMap[index]
			newMap[index] = e
		}
	}
}

//SetMax SetMax
func (this *LongLinkedSet) SetMax(max int) *LongLinkedSet {
	this.max = max
	return this
}
//Put Put
func (this *LongLinkedSet) Put(key int64) interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.put(key, PUT_LAST)
}
//PutLast PutLast
func (this *LongLinkedSet) PutLast(key int64) interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.put(key, PUT_FORCE_LAST)
}
//PutFirst PutFirst
func (this *LongLinkedSet) PutFirst(key int64) interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.put(key, PUT_FORCE_FIRST)
}

func (this *LongLinkedSet) put(key int64, m PUT_MODE) interface{} {
	tab := this.table
	keyHash := this.hash(key)
	index := keyHash % uint(len(tab))
	for e := tab[index]; e != nil; e = e.hash_next {
		if e.key==key {
			switch m {
			case PUT_FORCE_FIRST:
				if this.header.link_next != e {
					this.unchain(e)
					this.chain(this.header, this.header.link_next, e)
				}
			case PUT_FORCE_LAST:
				if this.header.link_prev != e {
					this.unchain(e)
					this.chain(this.header.link_prev, this.header, e)
				}
			}
			return key
		}
	}
	if this.max > 0 {
		switch m {
		case PUT_FORCE_FIRST, PUT_FIRST:
			for this.count >= this.max {
				k := this.header.link_prev.key
				this.remove(k)
			}
		case PUT_FORCE_LAST, PUT_LAST:
			for this.count >= this.max {
				k := this.header.link_next.key
				this.remove(k)
			}
			break
		}
	}
	if this.count >= this.threshold {
		this.rehash()
		tab = this.table
		index = keyHash % uint(len(tab))
	}
	e := &LongLinkedEntry{key: key,  hash_next: tab[index]}
	tab[index] = e
	switch m {
	case PUT_FORCE_FIRST, PUT_FIRST:
		this.chain(this.header, this.header.link_next, e)
	case PUT_FORCE_LAST, PUT_LAST:
		this.chain(this.header.link_prev, this.header, e)
	}
	this.count++
	return nil
}
//Remove Remove
func (this *LongLinkedSet) Remove(key int64) interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()

	return this.remove(key)
}
//RemoveFirst RemoveFirst
func (this *LongLinkedSet) RemoveFirst() interface{} {
	if this.IsEmpty() {
		return 0
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.remove(this.header.link_next.key)
}
//RemoveLast RemoveLast
func (this *LongLinkedSet) RemoveLast() interface{} {
	if this.IsEmpty() {
		return 0
	}
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.remove(this.header.link_prev.key)
}

func (this *LongLinkedSet) remove(key int64) interface{} {

	tab := this.table
	index := this.hash(key) % uint(len(tab))
	e := tab[index]
	var prev *LongLinkedEntry = nil
	for e != nil {
		if e.key==key {
			if prev != nil {
				prev.hash_next = e.hash_next
			} else {
				tab[index] = e.hash_next
			}
			this.count--
			//
			this.unchain(e)
			return key
		}
		prev = e
		e = e.hash_next
	}
	return nil
}
//IsEmpty IsEmpty
func (this *LongLinkedSet) IsEmpty() bool {
	return this.count == 0
}
//IsFull IsFull
func (this *LongLinkedSet) IsFull() bool {
	return this.max > 0 && this.max <= this.count
}
//Clear Clear
func (this *LongLinkedSet) Clear() {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.clear()
}
func (this *LongLinkedSet) clear() {
	tab := this.table
	for index := len(tab) - 1; index >= 0; index-- {
		tab[index] = nil
	}
	this.header.link_next = this.header
	this.header.link_prev = this.header
	this.count = 0
}

func (this *LongLinkedSet) chain(link_prev *LongLinkedEntry, link_next *LongLinkedEntry, e *LongLinkedEntry) {
	e.link_prev = link_prev
	e.link_next = link_next
	link_prev.link_next = e
	link_next.link_prev = e
}

func (this *LongLinkedSet) unchain(e *LongLinkedEntry) {
	e.link_prev.link_next = e.link_next
	e.link_next.link_prev = e.link_prev
	e.link_prev = nil
	e.link_next = nil
}
//ToString ToString
func (this *LongLinkedSet) ToString() string {
	this.lock.Lock()
	defer this.lock.Unlock()

	var buffer bytes.Buffer
	x := this.Keys()
	buffer.WriteString("{")
	for i := 0; x.HasMoreElements(); i++ {
		if i > 0 {
			buffer.WriteString(", ")
		}
		e := x.NextLong()
		buffer.WriteString(fmt.Sprintf("%d", e))
	}
	buffer.WriteString("}")
	return buffer.String()
}

type setLongSortable struct {
	compare func(a, b int64) bool
	data    []int64
}
//Len Len
func (this setLongSortable) Len() int {
	return len(this.data)
}
//Less Less
func (this setLongSortable) Less(i, j int) bool {
	return this.compare(this.data[i], this.data[j])
}
//Swap Swap
func (this setLongSortable) Swap(i, j int) {
	this.data[i], this.data[j] = this.data[j], this.data[i]
}
//Sort Sort
func (this *LongLinkedSet) Sort(c func(k1, k2 int64) bool) {
	this.lock.Lock()
	defer this.lock.Unlock()

	sz := this.Size()
	list := make([]int64, sz)
	en := this.Keys()
	for i := 0; i < sz; i++ {
		list[i] = en.NextLong()
	}
	sort.Sort(setLongSortable{compare: c, data: list})

	this.clear()
	for i := 0; i < sz; i++ {
		this.put(list[i], PUT_LAST)
	}
}
