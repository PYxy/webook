package main

import (
	"fmt"
	at "sync/atomic"
	"time"
)

type B struct {
	age int64
}

func (b *B) Set() {
	at.AddInt64(&b.age, 1)
}
func (b *B) Get() {
	fmt.Println(at.LoadInt64(&b.age))
}

type A struct {
	a []B
}

type Age interface {
	Set()
	Get()
}

func main() {

	v2()
	//a := math.MaxInt64
	//fmt.Println(a)
	//a += 2
	//fmt.Println(a)
}

func atomicT() {
	//status
	a := 1<<5 - 1
	//索引1位置的服务异常了
	a = a - (1 << 1)
	fmt.Println(a)
	fmt.Println("左移动1位", 1<<1)
	fmt.Println("左移动0位", 1<<0)
	for i := 0; i < 5; i++ {
		fmt.Println("索引位置：", i)
		fmt.Println(a&(1<<i) == 0)
	}
	a = a | (1 << 1)
	fmt.Println(a)
}

func v1() {
	bslice := make([]B, 0, 1)
	bslice = append(bslice, B{})
	for i := 0; i < 5; i++ {
		go func() {
			for k := 0; k < 10; k++ {
				aa := bslice[0]
				at.AddInt64(&aa.age, 1)
			}

		}()
	}
	time.Sleep(time.Second * 10)
	fmt.Println(bslice[0].age)
}

func v2() {
	bslice := make([]Age, 0, 1)
	bslice = append(bslice, &B{})
	for i := 0; i < 5; i++ {
		go func() {
			for k := 0; k < 10; k++ {
				aa := bslice[0]
				aa.Set()
			}

		}()
	}
	time.Sleep(time.Second * 10)
	bslice[0].Get()
}
