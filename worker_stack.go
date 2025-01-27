package pppool

import "time"

type workerStack struct {
	items  []worker
	expiry []worker
}

func newWorkerStack(size int) *workerStack {
	return &workerStack{
		items: make([]worker, 0, size),
	}
}

//实现worker_queue的所有接口

func (ws *workerStack) len() int {
	return len(ws.items)
}

func (ws *workerStack) isEmpty() bool {
	return len(ws.items) == 0
}

func (ws *workerStack) insert(w worker) error {
	ws.items = append(ws.items, w)
	return nil
}

func (ws *workerStack) detach() worker {
	l := ws.len()
	if l == 0 {
		return nil
	}
	w := ws.items[l-1]
	ws.items[l-1] = nil //避免内存泄漏
	ws.items = ws.items[:l-1]
	return w
}

func (ws *workerStack) refresh(duration time.Duration) []worker {
	n := ws.len()
	if n == 0 {
		return nil
	}

	expiryTime := time.Now().Add(-duration)
	index := ws.binarySearch(expiryTime)
	ws.expiry = ws.expiry[:0] //对一个空切片（nil）截取，是不会报错的
	if index != -1 {
		ws.expiry = append(ws.expiry, ws.items[:index]...)
		m := copy(ws.items, ws.items[index+1:])
		for i := m; i < n; i++ {
			ws.items[i] = nil
		}
		ws.items = ws.items[:m] //缩容
	}
	return ws.expiry
}

func (ws *workerStack) binarySearch(expiryTime time.Time) int {
	left := 0
	right := ws.len() - 1

	for left <= right {
		mid := left + ((right - left) >> 1) //避免超出int范围
		if expiryTime.Before(ws.items[mid].lastUsedTime()) {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}
	return right
}

func (ws *workerStack) reset() {
	for i := 0; i < ws.len(); i++ {
		ws.items[i].finish()
		ws.items[i] = nil
	}
	ws.items = ws.items[:0]
}

func (ws *workerStack) clean() {
	if len(ws.expiry) == 0 {
		return
	}
	for i := range ws.expiry {
		ws.expiry[i].finish()
		ws.expiry[i] = nil
	}
}
