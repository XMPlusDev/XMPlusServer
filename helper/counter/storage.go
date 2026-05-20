package counter

import (
	"sync"
	"sync/atomic"
)

// TrafficStorage holds atomic byte counters for one user.
type TrafficStorage struct {
	UpCounter   atomic.Int64
	DownCounter atomic.Int64
}

// TrafficCounter holds per-user storage, keyed by email.
type TrafficCounter struct {
	counters sync.Map // key: email → *TrafficStorage
}

func NewTrafficCounter() *TrafficCounter {
	return &TrafficCounter{}
}

func (c *TrafficCounter) GetCounter(email string) *TrafficStorage {
	if v, ok := c.counters.Load(email); ok {
		return v.(*TrafficStorage)
	}
	s := &TrafficStorage{}
	if v, loaded := c.counters.LoadOrStore(email, s); loaded {
		return v.(*TrafficStorage)
	}
	return s
}

func (c *TrafficCounter) GetUpCount(email string) int64 {
	if v, ok := c.counters.Load(email); ok {
		return v.(*TrafficStorage).UpCounter.Load()
	}
	return 0
}

func (c *TrafficCounter) GetDownCount(email string) int64 {
	if v, ok := c.counters.Load(email); ok {
		return v.(*TrafficStorage).DownCounter.Load()
	}
	return 0
}

func (c *TrafficCounter) Reset(email string) {
	if v, ok := c.counters.Load(email); ok {
		v.(*TrafficStorage).UpCounter.Store(0)
		v.(*TrafficStorage).DownCounter.Store(0)
	}
}

func (c *TrafficCounter) Delete(email string) {
	c.counters.Delete(email)
}

func (c *TrafficCounter) Len() int {
	n := 0
	c.counters.Range(func(_, _ interface{}) bool { n++; return true })
	return n
}