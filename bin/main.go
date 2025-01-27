package main

import (
	"net/http"
	_ "net/http/pprof"
	"pppool"
	"sync/atomic"
	"time"
)

var(
	sum int32
)

func incSumInt(i int32, str string) {
	atomic.AddInt32(&sum, i)
	println("goroutine come in" + "    " + str)
	time.Sleep(1 * time.Second)
}
func test(pool *pppool.Pool, str string) {
	
	defer pool.Release()
	for {
		err := pool.Submit(func() {
			incSumInt(int32(1), str)
		})
		if err != nil {
			println("error")
		}
	} 
}

func main() {
	// 启动 pprof 服务器
    go func() {
       http.ListenAndServe("localhost:8080", nil)
    }()
	atomic.StoreInt32(&sum, 0)
	pool, _ := pppool.NewPool(2, pppool.WithPreAlloc(true), pppool.WithDisablePurge(true))
	go test(pool, "1111")
	go test(pool, "2222")

	select{}
}