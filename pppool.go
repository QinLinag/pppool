package pppool

import (
	"context"
	"errors"
	"log"
	"math"
	"os"
	syncx "pppool/pkg/sync"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultPoolSize          = math.MaxInt32
	DefaultCleanIntervalTime = time.Second
)

const (
	OPEND = iota

	CLOSED
)

var (
	ErrPoolOverload        = errors.New("too many goroutines blocked on submit or Nonblocking is set")
	ErrorPoolClosed        = errors.New("the pool has been closed")
	ErrInvalidPoolExpiry   = errors.New("invalid expiry for pool")
	ErrInvalidPreAllocSize = errors.New("can not set up a negative capacity under PreAlloc mode")

	workerChanCap = func() int {
		if runtime.GOMAXPROCS(0) == 1 {
			return 0
		}
		return 1
	}()

	defaultLogger  Logger = Logger(log.New(os.Stderr, "[pppool]: ", log.LstdFlags|log.Lmsgprefix|log.Lmicroseconds))
	defaultPool, _        = NewPool(DefaultPoolSize)
)

func Submit(task func()) error {
	return defaultPool.Submit(task)
}

func Running() int {
	return defaultPool.Running()
}

func Cap() int {
	return defaultPool.Cap()
}

func Free() int {
	return defaultPool.Free()
}

func Release() {
	defaultPool.Release()
}

type Logger interface {
	Printf(format string, args ...any)
}

type poolCommon struct {
	capacity int32

	running int32

	lock sync.Locker

	workers workerQueue

	state int32

	cond sync.Cond

	allDone chan struct{}

	once *sync.Once //并发协程只会执行一次，比如关闭资源、申请资源

	workerCache sync.Pool //对象复用

	waiting   int32
	purgeDone int32
	purgeCtx  context.Context
	stopPurge context.CancelFunc

	ticktockDone int32
	ticktockCtx  context.Context
	stopticktock context.CancelFunc

	now     atomic.Value
	options *Options
}

func newPool(size int, options ...Option) (*poolCommon, error) {
	if size <= 0 {
		size = -1
	}

	opts := loadOptions(options...)

	if !opts.DisablePurge {
		if expiry := opts.ExpiryDuration; expiry < 0 {
			return nil, ErrInvalidPoolExpiry
		} else if expiry == 0 {
			opts.ExpiryDuration = DefaultCleanIntervalTime
		}

	}

	if opts.Logger == nil {
		opts.Logger = defaultLogger
	}

	p := &poolCommon{
		capacity: int32(size),
		allDone:  make(chan struct{}),
		lock:     syncx.NewSpinLock(),
		once:     &sync.Once{},
		options:  opts,
	}

	if p.options.PreAlloc {
		if size == -1 {
			return nil, ErrInvalidPreAllocSize
		}
		p.workers = newWorkerQueue(queueTypeLoopQueue, size)
	} else {
		p.workers = newWorkerQueue(queueTypeStack, 0)
	}
	p.cond = *sync.NewCond(p.lock)
	p.goPurge()    //开启一个协程去refresh过期的worker
	p.goTicktock() //开启一个协程去更新pool的时间

	return p, nil
}

func (p *poolCommon) purgeStaleWorkers() {
	ticker := time.NewTicker(p.options.ExpiryDuration)

	defer func() {
		ticker.Stop()
		atomic.StoreInt32(&p.purgeDone, 1)
	}()

	purgeCtx := p.purgeCtx
	for {
		select {
		case <-purgeCtx.Done():
			return
		case <-ticker.C:
		}
		if p.IsClosed() {
			break
		}

		var isDormant bool
		p.lock.Lock()
		staleWorkers := p.workers.refresh(p.options.ExpiryDuration)
		n := p.Running()
		isDormant = n == 0 || n == len(staleWorkers)
		p.lock.Unlock()

		// clean up the stale workers
		for i := range staleWorkers {
			staleWorkers[i].finish()
			staleWorkers[i] = nil
		}
		// p.workers.clean()

		//There might be a situation where all workers have been cleaned up
		//while some invokers still are stuck in p.cond.Wait(), then we need to awake those invokers.
		if isDormant && p.Waiting() > 0 {
			p.cond.Broadcast()
		}
	}
}

