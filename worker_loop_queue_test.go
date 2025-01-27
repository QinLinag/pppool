package pppool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)




func TestNewWorkerLoopQueue(t *testing.T) {
	q := newWorkerLoopQueue(10)
	require.EqualValues(t, 10, q.size)
	require.EqualValues(t, 0, q.head)
	require.EqualValues(t, 0, q.tail)
	require.Equal(t, false, q.isFull)
	require.Equal(t, true, q.isEmpty())
	require.Equal(t, 0, q.len())

	for i := 0; i < 10; i++ {
		q.insert(&goWorker{lastUsed: time.Now()})
	}
	require.Equal(t, false, q.isEmpty())
	require.EqualValues(t, 10, q.len())
	require.EqualValues(t, 0, q.head)
	require.EqualValues(t, q.head, q.tail)
	require.Equal(t, true, q.isFull)

	err := q.insert(&goWorker{lastUsed: time.Now()})
	require.NotNil(t, err, "queue is full, return should be error")
	require.Equal(t, err.Error(), ErrQueueIsFull.Error())

	for i := 0; i < 5; i++ {
		q.detach()
	}
	require.EqualValues(t, 5, q.len())
	require.Equal(t, false, q.isFull)
	require.EqualValues(t, 5, q.head)
	require.EqualValues(t, 0, q.tail)


	//tail < head的情况     测试tail是否能够正常循环
	for i := 0; i < 4; i++ {
		q.insert(&goWorker{lastUsed: time.Now()})
	}
	require.Equal(t, false, q.isFull)
	require.Equal(t, false, q.isEmpty())
	require.EqualValues(t, 9, q.len())
	require.EqualValues(t, 5, q.head)
	require.EqualValues(t, 4, q.tail)

	q.insert(&goWorker{lastUsed: time.Now()})
	require.Equal(t, true, q.isFull)
	require.Equal(t, false, q.isEmpty())
	require.EqualValues(t, 10, q.len())
	require.EqualValues(t, 5, q.head)
	require.EqualValues(t, 5, q.tail)
	
	//head < tail   测试head是否能正常循环

	for i := 0; i < 5; i++ {
		q.detach()
	}
	require.EqualValues(t, 5, q.len())
	require.Equal(t, false, q.isFull)
	require.EqualValues(t, 0, q.head)
	require.EqualValues(t, 5, q.tail)
	require.EqualValues(t, false, q.isFull)
	require.EqualValues(t, false, q.isEmpty())



	q.reset()
	require.EqualValues(t, 0, q.len())
	require.EqualValues(t, 0, q.size)
	require.EqualValues(t, 0, q.head)
	require.EqualValues(t, 0, q.tail)

}




func TestWorkerLoopQueueBinarySearch(t *testing.T) {
	//test 1  不满，并且 tail > head=0
	q1 := newWorkerLoopQueue(1000)
	expiry1 := time.Now()

	for i := 0; i < 200; i++ {
		q1.insert(&goWorker{lastUsed: time.Now()})
	}
	require.EqualValues(t, -1, q1.binarySearch(expiry1), "index should be -1, means expiriedWorkder nums is 0")
	require.EqualValues(t, 199, q1.binarySearch(time.Now()), "index should be 1, expiriedWorker nums is 200")

	//test 2 满的情况  isFull==true && tail==head==0
	q2 := newWorkerLoopQueue(1000)
	expiry2 := time.Now()
	for i := 0; i < 1000; i++ {
		q2.insert(&goWorker{lastUsed: time.Now()})
	}
	require.EqualValues(t, 0, q2.head)
	require.EqualValues(t, 0, q2.tail)
	require.EqualValues(t, -1, q2.binarySearch(expiry2), "index should be -1, means expiriedWorkder nums is 0")
	require.EqualValues(t, 999, q2.binarySearch(time.Now()), "index should be 1, expiriedWorker nums is 1000")


	//test 3 不满 tail > head > 0
	q3 := newWorkerLoopQueue(1000)
	expiry3 := time.Now()
	for i := 0; i < 500; i++ {
		q3.insert(&goWorker{lastUsed: time.Now()})
	}
	expiry4 := time.Now()
	for i := 0; i < 300; i++ {
		q3.detach()
	}
	require.EqualValues(t, 200, q3.len())
	require.EqualValues(t, 300, q3.head)
	require.EqualValues(t, 500, q3.tail)
	require.EqualValues(t, 299, q3.binarySearch(expiry3), "没有过期的，index should be (head - 1), means expiriedWorker nums is 0") //head - 1 == 99
	require.EqualValues(t, 499, q3.binarySearch(expiry4), "全部过期200个,index should be (tail - 1), expiriedWorker nums is (tail - head)")

	//test 4 不满 tail < head
	expiry5 := time.Now()
	for i := 0; i < 600; i++ {
		q3.insert(&goWorker{lastUsed: time.Now()})
	}
	require.EqualValues(t, 800, q3.len())
	require.EqualValues(t, 300, q3.head)
	require.EqualValues(t, 100, q3.tail)
	require.EqualValues(t, 299, q3.binarySearch(expiry3), "没有过期的，index should be (head - 1), means expiriedWorker nums is 0") //head - 1 == 99
	require.EqualValues(t, 499, q3.binarySearch(expiry4), "有200个过期，index should be (tail - 1), expiriedWorker nums is (tail - head)")
	require.EqualValues(t, 499, q3.binarySearch(expiry5), "有200个过期，index should be (tail - 1), expiriedWorker nums is (tail - head)")
	require.EqualValues(t, 99, q3.binarySearch(time.Now()), "全部过期800个，index should be (tail - 1), (700+100)")

	//test 5 queue满 并且tail==head！=0 的情况
	expiry6 := time.Now()
	for i := 0; i < 200; i++ {
		q3.insert(&goWorker{lastUsed: time.Now()})
	}
	require.EqualValues(t, 1000, q3.len())
	require.EqualValues(t, 300, q3.head)
	require.EqualValues(t, 300, q3.tail)
	require.EqualValues(t, true, q3.isFull)
	require.EqualValues(t, 299, q3.binarySearch(expiry3), "没有过期的，index should be (head - 1), means expiriedWorker nums is 0") //head - 1 == 99
	require.EqualValues(t, 499, q3.binarySearch(expiry4), "有200个过期，index should be (tail - 1), expiriedWorker nums is (tail - head)")
	require.EqualValues(t, 499, q3.binarySearch(expiry5), "有200个过期，index should be (tail - 1), expiriedWorker nums is (tail - head)")
	require.EqualValues(t, 99, q3.binarySearch(expiry6), "有800个过期，index should be (tail - 1), (700+100)")
	require.EqualValues(t, 299, q3.binarySearch(time.Now()), "全部过期1000个，index should be (tail - 1), (1000 - 300 + 300)")
}

