package idle_conn

import (
	"github.com/panjf2000/ants/v2"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const DefaultIdleTimeout = 20 * time.Second

var (
	logger   = log.Default()
	AntsPool *ants.Pool
)

func init() {
	if AntsPool == nil {
		AntsPool, _ = ants.NewPool(2000, ants.WithNonblocking(true))
	}
}

type ValidConn interface {
	net.Conn
}
type IdleConn[T ValidConn] struct {
	Conn        T
	mt          sync.Mutex
	LastTs      int64
	IdleTimeout time.Duration
}

// Create new IdleConn with default IdleTimeout
func NewIdleConn[T ValidConn](tt T) *IdleConn[T] {
	p := &IdleConn[T]{
		Conn:        tt,
		IdleTimeout: DefaultIdleTimeout,
	}
	return p
}

// Create new IdleConn with custom IdleTimeOut
func NewIdleConnWithIdleTimeOut[T ValidConn](tt T, idleTimeOut time.Duration) *IdleConn[T] {
	p := &IdleConn[T]{
		Conn:        tt,
		IdleTimeout: idleTimeOut,
	}
	return p
}

// Read data
func (ic *IdleConn[T]) Read(buf []byte) (int, error) {
	defer AntsPool.Submit(ic.UpdateIdleTime)
	return ic.Conn.Read(buf)
}

// Update deadline
func (ic *IdleConn[T]) UpdateIdleTime() {
	if ic.mt.TryLock() {
		defer ic.mt.Unlock()

		tsNow := time.Now()
		lastTs := atomic.LoadInt64(&ic.LastTs)
		if lastTs > tsNow.Unix() {
			return
		}
		//老的时间
		tsNext := tsNow.Add(4 * time.Second)
		tsRenew := tsNext.Add(ic.IdleTimeout)
		atomic.StoreInt64(&ic.LastTs, tsNext.Unix())

		logger.Printf("获取到锁,且应该更新, oldTs: %+v, 更新超时时间: %+v\n", ic.LastTs, tsRenew)
		_ = ic.Conn.SetReadDeadline(tsRenew)
		_ = ic.Conn.SetWriteDeadline(tsRenew)

	}
}

// Write data
func (ic *IdleConn[T]) Write(buf []byte) (int, error) {

	defer AntsPool.Submit(ic.UpdateIdleTime)
	return ic.Conn.Write(buf)
}
