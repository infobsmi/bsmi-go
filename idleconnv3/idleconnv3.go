package idleconnv3

import (
	"net"
	"sync"
)

type CloseableConn interface {
	Close() error
	Read(buf []byte) (int, error)
	Write(buf []byte) (int, error)
}
type IdleTimeoutConnV3 struct {
	mt     sync.Mutex
	update func()
	Conn   CloseableConn
}

func NewIdleTimeoutConnV3(conn net.Conn, fn func()) *IdleTimeoutConnV3 {
	c := &IdleTimeoutConnV3{
		Conn:   conn,
		update: fn,
	}
	return c
}

func (ic *IdleTimeoutConnV3) Read(buf []byte) (int, error) {
	go ic.UpdateIdleTime()
	return ic.Conn.Read(buf)
}

func (ic *IdleTimeoutConnV3) UpdateIdleTime() {
	ic.mt.Lock()
	defer ic.mt.Unlock()
	ic.update()
}

func (ic *IdleTimeoutConnV3) Write(buf []byte) (int, error) {
	go ic.UpdateIdleTime()
	return ic.Conn.Write(buf)
}

func (c *IdleTimeoutConnV3) Close() {
	if c.Conn != nil {
		c.Conn.Close()
	}
}
