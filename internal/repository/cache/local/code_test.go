package local

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestNewLocalCache(t *testing.T) {
	l_cache := NewLocalCache()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	fmt.Println("操作结果:", l_cache.Set(ctx, "login", "13719088000", "789789", 3))

	//for i := 0; i < 100; i++ {
	//	go func() {
	//		fmt.Println("操作结果:", l_cache.Set(ctx, "login", "13719088025", "789789", 3))
	//	}()
	//
	//}
	fmt.Println("读操作.......................")
	for i := 0; i < 3; i++ {
		go func() {
			ok, err := l_cache.Verify(ctx, "login", "13719088000", "789788")
			fmt.Println(ok, err)
		}()

	}
	//fmt.Println("操作结果:", l_cache.Set(ctx, "login", "13719088025", "789789", 3))
	//fmt.Println("操作结果:", l_cache.Set(ctx, "login", "13719088025", "789789", 3))
	time.Sleep(time.Second * 5)
}