func TestWorkerLoopQueueRefresh(t *testing.T) {
	//test 1
	q1 := newWorkerLoopQueue(100)
	for i := 0; i < 50; i++ {
		q1.insert(&goWorker{lastUsed: time.Now()})
	}
	require.EqualValues(t, 0, q1.head)
	require.EqualValues(t, 50, q1.tail)

	require.EqualValues(t, 0, len(q1.refresh(time.Second)), "不存在，最后一次运行时在一秒前的worker，")
	require.EqualValues(t, 50, len(q1.refresh(0)), "现在时间点前的worker全部过期，50个")
	require.EqualValues(t, 0, q1.len())
	require.EqualValues(t, 50, q1.head)
	require.EqualValues(t, 50, q1.tail)
	

	//test 2
	for i := 0; i < 100; i++ {
		q1.insert(&goWorker{lastUsed: time.Now()})
	}
	require.EqualValues(t, 50, q1.head)
	require.EqualValues(t, 50, q1.tail)
	require.EqualValues(t, true, q1.isFull)

	require.EqualValues(t, 0, len(q1.refresh(time.Second)), "不存在，最后一次运行时在一秒前的worker，")
	require.EqualValues(t, 100, len(q1.refresh(0)), "现在时间点前的worker全部过期，100个")
	require.EqualValues(t, 0, q1.len())
	require.EqualValues(t, 50, q1.head)
	require.EqualValues(t, 50, q1.tail)
	require.EqualValues(t, true, q1.isEmpty())

	//test 3
	for i := 0; i < 40; i++ {
		q1.insert(&goWorker{lastUsed: time.Now()})
	}
	require.EqualValues(t, 40, q1.len())
	require.EqualValues(t, 50, q1.head)
	require.EqualValues(t, 90, q1.tail)
	time.Sleep(2 * time.Second)

	for i := 0; i < 40; i++ {
		q1.insert(&goWorker{lastUsed: time.Now()})
	}
	require.EqualValues(t, 80, q1.len())
	require.EqualValues(t, 50, q1.head)
	require.EqualValues(t, 30, q1.tail)
	require.EqualValues(t, 0, len(q1.refresh(3 * time.Second)), "不存在，最后一次运行时在3秒前的worker，")
	require.EqualValues(t, 80, q1.len())
	require.EqualValues(t, 50, q1.head)
	require.EqualValues(t, 30, q1.tail)

	require.EqualValues(t, 40, len(q1.refresh(1 * time.Second)), "存活超过1s的全部是超时，有40个")
	require.EqualValues(t, 40, q1.len())

	require.EqualValues(t, 90, q1.head)
	require.EqualValues(t, 30, q1.tail)
	
	time.Sleep(time.Second)
	//test 4
	for i := 0; i < 40; i++ {
		q1.insert(&goWorker{lastUsed: time.Now()})
	}
	require.EqualValues(t, 90, q1.head)
	require.EqualValues(t, 70, q1.tail)
	require.EqualValues(t, 40, len(q1.refresh(time.Second)), "有40个")

	require.EqualValues(t, 40, q1.len())
	require.EqualValues(t, 30, q1.head)
	require.EqualValues(t, 70, q1.tail)
}