package task

import (
	"context"
	"log"
	"sync"
)

var Logger *log.Logger

// OnSuccess executes g() after f() returns nil.
func OnSuccess(f func() error, g func() error) func() error {
	return func() error {
		if err := f(); err != nil {
			return err
		}
		return g()
	}
}

// Run executes a list of tasks in parallel, returns the first error encountered or nil if all tasks pass.
func Run(ctx context.Context, tasks ...func() error) error {
	allDone := make(chan bool, 1)
	var wg sync.WaitGroup
	earlyErr := make(chan error, len(tasks))

	for _, fft := range tasks {
		wg.Add(1)

		task := fft
		go func() {
			defer wg.Done()
			err := task()
			if err == nil {
				Logger.Println("task running ok, no err")
				return
			}
			Logger.Printf("task running failed, has error:%s", err.Error())
			earlyErr <- err
		}()
	}
	go func() {
		wg.Wait()
		allDone <- true
	}()

	select {
	case err := <-earlyErr:
		Logger.Println("quit due to error")
		return err
	case <-ctx.Done():
		Logger.Println("quit due to ctx.Err")
		return ctx.Err()
	case <-allDone:
		Logger.Println("All done")
		return nil
	}

}
