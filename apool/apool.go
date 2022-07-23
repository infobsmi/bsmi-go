package apool

import (
	"github.com/panjf2000/ants/v2"
	"log"
	"sync"
	"time"
)

var (
	Logger   *log.Logger
	AntsPool *ants.Pool

	RebootLock           sync.Locker
	InitializeLock       sync.Locker
	GlobalTenIdleTimeOut = 10 * time.Second
)

// 初始化池子

func InitAP() {
	if AntsPool == nil {
		AntsPool, _ = ants.NewPool(50000, ants.WithNonblocking(true),
			ants.WithExpiryDuration(GlobalTenIdleTimeOut), ants.WithLogger(Logger))
	}
}

// 初始化池子带参数
func InitAPWith(
	size int,
	isNonBlocking bool,
	idleTimeOut time.Duration,
	logger *log.Logger,

) {
	InitializeLock.Lock()
	defer InitializeLock.Unlock()
	if AntsPool == nil {
		AntsPool, _ = ants.NewPool(size, ants.WithNonblocking(isNonBlocking),
			ants.WithExpiryDuration(idleTimeOut), ants.WithLogger(logger))
	}
}

// 获取池子实例

func GetAP() *ants.Pool {

	if AntsPool.IsClosed() {
		RebootLock.Lock()
		defer RebootLock.Unlock()
		AntsPool.Reboot()
	}
	return AntsPool
}

// 提交任务
func APSubmit(f func()) {
	err := GetAP().Submit(f)
	if err != nil {
		Logger.Println(err.Error())
	}
}

// 每隔6秒打印 池子动态
func APStat() {
	tk := time.NewTicker(6 * time.Second)
	go func() {
		for {
			select {
			case <-tk.C:
				ap := GetAP()
				Logger.Printf(" pool上限: %d 当前: %d", ap.Cap(), ap.Running())
			}
		}
	}()
}
