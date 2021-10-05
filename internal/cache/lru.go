package cache

import (
	"container/list"
	"sync"
	"time"
	"unsafe"

	"github.com/rueian/rueidis/internal/proto"
)

const (
	entrySize   = int(unsafe.Sizeof(entry{})) + int(unsafe.Sizeof(&entry{}))
	elementSize = int(unsafe.Sizeof(list.Element{})) + int(unsafe.Sizeof(&list.Element{}))
	stringSSize = int(unsafe.Sizeof(""))

	EntryMinSize = entrySize + elementSize + stringSSize*2 + proto.MessageStructSize
)

type entry struct {
	val  proto.Message
	ttl  time.Time
	key  string
	ch   chan struct{}
	size int
}

type LRU struct {
	mu sync.Mutex

	store map[string]*list.Element
	list  *list.List

	size int
	max  int
}

func NewLRU(max int) *LRU {
	return &LRU{
		max:   max,
		store: make(map[string]*list.Element),
		list:  list.New(),
	}
}

func (c *LRU) GetOrPrepare(key string, ttl time.Duration) (v proto.Message, ch chan struct{}) {
	c.mu.Lock()
	ele, ok := c.store[key]
	if ok {
		e := ele.Value.(*entry)
		if e.ttl.After(time.Now()) {
			v = e.val
			ch = e.ch
			c.list.MoveToBack(ele)
		} else {
			e.val = proto.Message{}
			e.ttl = time.Now().Add(ttl)
			e.ch = make(chan struct{}, 1)
			c.list.MoveToBack(ele)
		}
	} else if c.list != nil {
		c.list.PushBack(&entry{
			key: key,
			ttl: time.Now().Add(ttl),
			ch:  make(chan struct{}, 1),
		})
		c.store[key] = c.list.Back()
	}
	c.mu.Unlock()
	return v, ch
}

func (c *LRU) Update(key string, value proto.Message) {
	var ch chan struct{}
	c.mu.Lock()
	ele, ok := c.store[key]
	if ok {
		e := ele.Value.(*entry)
		e.val = value
		e.size = entrySize + elementSize + 2*(stringSSize+len(key)) + value.Size()
		ch = e.ch
		e.ch = nil

		c.size += e.size
		for c.size > c.max {
			if ele = c.list.Front(); ele != nil {
				e = ele.Value.(*entry)
				delete(c.store, e.key)
				c.list.Remove(ele)
				c.size -= e.size
			}
		}
	}
	c.mu.Unlock()
	if ch != nil {
		close(ch)
	}
}

func (c *LRU) Delete(keys []proto.Message) {
	c.mu.Lock()
	for _, k := range keys {
		e, ok := c.store[k.String]
		if ok {
			delete(c.store, k.String)
			c.list.Remove(e)
		}
	}
	c.mu.Unlock()
}

func (c *LRU) DeleteAll() {
	c.mu.Lock()
	c.store = make(map[string]*list.Element)
	c.list = nil
	c.mu.Unlock()
}