package queue

import (
	"sync"

	"gitlab.whatap.io/hsnam/focus-agent/whatap/util/list"
)

type RequestDoubleQueue struct {
	queue1      list.LinkedList
	queue2      list.LinkedList
	capacity1   int
	capacity2   int
	lock        *sync.Cond
	failed1     func(interface{})
	overflowed1 func(interface{})
	failed2     func(interface{})
	overflowed2 func(interface{})
}

func NewRequestDoubleQueue(size1 int, size2 int) *RequestDoubleQueue {
	q := new(RequestDoubleQueue)
	q.lock = sync.NewCond(new(sync.Mutex))
	q.capacity1 = size1
	q.capacity2 = size2
	return q
}
func (this *RequestDoubleQueue) Get() interface{} {
	this.lock.L.Lock()
	defer this.lock.L.Unlock()

	for this.queue1.Size() <= 0 && this.queue2.Size() <= 0 {
		this.lock.Wait()
	}
	if this.queue1.Size() > 0 {
		return this.queue1.RemoveFirst()
	}
	if this.queue2.Size() > 0 {
		return this.queue2.RemoveFirst()
	}
	return nil /*impossible*/
}
func (this *RequestDoubleQueue) Put1(v interface{}) bool {
	this.lock.L.Lock()
	defer this.lock.L.Unlock()

	if this.capacity1 <= 0 || this.queue1.Size() < this.capacity1 {
		this.queue1.Add(v)
		this.lock.Broadcast()
		return true
	} else {
		if this.failed1 != nil {
			this.failed1(v)
		}
		this.lock.Broadcast()
		return false
	}
}
func (this *RequestDoubleQueue) Put2(v interface{}) bool {
	this.lock.L.Lock()
	defer this.lock.L.Unlock()

	if this.capacity2 <= 0 || this.queue2.Size() < this.capacity2 {
		this.queue2.Add(v)
		this.lock.Broadcast()
		return true
	} else {
		if this.failed2 != nil {
			this.failed2(v)
		}
		this.lock.Broadcast()
		return false
	}

}
func (this *RequestDoubleQueue) PutForce1(v interface{}) bool {
	this.lock.L.Lock()
	defer this.lock.L.Unlock()

	if this.capacity1 <= 0 || this.queue1.Size() < this.capacity1 {
		this.queue1.Add(v)
		this.lock.Broadcast()
		return true
	} else {
		for this.queue1.Size() >= this.capacity1 {
			o := this.queue1.RemoveFirst()
			if this.overflowed1 != nil {
				this.overflowed1(o)
			}
		}
		this.queue1.Add(v)
		this.lock.Broadcast()
		return false
	}
}
func (this *RequestDoubleQueue) PutForce2(v interface{}) bool {
	this.lock.L.Lock()
	defer this.lock.L.Unlock()

	if this.capacity2 <= 0 || this.queue2.Size() < this.capacity2 {
		this.queue2.Add(v)
		this.lock.Broadcast()
		return true
	} else {
		for this.queue2.Size() >= this.capacity2 {
			o := this.queue2.RemoveFirst()
			if this.overflowed2 != nil {
				this.overflowed2(o)
			}
		}
		this.queue2.Add(v)
		this.lock.Broadcast()
		return false
	}
}
func (this *RequestDoubleQueue) Clear() {
	this.lock.L.Lock()
	defer this.lock.L.Unlock()
	this.queue1.Clear()
	this.queue2.Clear()
}
func (this *RequestDoubleQueue) Size() int {
	return this.queue1.Size() + this.queue2.Size()
}
func (this *RequestDoubleQueue) Size1() int {
	return this.queue1.Size()
}
func (this *RequestDoubleQueue) Size2() int {
	return this.queue2.Size()
}

func (this *RequestDoubleQueue) GetCapacity1() int {
	return this.capacity1
}
func (this *RequestDoubleQueue) GetCapacity2() int {
	return this.capacity2
}

func (this *RequestDoubleQueue) SetCapacity(sz1 int, sz2 int) {
	this.capacity1 = sz1
	this.capacity2 = sz2
}
