package razlog

import (
	"fmt"
	"sync/atomic"
)

type RazLog struct {
	cnt    int32
	maxCnt int32
	val    any
}

func NewRazLog(maxCnt int32) *RazLog {
	return &RazLog{
		maxCnt: maxCnt,
		cnt:    0,
	}
}

func (r *RazLog) Add(log string, adds ...any) *RazLog {
	if atomic.LoadInt32(&r.cnt) >= r.maxCnt {
		atomic.StoreInt32(&r.cnt, 0)

		r.val = fmt.Sprintf(log, adds...)
		return r
	}
	r.val = nil
	atomic.StoreInt32(&r.cnt, atomic.LoadInt32(&r.cnt)+1)

	return r

}

func (r *RazLog) Do(p func(v ...any)) {
	if r.val != nil {
		p(r.val)
	}
}
