package idleconnv2

import (
	"github.com/infobsmi/bsmi-go/network"
	"github.com/panjf2000/ants/v2"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const DefaultIdleTimeout = 20 * time.Second
const LOG_LIMITER_COUNT = 6

var (
	logger   = log.Default()
	AntsPool *ants.Pool
)

func init() {
	if AntsPool == nil {
		AntsPool, _ = ants.NewPool(2000, ants.WithNonblocking(true))
	}
}

type IdleConn[T network.Conn] struct {
	Conn        T
	mt          sync.Mutex
	LastTs      int64
	TsRenew     int64
	IdleTimeout time.Duration
	shiftLog    int32
}

// Create new IdleConn with default IdleTimeout
func NewIdleConn[T network.Conn](tt T) *IdleConn[T] {
	p := &IdleConn[T]{
		Conn:        tt,
		IdleTimeout: DefaultIdleTimeout,
	}
	return p
}

// Create new IdleConn with custom IdleTimeOut
func NewIdleConnWithIdleTimeOut[T network.Conn](tt T, idleTimeOut time.Duration) *IdleConn[T] {
	p := &IdleConn[T]{
		Conn:        tt,
		IdleTimeout: idleTimeOut,
	}
	return p
}

func (c *IdleConn) GetTsRenew() int64 {
	return atomic.LoadInt64(&c.TsRenew)
}

func (c *IdleConn) razerLog(str string) {
	var cm = "razerLog@idleconnv2 > "
	loadInt32 := atomic.LoadInt32(&c.shiftLog)
	//logger.Println(cm+"loadInt32: %d", loadInt32)
	if loadInt32 >= LOG_LIMITER_COUNT {
		atomic.StoreInt32(&c.shiftLog, 0)

		logger.Println(cm + str)
	}
	nextInt := loadInt32 + 1
	//logger.Println(cm+"nextInt: %d", nextInt)
	atomic.StoreInt32(&c.shiftLog, nextInt)
}
func (c *IdleConn) Close() error {
	return c.Conn.Close()
}

func (c *IdleConn) CleanConnJob() {
	go func() {
		for {
			if c == nil {
				logger.Println(" 清理程序结束")
				break
			}
			if c.GetTsRenew() > 0 && c.GetTsRenew() < time.Now().Unix() {
				logger.Println("长时间没有读写，自动关闭链接: %d", c.GetTsRenew())
				c.Close()
				c.Conn = nil
				c = nil
				break
			}
			time.Sleep(2 * time.Second)
		}
	}()
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
		atomic.StoreInt64(&ic.TsRenew, tsRenew.Unix())

		logger.Printf("获取到锁,且应该更新, oldTs: %+v, 更新超时时间: %+v\n", ic.LastTs, tsRenew)

	}
}

// Write data
func (ic *IdleConn[T]) Write(buf []byte) (int, error) {

	defer AntsPool.Submit(ic.UpdateIdleTime)
	return ic.Conn.Write(buf)
}
