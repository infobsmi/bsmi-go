package idle_conn

import (
	"net"
	"time"
)

const IdleTimeout = 120 * time.Second

type ValidConn interface {
	net.Conn
}
type IdleConn[T ValidConn] struct {
	Conn T
}

func (ic *IdleConn[T]) Read(buf []byte) (int, error) {
	go ic.UpdateIdleTime(time.Now().Add(IdleTimeout))
	return ic.Conn.Read(buf)
}

func (ic *IdleConn[T]) UpdateIdleTime(t time.Time) {
	_ = ic.Conn.SetDeadline(t)
}

func (ic *IdleConn[T]) Write(buf []byte) (int, error) {
	go ic.UpdateIdleTime(time.Now().Add(IdleTimeout))
	return ic.Conn.Write(buf)
}
