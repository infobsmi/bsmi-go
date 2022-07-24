package poolutils

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/exp/slices"
)

var Logger *log.Logger

const POOL_CONN = 11
const LOG_LIMITER_COUNT = 7

type PoolItemConn[T any] struct {
	Conn T
}
type BoxIConn[T any] struct {
	Label string
	Ok    bool
	Item  *PoolItemConn[T]
}

type PoolConn[T any] struct {
	Conns1             []*PoolItemConn[T]
	Conns2             []*PoolItemConn[T]
	pLock1             sync.Mutex
	pLock2             sync.Mutex
	asyncFillLock1     sync.Mutex
	asyncFillLock2     sync.Mutex
	DialFuncRegistered int
	DialFuncForPool    func() (T, error)
	ctFillLog          int32
	ctTidyLog          int32
}

func NewPoolConn[T any]() *PoolConn[T] {
	return &PoolConn[T]{
		DialFuncRegistered: 0,
	}
}
func (sp *PoolConn[T]) PoolGet(idx int, len int) []*PoolItemConn[T] {
	var a []*PoolItemConn[T]
	var cP *[]*PoolItemConn[T]
	switch idx {
	case 1:
		cP = &sp.Conns1
	case 2:
		cP = &sp.Conns2
	}
	if sp.getLenOfPool(idx) == 0 {
		return nil
	}
	conns := *cP
	if conns == nil {
		return nil
	}
	if sp.getLenOfPool(idx) > len {
		pl := len - 1
		a = conns[:pl]
		*cP = conns[pl:]
	} else {
		a = *cP
		*cP = []*PoolItemConn[T]{}
	}
	return a
}
func (sp *PoolConn[T]) PoolPut(idx int, it []*PoolItemConn[T]) {

	var cP *[]*PoolItemConn[T]
	var lk *sync.Mutex
	switch idx {
	case 1:
		cP = &sp.Conns1
		lk = &sp.pLock1
	case 2:
		cP = &sp.Conns2
		lk = &sp.pLock2
	}

	lk.Lock()
	*cP = append(*cP, it...)
	lk.Unlock()
}

// 放入一条链接
func (sp *PoolConn[T]) PoolPutSingle(it T) {
	if len(sp.Conns1) < POOL_CONN {
		sp.pLock1.Lock()
		sp.Conns1 = append(sp.Conns1, &PoolItemConn[T]{
			Conn: it,
		})
		sp.pLock1.Unlock()
	} else {
		sp.pLock2.Lock()
		sp.Conns2 = append(sp.Conns2, &PoolItemConn[T]{
			Conn: it,
		})
		sp.pLock2.Unlock()
	}
}

func (sp *PoolConn[T]) BackgroundCron() {
	go func() {
		tj := time.NewTicker(3 * time.Second)
		for {
			select {
			case <-tj.C:
				go sp.CleanExpirePool(1)
				go sp.CleanExpirePool(2)
			}
		}
	}()
	go func() {
		tk := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-tk.C:
				if sp.DialFuncForPool != nil {
					go sp.asyncFillPool(1)
					go sp.asyncFillPool(2)
				}
			}
		}
	}()

}

func (sp *PoolConn[T]) getLenOfPool(idx int) int {
	var a *[]*PoolItemConn[T]

	switch idx {
	case 1:
		a = &sp.Conns1
	case 2:
		a = &sp.Conns2
	}
	return len(*a)
}

func (sp *PoolConn[T]) asyncFillPool(idx int) {

	var logStr = " asyncFillPool: "
	//重新算一次
	apLen := sp.getLenOfPool(idx)

	if apLen < POOL_CONN {
		logStr += fmt.Sprintf("当前pool %d,  size: %d", idx, apLen)
		//先清理过期的conn

		var tpp []*PoolItemConn[T]
		for j := 0; j < 3; j++ {
			tmpc, err := sp.DialFuncForPool()
			kSize := 6 + apLen
			if kSize > 14 {
				kSize = 14
			}
			if err == nil {
				tpp = append(tpp, &PoolItemConn[T]{
					Conn: tmpc,
				})
			}
		}
		if len(tpp) > 0 {
			sp.PoolPut(idx, tpp)
			go func() {
				//重新算一次
				apLen := sp.getLenOfPool(idx)
				logStr += fmt.Sprintf(" > 补充完毕，新补充: %d, 当前pool: %d size: %d", len(tpp), idx, apLen)
				if atomic.LoadInt32(&sp.ctFillLog) >= LOG_LIMITER_COUNT {
					atomic.StoreInt32(&sp.ctFillLog, 0)

					defer Logger.Println(logStr)
				}
				atomic.StoreInt32(&sp.ctFillLog, atomic.LoadInt32(&sp.ctFillLog)+1)
			}()
		}

	}

}

func (sp *PoolConn[T]) PickConnOrDial(dialerFunc func() (T, error), ctx context.Context, destination net.Addr) (any, error) {

	/*rawConn, err := dialer.Dial(ctx, destination)
	return rawConn, err*/
	if sp.DialFuncRegistered == 0 {
		//log.LogSugar.Infof("DialFuncRegistered == 0, 注册拨号器")
		sp.DialFuncRegistered = 1
		sp.DialFuncForPool = dialerFunc
	}
	pc, err := sp.getConnFromPool()
	return pc, err
}

