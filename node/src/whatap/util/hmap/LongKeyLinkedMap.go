package hmap

import (
	"fmt"
	"math"
	"sort"
	"sync"
	// "io"

	//GetKeySet
	"container/list"

	"github.com/whatap/golib/util/panicutil"
	"github.com/whatap/kube/node/src/whatap/util/stringutil"
)

// LongKeyLinkedMap LongKeyLinkedMap
type LongKeyLinkedMap struct {
	table      []*LongKeyLinkedEntry
	header     *LongKeyLinkedEntry
	count      int
	threshold  int
	loadFactor float32
	max        int
	lock       sync.Mutex
}

// NewLongKeyLinkedMapDefault NewLongKeyLinkedMapDefault
func NewLongKeyLinkedMapDefault() *LongKeyLinkedMap {
	p := NewLongKeyLinkedMap(DEFAULT_CAPACITY, DEFAULT_LOAD_FACTOR)
	return p
}

// NewLongKeyLinkedMap NewLongKeyLinkedMap
func NewLongKeyLinkedMap(initCapacity int, loadFactor float32) *LongKeyLinkedMap {
	defer func() {
		if r := recover(); r != nil {
			panicutil.Debug("WA822", r)
			//return NewIntKeyMap(DEFAULT_CAPACITY, DEFAULT_LOAD_FACTOR)
		}
	}()

	p := new(LongKeyLinkedMap)

	if initCapacity < 0 {
		panic(fmt.Sprintf("Capacity Error: %d", initCapacity))
		//throw new RuntimeException("Capacity Error: " + initCapacity);
	}
	if loadFactor <= 0 {
		panic(fmt.Sprintf("Load Count Error: %d", loadFactor))
		//throw new RuntimeException("Load Count Error: " + loadFactor);
	}
	if initCapacity == 0 {
		initCapacity = 1
	}
	p.loadFactor = loadFactor
	p.table = make([]*LongKeyLinkedEntry, initCapacity)
	p.header = NewLongKeyLinkedEntry(0, nil, nil)
	p.header.link_prev = p.header
	p.header.link_next = p.header.link_prev

	p.threshold = int(float32(initCapacity) * loadFactor)

	return p
}

// Size Size
func (this *LongKeyLinkedMap) Size() int {
	return this.count
}

// KeyArray KeyArray
func (this *LongKeyLinkedMap) KeyArray() []int64 {
	_keys := make([]int64, this.Size())

	en := this.Keys()
	for i := 0; i < len(_keys); i++ {
		_keys[i] = en.NextLong()
	}
	return _keys
}

// GetKeySet GetKeySet
func (this *LongKeyLinkedMap) GetKeySet() *LongLinkedSet {
	_keys := NewLongLinkedSet()
	en := this.Keys()
	for en.HasMoreElements() {
		_keys.Put(en.NextLong())
	}
	return _keys
}

// Keys Keys
func (this *LongKeyLinkedMap) Keys() LongEnumer {
	this.lock.Lock()
	defer this.lock.Unlock()

	//return &LongKeyEnumerImpl{parent: this, entry: this.header.link_next}
	return NewLongKeyLinkedEnumer(this, this.header.link_next, ELEMENT_TYPE_KEYS)
}

// Values Values
func (this *LongKeyLinkedMap) Values() Enumeration {
	this.lock.Lock()
	defer this.lock.Unlock()
	return NewLongKeyLinkedEnumer(this, this.header.link_next, ELEMENT_TYPE_VALUES)
}

// ValueIterator ValueIterator
func (this *LongKeyLinkedMap) ValueIterator() interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	return NewLongKeyLinkedEnumer(this, this.header.link_next, ELEMENT_TYPE_VALUES)
}

// Entries Entries
func (this *LongKeyLinkedMap) Entries() Enumeration {
	this.lock.Lock()
	defer this.lock.Unlock()
	return NewLongKeyLinkedEnumer(this, this.header.link_next, ELEMENT_TYPE_ENTRIES)
}

// ContainsValue ContainsValue
func (this *LongKeyLinkedMap) ContainsValue(value interface{}) bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	if value == nil {
		panic("Value is Nil")
		//throw new NullPointerException();
	}
	tab := this.table

	for i := len(tab); i > 0; i-- {
		for e := tab[i]; e != nil; e = e.next {

			// TODO COmpareUtil
			//				if (CompareUtil.equals(e.value, value)) {
			//					return true;
			//				}
			if e.value == value {
				return true
			}
		}
	}
	return false
}

// ContainsKey ContainsKey
func (this *LongKeyLinkedMap) ContainsKey(key int64) bool {
	this.lock.Lock()
	defer this.lock.Unlock()

	tab := this.table
	index := this.hash(key) % uint(len(tab))
	for e := tab[index]; e != nil; e = e.next {
		if e.key == key {
			return true
		}
	}
	return false
}

