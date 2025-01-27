package pppool

type Pool struct {
	*poolCommon
}

func (p *Pool) Submit(task func()) error {
	if p.IsClosed() {
		return ErrorPoolClosed
	}
	w, err := p.retrieveWorker()
	if w != nil {
		w.inputFunc(task)
	}
	return err
}

func NewPool(size int, options ...Option) (*Pool, error) {
	pc, err := newPool(size, options...)
	if err != nil {
		return nil, err
	}
	pool := &Pool{poolCommon: pc}
	pool.workerCache.New = func() any { //sync.Pool 复用缓冲没有对象时应该如何做
		return &goWorker{
			pool: pool,
			task: make(chan func(), workerChanCap),
		}
	}
	return pool, nil
}
