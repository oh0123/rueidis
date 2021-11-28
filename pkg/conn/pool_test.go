package conn

import (
	"errors"
	"sync/atomic"
	"testing"
)

func TestPool(t *testing.T) {
	setup := func(size int) (*pool, *int32) {
		var count int32
		return newPool(size, func() *wire {
			atomic.AddInt32(&count, 1)
			return &wire{}
		}), &count
	}

	t.Run("DefaultPoolSize", func(t *testing.T) {
		p := newPool(0, func() *wire { return nil })
		if cap(p.list) == 0 {
			t.Fatalf("DefaultPoolSize is not applied")
		}
	})

	t.Run("Reuse", func(t *testing.T) {
		pool, count := setup(100)
		for i := 0; i < 1000; i++ {
			pool.Store(pool.Acquire())
		}
		if atomic.LoadInt32(count) != 1 {
			t.Fatalf("pool does not reuse connection")
		}
	})

	t.Run("NotExceed", func(t *testing.T) {
		conn := make([]*wire, 100)
		pool, count := setup(len(conn))
		for i := 0; i < len(conn); i++ {
			conn[i] = pool.Acquire()
		}
		if atomic.LoadInt32(count) != 100 {
			t.Fatalf("unexpected acquire count")
		}
		go func() {
			for i := 0; i < len(conn); i++ {
				pool.Store(conn[i])
			}
		}()
		for i := 0; i < len(conn); i++ {
			pool.Acquire()
		}
		if atomic.LoadInt32(count) > 100 {
			t.Fatalf("pool must not exceed the size limit")
		}
	})

	t.Run("NoShare", func(t *testing.T) {
		conn := make([]*wire, 100)
		pool, _ := setup(len(conn))
		for i := 0; i < len(conn); i++ {
			w := pool.Acquire()
			go pool.Store(w)
		}
		for i := 0; i < len(conn); i++ {
			conn[i] = pool.Acquire()
		}
		for i := 0; i < len(conn); i++ {
			for j := i + 1; j < len(conn); j++ {
				if conn[i] == conn[j] {
					t.Fatalf("pool must not output acquired connection")
				}
			}
		}
	})
}

func TestPoolError(t *testing.T) {
	setup := func(size int) (*pool, *int32) {
		var count int32
		return newPool(size, func() *wire {
			w := &wire{}
			c := atomic.AddInt32(&count, 1)
			if c%2 == 0 {
				w.error.Store(&errWrap{error: errors.New("any")})
			}
			return w
		}), &count
	}

	t.Run("NotStoreErrConn", func(t *testing.T) {
		conn := make([]*wire, 100)
		pool, count := setup(len(conn))
		for i := 0; i < len(conn); i++ {
			conn[i] = pool.Acquire()
		}
		if atomic.LoadInt32(count) != int32(len(conn)) {
			t.Fatalf("unexpected acquire count")
		}
		for i := 0; i < len(conn); i++ {
			pool.Store(conn[i])
		}
		for i := 0; i < len(conn); i++ {
			conn[i] = pool.Acquire()
		}
		if atomic.LoadInt32(count) != int32(len(conn)+len(conn)/2) {
			t.Fatalf("unexpected acquire count")
		}
	})
}