// Get Get
func (this *LongKeyLinkedMap) Get(key int64) interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	tab := this.table
	index := this.hash(key) % uint(len(tab))
	for e := tab[index]; e != nil; e = e.next {
		//if (CompareUtil.equals(e.key, key)) {
		if e.key == key {
			return e.value
		}
	}
	return nil
}

// GetLRU GetLRU
func (this *LongKeyLinkedMap) GetLRU(key int64) interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	tab := this.table
	index := this.hash(key) % uint(len(tab))

	for e := tab[index]; e != nil; e = e.next {
		if e.key == key {
			old := e.value
			if this.header.link_prev != e {
				this.unchain(e)
				this.chain(this.header.link_prev, this.header, e)
			}
			return old
		}
	}
	return nil
}

// GetFirstKey GetFirstKey
func (this *LongKeyLinkedMap) GetFirstKey() int64 {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.header.link_next.key
}

// GetLastKey GetLastKey
func (this *LongKeyLinkedMap) GetLastKey() int64 {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.header.link_prev.key
}

// GetFirstValue GetFirstValue
func (this *LongKeyLinkedMap) GetFirstValue() interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.header.link_next.value
}

// GetLastValue GetLastValue
func (this *LongKeyLinkedMap) GetLastValue() interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	return this.header.link_prev.value
}

func (this *LongKeyLinkedMap) overflowed(key int64, value interface{}) {
}

func (this *LongKeyLinkedMap) create(key int64) interface{} {
	panic("not implemented create()")
	//throw new RuntimeException("not implemented create()");
}

func (this *LongKeyLinkedMap) Intern(key int64) interface{} {
	return this._intern(key, PUT_LAST)
}

func (this *LongKeyLinkedMap) _intern(key int64, m PUT_MODE) interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()
	tab := this.table
	index := this.hash(key) % uint(len(tab))

	for e := tab[index]; e != nil; e = e.next {
		//if (CompareUtil.equals(e.key, key)) {
		if e.key == key {
			return e.value
		}
	}
	value := this.create(key)
	if value == nil {
		return nil
	}
	if this.max > 0 {
		switch m {
		case PUT_FORCE_FIRST:
			fallthrough
		case PUT_FIRST:
			for this.count >= this.max {
				k := this.header.link_prev.key
				v := this.remove(k)
				this.overflowed(k, v)
			}
		case PUT_FORCE_LAST:
			fallthrough
		case PUT_LAST:
			for this.count >= this.max {
				k := this.header.link_next.key
				v := this.remove(k)
				this.overflowed(k, v)
			}
		}
	}
	if this.count >= this.threshold {
		this.rehash()
		tab = this.table
		index = this.hash(key) % uint(len(tab))
	}

	e := NewLongKeyLinkedEntry(key, value, tab[index])
	tab[index] = e
	switch m {
	case PUT_FORCE_FIRST:
		fallthrough
	case PUT_FIRST:
		this.chain(this.header, this.header.link_next, e)
	case PUT_FORCE_LAST:
	case PUT_LAST:
		this.chain(this.header.link_prev, this.header, e)
	}
	this.count++
	return value
}

func (this *LongKeyLinkedMap) hash(key int64) uint {
	return uint(key & math.MaxInt32)
}

func (this *LongKeyLinkedMap) rehash() {
	oldCapacity := len(this.table)
	oldMap := this.table
	newCapacity := oldCapacity*2 + 1
	newMap := make([]*LongKeyLinkedEntry, newCapacity)
	this.threshold = int(float32(newCapacity) * this.loadFactor)
	this.table = newMap
	for i := oldCapacity; i > 0; i-- {
		old := oldMap[i-1]
		for old != nil {
			e := old
			old = old.next
			key := e.key
			index := this.hash(key) % uint(newCapacity)
			e.next = newMap[index]
			newMap[index] = e
		}
	}
}

// SetMax SetMax
func (this *LongKeyLinkedMap) SetMax(max int) *LongKeyLinkedMap {
	this.max = max
	return this
}

// IsFull IsFull
func (this *LongKeyLinkedMap) IsFull() bool {
	return this.max > 0 && this.max <= this.count
}

// Put Put
func (this *LongKeyLinkedMap) Put(key int64, value interface{}) interface{} {
	return this.put(key, value, PUT_LAST)
}

// PutLast PutLast
func (this *LongKeyLinkedMap) PutLast(key int64, value interface{}) interface{} {
	return this.put(key, value, PUT_FORCE_LAST)
}

// PutFirst PutFirst
func (this *LongKeyLinkedMap) PutFirst(key int64, value interface{}) interface{} {
	return this.put(key, value, PUT_FORCE_FIRST)
}

