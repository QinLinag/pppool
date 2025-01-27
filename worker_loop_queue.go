package pppool

import (
	"errors"
	"time"
)

var (
	ErrQueueIsFull = errors.New("worker queue is full")
)

type loopQueue struct {
	items  []worker
	expiry []worker

	head   int
	tail   int
	size   int  //循环队列总大小
	isFull bool //本循环队列并不牺牲一个存储单元
}

func newWorkerLoopQueue(size int) *loopQueue {
	if size <= 0 {
		return nil
	}
	return &loopQueue{ //初始loopqueue为空 head==tail==0
		items:  make([]worker, size), //预先全部开辟
		tail:   0,
		head:   0,
		size:   size,
		isFull: false,
	}
}

func (wq *loopQueue) len() int {
	if wq.isFull && wq.tail == wq.head {
		return wq.size
	}
	if wq.isEmpty() {
		return 0
	}
	if wq.tail > wq.head {
		return wq.tail - wq.head
	} else {
		return (wq.size - wq.head) + wq.tail
	}
}

func (wq *loopQueue) isEmpty() bool {
	return wq.head == wq.tail && !wq.isFull
}

func (wq *loopQueue) insert(w worker) error {
	if wq.isFull {
		return ErrQueueIsFull
	}
	wq.items[wq.tail] = w
	wq.tail = (wq.tail + 1) % wq.size

	//判断是否满
	if wq.tail == wq.head {
		wq.isFull = true
	}
	return nil
}

func (wq *loopQueue) detach() worker {
	if wq.isEmpty() {
		return nil
	}

	w := wq.items[wq.head]
	wq.items[wq.head] = nil
	wq.head = (wq.head + 1) % wq.size
	wq.isFull = false
	return w
}

func (wq *loopQueue) refresh(duration time.Duration) []worker {
	if wq.isEmpty() {
		return nil
	}
	expiryTime := time.Now().Add(-duration)
	index := wq.binarySearch(expiryTime)

	if index >= wq.size || index < -1 {
		//TODO error
	}

	if wq.head < wq.tail {
		if index == (wq.head - 1) {
			return nil
		}
		if index >= wq.tail || index < (wq.head-1) {
			//TODO error
		}
		m := index - wq.head + 1
		wq.expiry = wq.expiry[:0]
		wq.expiry = append(wq.expiry, wq.items[wq.head:index+1]...)
		if m != len(wq.expiry) {
			//TODO
		}
		for i := wq.head; i <= index; i++ {
			wq.items[i] = nil
		}
		wq.head = (index + 1) % wq.size
		wq.isFull = false
		return wq.expiry
	} else {
		if index == (wq.head - 1) {
			if !wq.isFull {
				return nil
			} else {
				if expiryTime.Before(wq.items[wq.head].lastUsedTime()) {
					return nil
				}
			}
		}
		if wq.head < index && index < wq.size {
			m := index - wq.head + 1
			wq.expiry = wq.expiry[:0]
			wq.expiry = append(wq.expiry, wq.items[wq.head:index+1]...)
			if m != len(wq.expiry) {
				//TODO error
			}
			for i := wq.head; i <= index; i++ {
				wq.items[i] = nil
			}
			wq.head = (index + 1) % wq.size
			wq.isFull = false
			return wq.expiry
		} else {
			m := (wq.size - wq.head) + (index + 1)
			wq.expiry = wq.expiry[:0]
			wq.expiry = append(wq.expiry, wq.items[wq.head:wq.size]...)
			wq.expiry = append(wq.expiry, wq.items[0:index+1]...)
			if m != len(wq.expiry) {
				//TODO error
			}
			for i := wq.head; i < wq.size; i++ {
				wq.items[i] = nil
			}
			for i := 0; i <= index; i++ {
				wq.items[i] = nil
			}
			wq.head = (index + 1) % wq.size
			wq.isFull = false
			return wq.expiry
		}
	}
}

func (wq *loopQueue) reset() {
	if wq.isEmpty() {
		return
	}

retry:
	if w := wq.detach(); w != nil {
		w.finish()
		goto retry
	}
	wq.items = wq.items[:0]
	wq.size = 0
	wq.head = 0
	wq.tail = 0
}

func (wq *loopQueue) binarySearch(expiryTime time.Time) int {
	if wq.tail > wq.head { //正常的二分查找
		left := wq.head
		right := wq.tail - 1
		return wq.binarySearchHelp(left, right, expiryTime)
	} else {
		if expiryTime.Before(wq.items[wq.size-1].lastUsedTime()) {
			left := wq.head
			right := wq.size - 1
			return wq.binarySearchHelp(left, right, expiryTime)
		} else { //head右面全部过期了
			if wq.tail == 0 { //左边没有过期的
				return wq.size - 1
			} else {
				left := 0
				right := wq.tail - 1
				index := wq.binarySearchHelp(left, right, expiryTime)
				if index == -1 { //左边没有过期的
					return wq.size - 1
				}
				return index
			}
		}
	}
}

func (wq *loopQueue) binarySearchHelp(left int, right int, expiryTime time.Time) int {
	for left <= right {
		mid := left + (right-left)>>1
		if expiryTime.Before(wq.items[mid].lastUsedTime()) {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}
	return right
}

func (ws *loopQueue) clean() {
	if len(ws.expiry) == 0 {
		return
	}
	for i := range ws.expiry {
		ws.expiry[i].finish()
		ws.expiry[i] = nil
	}
}
