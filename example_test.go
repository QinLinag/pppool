package pppool_test

import (
	"fmt"
	"pppool"
	"sync"
	"sync/atomic"

)

var(
	sum int32
	wg sync.WaitGroup
)

func incSumInt(i int32) {
	atomic.AddInt32(&sum, i)
	wg.Done()
}

func ExamplePool() {
	atomic.StoreInt32(&sum, 0)
	runTimes := 1000
	wg.Add(runTimes)
	pool, _ := pppool.NewPool(10)
	defer pool.Release()

	for i := 0; i < runTimes; i++ {
		j := i
		_ = pool.Submit(func() {
			incSumInt(int32(j))
		})
	} 
	wg.Wait()
	fmt.Printf("The result is %d\n", sum)
	fmt.Println("----")
}

func main() {
	ExamplePool()
}