func (this *LongKeyLinkedMap) put(key int64, value interface{}, m PUT_MODE) interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()

	tab := this.table
	index := this.hash(key) % uint(len(tab))

	for e := tab[index]; e != nil; e = e.next {
		//if (CompareUtil.equals(e.key, key)) {
		if e.key == key {
			old := e.value
			e.value = value
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
			return old
		}
	}

	if this.max > 0 {
		switch m {
		case PUT_FORCE_FIRST:
			fallthrough

		case PUT_FIRST:
			for this.count >= this.max {
				// removeLast();
				k := this.header.link_prev.key
				v := this.remove(k)
				this.overflowed(k, v)
			}

		case PUT_FORCE_LAST:
			fallthrough
		case PUT_LAST:
			for this.count >= this.max {
				// removeFirst();
				k := this.header.link_next.key
				v := this.remove(k)
				this.overflowed(k, v)
			}

		}
	}
	if this.count >= this.threshold {
		this.rehash()
		tab = this.table
		index = this.hash(key) % uint(len(tab))
	}

	e := NewLongKeyLinkedEntry(key, value, tab[index])
	tab[index] = e
	switch m {
	case PUT_FORCE_FIRST:
		fallthrough
	case PUT_FIRST:
		this.chain(this.header, this.header.link_next, e)
	case PUT_FORCE_LAST:
	case PUT_LAST:
		this.chain(this.header.link_prev, this.header, e)
	}
	this.count++
	return nil
}
func (this *LongKeyLinkedMap) remove(key int64) interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()

	tab := this.table
	index := this.hash(key) % uint(len(tab))
	var prev *LongKeyLinkedEntry
	prev = nil

	for e := tab[index]; e != nil; e = e.next {
		if e.key == key {
			if prev != nil {
				prev.next = e.next
			} else {
				tab[index] = e.next
			}
			this.count--
			oldValue := e.value
			e.value = nil
			this.unchain(e)
			return oldValue
		}

		prev = e
	}

	return nil
}

// Remove Remove
func (this *LongKeyLinkedMap) Remove(key int64) interface{} {
	return this.remove(key)
}

// RemoveFirst RemoveFirst
func (this *LongKeyLinkedMap) RemoveFirst() interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.IsEmpty() {
		return nil
	}
	return this.remove(this.header.link_next.key)
}

// RemoveLast RemoveLast
func (this *LongKeyLinkedMap) RemoveLast() interface{} {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.IsEmpty() {
		return nil
	}

	return this.remove(this.header.link_prev.key)
}

// IsEmpty IsEmpty
func (this *LongKeyLinkedMap) IsEmpty() bool {
	return this.Size() == 0
}

// Clear Clear
func (this *LongKeyLinkedMap) Clear() {
	this.lock.Lock()
	defer this.lock.Unlock()

	tab := this.table
	//index := this.hash(key) % uint(len(tab))
	for index := len(tab) - 1; index >= 0; index-- {
		tab[index] = nil
	}

	this.header.link_prev = this.header
	this.header.link_next = this.header.link_prev
	this.count = 0
}

// ToString ToString
func (this *LongKeyLinkedMap) ToString() string {
	buf := stringutil.NewStringBuffer()
	it := this.Entries()
	buf.Append("{")
	for i := 0; it.HasMoreElements(); i++ {
		e := it.NextElement().(*LongKeyLinkedEntry)
		if i > 0 {
			buf.Append(", ")
		}
		buf.Append(fmt.Sprintf("%d=%v", e.GetKey(), e.GetValue()))
	}
	buf.Append("}")
	return buf.ToString()
}

// ToFormatString ToFormatString
func (this *LongKeyLinkedMap) ToFormatString() string {
	buf := stringutil.NewStringBuffer()
	it := this.Entries()
	buf.Append("{")
	for i := 0; it.HasMoreElements(); i++ {
		e := it.NextElement().(*LongKeyLinkedEntry)
		if i > 0 {
			buf.Append(", ")
		}
		buf.Append(fmt.Sprintf("%d=%v", e.GetKey(), e.GetValue())).Append("\n")

	}
	buf.Append("}")
	return buf.ToString()
}

func (this *LongKeyLinkedMap) chain(link_prev *LongKeyLinkedEntry, link_next *LongKeyLinkedEntry, e *LongKeyLinkedEntry) {
	e.link_prev = link_prev
	e.link_next = link_next
	link_prev.link_next = e
	link_next.link_prev = e
}

func (this *LongKeyLinkedMap) unchain(e *LongKeyLinkedEntry) {
	e.link_prev.link_next = e.link_next
	e.link_next.link_prev = e.link_prev
	e.link_prev = nil
	e.link_next = nil
}

