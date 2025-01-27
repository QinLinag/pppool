package pppool

import (
	"runtime/debug"
	"time"
)

type worker interface {
	run()
	finish()
	lastUsedTime() time.Time
	setLastUsedTime(time.Time)
	inputFunc(func())
	inputArg(any)
}

type goWorker struct {
	worker

	pool *Pool

	task chan func()

	lastUsed time.Time
}

func (w *goWorker) run() {
	w.pool.addRunning(1)
	go func() {
		defer func() {
			if w.pool.addRunning(-1) == 0 && w.pool.IsClosed() {
				w.pool.once.Do(func() {
					close(w.pool.allDone)
				})
			}
			w.pool.workerCache.Put(w)
			if p := recover(); p != nil {
				if ph := w.pool.options.PanicHandler; ph != nil {
					ph(p)
				} else {
					w.pool.options.Logger.Printf("worker exits from panic: %v\n%s\n", p, debug.Stack())
				}
			}
			//cal signal() here in case there are goroutines waiting for avaliable workers
			w.pool.cond.Signal()
		}()
		
		for fn := range w.task {  //阻塞等待人物
			if fn == nil {  //finish
				return
			}
			fn()
			if ok := w.pool.revertWorker(w); !ok {  //将worker放入pool的worker queue中，
				return
			}
		}
	}()
}


func (w *goWorker) finish() {
	w.task <- nil
}

func (w *goWorker) lastUsedTime() time.Time {
	return w.lastUsed
}

func (w *goWorker) setLastUsedTime(t time.Time) {
	if !t.After(time.Now()) {
		w.lastUsed = t
	} else {
		//报错
	}
}

func (w *goWorker) inputFunc(fn func()) {
	w.task <- fn
}