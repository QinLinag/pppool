package pppool

import (
	"errors"
	"time"
)

var errQueueIsFull = errors.New("the queue is full")
type queueType int
const (
	queueTypeStack queueType = 1 << iota
	queueTypeLoopQueue
)

type workerQueue interface {
	len() int
	isEmpty() bool
	insert(worker) error
	detach() worker
	refresh(duration time.Duration) []worker
	reset()
	clean()
}




func newWorkerQueue(qType queueType, size int) workerQueue {
	switch qType {
	case queueTypeStack:
		return newWorkerStack(size)
	case queueTypeLoopQueue:
		return newWorkerLoopQueue(size)
	default:
		return newWorkerStack(size)
	}
}