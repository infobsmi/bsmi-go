package locker

import (
	"context"
	"time"
)

type TimeoutLocker struct {
	ch chan struct{}
}

func NewTimeoutLocker() *TimeoutLocker {
	return &TimeoutLocker{
		ch: make(chan struct{}, 1),
	}
}

func (l *TimeoutLocker) Unlock() {
	<-l.ch
}

func (l *TimeoutLocker) doLock(ctx context.Context) bool {
	select {
	case l.ch <- struct{}{}:
		return true
	case <-ctx.Done():
		return false
	}
}
func (l *TimeoutLocker) TryLock(t time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), t)
	defer cancel()
	return l.doLock(ctx)
}