const nowTimeUpdateInterval = 500 * time.Millisecond

func (p *poolCommon) ticktock() {
	ticker := time.NewTicker(nowTimeUpdateInterval)
	defer func() {
		ticker.Stop()
		atomic.StoreInt32(&p.ticktockDone, 1)
	}()
	ticktockCtx := p.ticktockCtx
	for {
		select {
		case <-ticktockCtx.Done():
			return
		case <-ticker.C:
		}
		if p.IsClosed() {
			break
		}
		p.now.Store(time.Now())
	}
}

func (p *poolCommon) goPurge() {
	if p.options.DisablePurge {
		return
	}

	p.purgeCtx, p.stopPurge = context.WithCancel(context.Background())
	go p.purgeStaleWorkers()
}

func (p *poolCommon) goTicktock() {
	p.now.Store(time.Now())
	p.ticktockCtx, p.stopticktock = context.WithCancel(context.Background())
	go p.ticktock()
}

func (p *poolCommon) Waiting() int {
	return int(atomic.LoadInt32(&p.waiting))
}

func (p *poolCommon) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

func (p *poolCommon) Free() int {
	c := p.Cap()
	if c < 0 {
		return -1
	}
	return c - p.Running()
}

func (p *poolCommon) IsClosed() bool {
	return atomic.LoadInt32(&p.state) == CLOSED
}

func (p *poolCommon) Release() {
	if !atomic.CompareAndSwapInt32(&p.state, OPEND, CLOSED) {
		return
	}
	if p.stopPurge != nil {
		p.stopPurge()
		p.stopPurge = nil
	}
	if p.stopticktock != nil {
		p.stopticktock()
		p.stopticktock = nil
	}
	p.lock.Lock()
	p.workers.reset()
	p.lock.Unlock()
	p.cond.Broadcast()
}

func (p *poolCommon) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

func (p *poolCommon) addRunning(delta int) int {
	return int(atomic.AddInt32(&p.running, int32(delta)))
}

func (p *poolCommon) addWaiting(delta int) {
	atomic.AddInt32(&p.waiting, int32(delta))
}

func (p *poolCommon) retrieveWorker() (w worker, err error) {
	p.lock.Lock()

retry:

	//直接中workers中取一个worker
	if w = p.workers.detach(); w != nil {
		p.lock.Unlock()
		return
	}
	//if worker queue is empry, and we don't run out of the pool capacity
	//then just spawn a new worker goroutine
	if capacity := p.Cap(); capacity == -1 || capacity > p.Running() {
		p.lock.Unlock()
		w = p.workerCache.Get().(worker)
		w.run()
		return
	}

	//Bail out early if it's in nonblocking mode or the number of pending callers reaches the maximum limit value
	if p.options.Nonblocking || (p.options.MaxBlockingTasks != 0 && p.Waiting() >= p.options.MaxBlockingTasks) {
		p.lock.Unlock()
		return nil, ErrPoolOverload
	}
	p.addWaiting(1)
	p.cond.Wait()
	p.addWaiting(-1)
	if p.IsClosed() {
		p.lock.Unlock()
		return nil, ErrorPoolClosed
	}

	goto retry
}

func (p *poolCommon) revertWorker(worker worker) bool {
	if capacity := p.Cap(); (capacity > 0 && p.Running() > capacity) || p.IsClosed() {
		p.cond.Broadcast()
		return false
	}
	worker.setLastUsedTime(time.Now())

	p.lock.Lock()

	if p.IsClosed() {
		p.lock.Lock()
		return false
	}
	if err := p.workers.insert(worker); err != nil {
		return false
	}
	// notfiy the invoker stuct in "retrieveWorker()" of there is an avalible worker in the worker queue
	p.cond.Signal()
	p.lock.Unlock()
	return true
}
