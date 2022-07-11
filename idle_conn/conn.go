package idle_conn

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const IdleTimeout = 120 * time.Second

var logger = log.Default()

type ValidConn interface {
	net.Conn
}
type IdleConn[T ValidConn] struct {
	Conn   T
	mt     sync.Mutex
	LastTs int64
}

func (ic *IdleConn[T]) Read(buf []byte) (int, error) {
	go ic.UpdateIdleTime(time.Now().Add(IdleTimeout))
	return ic.Conn.Read(buf)
}

func (ic *IdleConn[T]) UpdateIdleTime(t time.Time) {
	if ic.mt.TryLock() {
		defer ic.mt.Unlock()

		tsNow := time.Now()
		lastTs := atomic.LoadInt64(&ic.LastTs)
		if lastTs > tsNow.Unix() {
			return
		}
		//老的时间
		tsNext := tsNow.Add(4 * time.Second)
		tsRenew := tsNext.Add(IdleTimeout)
		atomic.StoreInt64(&ic.LastTs, tsNext.Unix())

		logger.Printf("获取到锁,且应该更新, oldTs: %+v, 更新超时时间: %+v\n", ic.LastTs, tsRenew)
		_ = ic.Conn.SetReadDeadline(tsRenew)
		_ = ic.Conn.SetWriteDeadline(tsRenew)

	}
}

func (ic *IdleConn[T]) Write(buf []byte) (int, error) {
	go ic.UpdateIdleTime(time.Now().Add(IdleTimeout))
	return ic.Conn.Write(buf)
}
