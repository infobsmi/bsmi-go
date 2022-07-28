package timer

import (
	"context"
	"log"
	"sync"
	"time"
)

var Logger log.Logger

type ActivityUpdater interface {
	Update()
}

type ActivityTimer struct {
	sync.RWMutex
	updated     chan struct{}
	onTimeout   func()
	timerClosed bool
}

func (t *ActivityTimer) Update() {
	select {
	case t.updated <- struct{}{}:
	default:
	}
}

func (t *ActivityTimer) check() {
	select {
	case <-t.updated:
	default:
		t.finish()
	}
}

func (t *ActivityTimer) finish() {
	t.Lock()
	defer t.Unlock()

	t.timerClosed = true
	if t.onTimeout != nil {
		t.onTimeout()
		t.onTimeout = nil
	}
}

func (t *ActivityTimer) SetTimeout(timeout time.Duration) {
	if timeout == 0 {
		t.finish()
		return
	}

	//过N 秒，执行一次 check
	t.Update()
	go func() {
		for {
			if t.timerClosed {
				Logger.Println("ActivityTimer finish and close")
				break
			}
			time.Sleep(timeout)
			t.check()
		}
	}()
}

func CancelAfterInactivity(ctx context.Context, cancel func(), timeout time.Duration) *ActivityTimer {
	timer := &ActivityTimer{
		updated:   make(chan struct{}, 1),
		onTimeout: cancel,
	}
	timer.SetTimeout(timeout)
	return timer
}
