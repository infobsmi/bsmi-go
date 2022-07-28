package idleconnv3

import (
	"net"
	"testing"
)

func TestIdleConnV3(t *testing.T) {
	var a net.Conn
	var _ = NewIdleTimeoutConnV3(a, func() {

	})
}