func (sp *PoolConn[T]) getConnFromPool() (cc any, err error) {

	v, ec := sp.shiftItemFromPool()

	if ec == nil {
		return v.Conn, nil
	}

	//	log.LogSugar.Infof("兜底逻辑，不应该走到这里，手动创建一个拨号，返回链接")
	c, err := sp.shiftItemFromPool()
	if err != nil {
		return nil, err
	}
	return c.Conn, nil
}

func (sp *PoolConn[T]) shiftItemFromPool() (*PoolItemConn[T], error) {

	logStr := ""

	for k := 1; k < 3; k++ {
		x := sp.getConnFromPoolChan(k)
		logStr += fmt.Sprintf("处理第 %d 条消息 > ", k)
		if x.Label != "" {
			logStr += x.Label
		}
		if x.Ok {
			logStr += " > 获取成功"
			Logger.Println(logStr)
			return x.Item, nil
		}
	}

	logStr += " > 所有的方式都失败，手动拨号"
	Logger.Println(logStr)
	c, err := sp.DialFuncForPool()
	if err == nil {
		return &PoolItemConn[T]{
			Conn: c,
		}, nil
	}
	return nil, errors.New("can not dial ")
}

func (sp *PoolConn[T]) getConnFromPoolChan(idx int) BoxIConn[T] {

	var lk *sync.Mutex
	switch idx {
	case 1:
		lk = &sp.pLock1
	case 2:
		lk = &sp.pLock2
	}
	if !lk.TryLock() {
		return BoxIConn[T]{
			Label: " > getConnFromPoolChan 没有获取到lock，所以这一轮先不处理",
			Ok:    false,
			Item:  nil,
		}
	}
	defer lk.Unlock()
	if sp.getLenOfPool(idx) > 0 {
		pb := sp.PoolGet(idx, 3)
		var failedConn []*PoolItemConn[T]
		for i, vv := range pb {
			v := vv
			if v == nil {
				failedConn = append(failedConn, v)
			}
			if v != nil {
				//pb中，index为i的已经被使用了， 重新插入的时候，这个要改
				failedConn = append(failedConn, v)
				go sp.RefillPool(idx, i, pb, failedConn)
				return BoxIConn[T]{
					Label: fmt.Sprintf(" > 来自缓存池 %d，成功获取链接 ;    获取链接成功, 剩余数量: %d", idx, sp.getLenOfPool(idx)),
					Ok:    true,
					Item:  v,
				}
			}
			break
		}
	}
	return BoxIConn[T]{
		Ok:   false,
		Item: nil,
	}
}

func (sp *PoolConn[T]) razerLog(str string, shiftLog *int32) {
	if atomic.LoadInt32(shiftLog) >= LOG_LIMITER_COUNT {
		atomic.StoreInt32(shiftLog, 0)

		Logger.Println(str)
	}
	atomic.StoreInt32(shiftLog, atomic.LoadInt32(shiftLog)+1)
}

func (sp *PoolConn[T]) CleanExpirePool(idx int) {
	var lk *sync.Mutex
	if idx == 1 {
		lk = &sp.asyncFillLock1
	} else {
		lk = &sp.asyncFillLock2
	}
	lk.Lock()

	defer lk.Unlock()

	var a *[]*PoolItemConn[T]
	switch idx {
	case 1:
		a = &sp.Conns1
	case 2:
		a = &sp.Conns2
	}
	apLen := sp.getLenOfPool(idx)
	if apLen > 0 {
		var tp []*PoolItemConn[T]
		var wg sync.WaitGroup

		conns := *a
		var renewCount = 0
		if conns != nil {
			for k := 0; k < apLen; k++ {
				wg.Add(1)
				cv := conns[k]
				go func() {
					defer wg.Done()
					if cv != nil {
						renewCount += 1
						tp = append(tp, cv)
					}
				}()
			}
		}
		wg.Wait()
		tpLen := len(tp)
		if renewCount > 0 {
			logStr := fmt.Sprintf(" > 当前pool %d  连接， 清理前连接数 %d, 清理 %d, 清理后 %d", idx, apLen, apLen-tpLen, tpLen)
			go sp.razerLog(logStr, &sp.ctTidyLog)
			*a = tp
		}
		//把有效的队列，替换为当前队列
	}
}

func (sp *PoolConn[T]) RefillPool(idx int, i int, pb []*PoolItemConn[T], failedConn []*PoolItemConn[T]) {
	var lk *sync.Mutex
	if idx == 1 {
		lk = &sp.asyncFillLock1
	} else {
		lk = &sp.asyncFillLock2
	}

	var a *[]*PoolItemConn[T]
	switch idx {
	case 1:
		a = &sp.Conns1
	case 2:
		a = &sp.Conns2
	}

	//tn := time.Now().UnixMilli()
	var tpp []*PoolItemConn[T]
	for ci, v := range pb {
		if i != ci && v != nil && !slices.Contains(failedConn, v) {
			tpp = append(tpp, v)
		}
	}
	oldLen := len(*a)
	if len(tpp) > 0 {
		lk.Lock()
		*a = append(*a, tpp...)
		Logger.Printf("当前pool %d 回填  有效的连接， 回填 前连接数 %d, 回填 %d, 回填后 %d", idx, oldLen, len(tpp), len(*a))
		defer lk.Unlock()
	}

}
