package pppool

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)



func TestNewWorkerStack(t *testing.T) {
	size := 1000
	q := newWorkerStack(size)
	require.EqualValues(t, 0, q.len(), "Len error")
	require.Equal(t, true, q.isEmpty(), "IsEmpty error")
	require.Nil(t, q.detach(), "Dequeue error")
}

func TestBinarySearch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip this test on windows platform")
	}

	//test 1
	q := newWorkerStack(0)
	expiry1 := time.Now()
	q.insert(&goWorker{lastUsed: time.Now()})
	require.EqualValues(t, 0, q.binarySearch(time.Now()), "return index should be 0")
	require.EqualValues(t, -1, q.binarySearch(expiry1), "return index should be -1")

	//test 2
	expiry2 := time.Now()
	q.insert(&goWorker{lastUsed: time.Now()})
	require.EqualValues(t, 1, q.binarySearch(time.Now()), "index should be 1")
	require.EqualValues(t, 0, q.binarySearch(expiry2), "index should 0")
	require.EqualValues(t, -1, q.binarySearch(expiry1), "index should be -1")
}

func TestWorkerQueueStack(t *testing.T){
	q := newWorkerStack(0)
	
	for i := 0; i < 5; i++ {
		q.insert(&goWorker{lastUsed: time.Now()})
	}
	require.EqualValues(t, 5, q.len(), "workerQueue error")

	q.detach()
	require.EqualValues(t, 4, q.len(), "workerQueue error")

	time.Sleep(time.Second)

	
	for i := 0; i < 100; i++ {
		q.insert(&goWorker{lastUsed: time.Now()})
	}

	require.EqualValues(t, 104, q.len(), "workerQueue error")
	q.refresh(time.Second)
	
	require.EqualValues(t, 100, q.len(),"workerQueue error")
}

func TestExample(t *testing.T) {
	arry := make([]int, 0, 10)
	require.EqualValues(t, 0, len(arry))

	for i := 0; i < 10; i++ {
		arry = append(arry, i)
	}
	require.EqualValues(t, 10, len(arry))

	cpNum := copy(arry, arry[5:])
	require.EqualValues(t, 5, cpNum)

	require.EqualValues(t, 10, len(arry))
	for i := 5; i < 10; i++ {
		require.EqualValues(t, i, arry[i - 5])
	}

	arry = arry[:cpNum]
	require.EqualValues(t, cpNum, len(arry))

}