// ToKeySet java hashset 을 list  변환해서 반환
func (this *LongKeyLinkedMap) ToKeySet() *list.List {

	keyList := list.New()
	en := this.Keys()

	for en.HasMoreElements() {

		keyList.PushFront(en.NextLong())

	}
	return keyList
}

// Sort Sort
func (this *LongKeyLinkedMap) Sort(c func(k1, k2 int64) bool) {
	this.lock.Lock()
	defer this.lock.Unlock()

	sz := this.Size()
	entryList := make([]*LongKeyLinkedEntry, sz)
	en := this.Entries()
	for i := 0; i < sz; i++ {
		entryList[i] = en.NextElement().(*LongKeyLinkedEntry)
	}
	sort.Sort(LongKeySortable{compare: c, data: entryList})
	this.Clear()
	for i := 0; i < sz; i++ {
		this.put(entryList[i].GetKey(), entryList[i].GetValue(), PUT_LAST)
	}
}

// LongKeyLinkedEnumer LongKeyLinkedEnumer
type LongKeyLinkedEnumer struct {
	parent *LongKeyLinkedMap
	entry  *LongKeyLinkedEntry
	Type   int
}

// NewLongKeyLinkedEnumer NewLongKeyLinkedEnumer
func NewLongKeyLinkedEnumer(parent *LongKeyLinkedMap, entry *LongKeyLinkedEntry, Type int) *LongKeyLinkedEnumer {
	p := new(LongKeyLinkedEnumer)
	p.parent = parent
	p.entry = entry
	p.Type = Type

	return p
}

// HasNext HasNext
func (this *LongKeyLinkedEnumer) HasNext() bool {
	return this.HasMoreElements()
}

// HasMoreElements HasMoreElements
func (this *LongKeyLinkedEnumer) HasMoreElements() bool {
	return this.parent.header != this.entry && this.entry != nil
}

// Next Next
func (this *LongKeyLinkedEnumer) Next() interface{} {
	return this.NextElement()
}

// NextElement NextElement
func (this *LongKeyLinkedEnumer) NextElement() interface{} {
	if this.HasMoreElements() {
		e := this.entry
		this.entry = e.link_next

		switch this.Type {
		case ELEMENT_TYPE_KEYS:
			//return (V) new Long(e.key);
			return e.key
		case ELEMENT_TYPE_VALUES:
			return e.value
		default:
			return e
		}
	}
	panic("no more next")
	//throw new NoSuchElementException("no more next");
}

// NextLong NextLong
func (this *LongKeyLinkedEnumer) NextLong() int64 {
	if this.HasMoreElements() {
		e := this.entry
		this.entry = e.link_next
		return e.key
	}
	panic("no more next")
	//throw new NoSuchElementException("no more next");
}

// Remove Remove
func (this *LongKeyLinkedEnumer) Remove() {

}

// LongKeySortable implements sort.Interface
type LongKeySortable struct {
	// func(a, b, int64) bool
	compare func(a, b int64) bool
	// []*LongKeyLinkedEntry
	data []*LongKeyLinkedEntry
}

// Len Len
func (this LongKeySortable) Len() int {
	return len(this.data)
}

// Less Less
func (this LongKeySortable) Less(i, j int) bool {
	return this.compare(this.data[i].GetKey(), this.data[j].GetKey())
}

// Swap Swap
func (this LongKeySortable) Swap(i, j int) {
	this.data[i], this.data[j] = this.data[j], this.data[i]
}

//func main() {
//		m = NewLongKeyLinkedMapDefault.setMax(5);
//		// System.out.println(m.getFirstValue());
//		// System.out.println(m.getLastKey());
//		for i := 0; i < 10; i++ {
//			m.putFirst(i, i);
//		}
////		for (int i = 1; i < 10; i+=2) {
////			m.put(i, i);
////		}
////		m.sort(new Comparator<LongKeyLinkedMap.LongKeyLinkedEntry<Integer>>() {
////			public int compare(LongKeyLinkedEntry<Integer> o1, LongKeyLinkedEntry<Integer> o2) {
////				return o1.key - o2.key;
////			}
////		});
//		fmt.Println(m);
//			// System.out.println("==================================");
//		// for(int i=0; i <10; i++){
//		// m.putLast(i, i);
//		// System.out.println(m);
//		// }
//		// System.out.println("==================================");
//		// for(int i=0; i <10; i++){
//		// m.putFirst(i, i);
//		// System.out.println(m);
//		// }
////		IntEnumer e = m.keys();
////		System.out.println("==================================");
////		for (int i = 0; i < 10; i++) {
////			m.removeFirst();
////			System.out.println(m);
////		}
////		System.out.println("==================================");
////		while (e.hasMoreElements()) {
////			System.out.println(e.nextInt());
////		}
//	}

//	private static void print(Object e) {
//		System.out.println(e);
//	}
