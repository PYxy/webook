package test_demo

import (
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"unsafe"
)

//	func Test_slice(t *testing.T) {
//		s := make([]int, 10, 12)
//		v := s[10]
//		fmt.Println(v)
//		// 求问，此时数组访问是否会越界
//	}
//
//	func Test_slice(t *testing.T) {
//		s := make([]int, 10, 12)
//		s1 := s[8:]
//		s1[0] = -1
//		t.Logf("s: %v", s)
//	}
//func Test_slice(t *testing.T) {
//	//[0,0,0,0,0,0,0,0,0,0,0,0]
//	s := make([]int, 10, 12)
//	s1 := s[8:]
//	fmt.Println(len(s1), cap(s1))
//	s1 = append(s1, []int{10, 11, 12}...)
//	v := s[10]
//	fmt.Println(v)
//	// ...
//	// 求问，此时数组访问是否会越界
//}

//func Test_slice(t *testing.T) {
//	s := make([]int, 10, 12)
//	s1 := s[8:]
//	changeSlice(s1)
//	t.Logf("s: %v", s)
//}
//
//func changeSlice(s1 []int) {
//	s1[0] = -1
//}

func Test_slice(t *testing.T) {
	s := make([]int, 10, 12)
	fmt.Println("切片所占长度：", unsafe.Sizeof(s))
	unsafeLen := unsafe.Add(unsafe.Pointer(&s), 8)
	unsafeCap := unsafe.Add(unsafe.Pointer(&s), 16)
	fmt.Println("长度:", *(*int)(unsafeLen))
	fmt.Println("容量:", *(*int)(unsafeCap))
	s1 := s[8:]
	changeSlice(s1)
	fmt.Println("外面。。。")
	t.Logf("s: %v[%p], len of s: %d, cap of s: %d", s, s, len(s), cap(s))
	t.Logf("s1: %v[%p], len of s1: %d, cap of s1: %d", s1, s1, len(s1), cap(s1))
	//sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&s[0])))
	data := unsafe.Add(unsafe.Pointer(&s[0]), 8)
	data2 := unsafe.Add(unsafe.Pointer(&s[0]), 80)

	fmt.Println(data, data2)
	// 索引下标为10的数是date2
	fmt.Println(*(*int)(data), *(*int)(data2))

}

func changeSlice(s1 []int) {
	fmt.Println("新的......")
	fmt.Printf("s1: %v [%p], len of s1: %d, cap of s1: %d \n", s1, s1, len(s1), cap(s1))

	s1 = append(s1, 10)
	fmt.Printf("s1: %v [%p], len of s1: %d, cap of s1: %d \n", s1, s1, len(s1), cap(s1))

}

type Age struct {
	Name string
	Age  int
}

//func Test_string(t *testing.T) {
//	a := Age{
//		Name: "nihao",
//		Age:  16,
//	}
//	//结构体Age 中Name 的内存地址
//	ptr := unsafe.Pointer(&a)
//	fmt.Println(*(*string)(ptr))
//
//	acturlStr := unsafe.Add(ptr, 8)
//	fmt.Println(*(*int)(acturlStr)) // Name 字段的长度
//	agePtr := unsafe.Add(ptr, 16)
//
//	fmt.Println("a 实例的 Age:", *(*int)(agePtr))
//
//	//操作方法2
//	//直接获取a 实例的Age 字段
//	res := (*int)(unsafe.Pointer(uintptr(unsafe.Pointer(&a)) + unsafe.Sizeof("")))
//	//修改值
//	*res = 8
//
//	fmt.Println(a)
//
//}
//import (
//"fmt"
//"sync/atomic"
//"unsafe"
//)

// Defining a struct type L
type myStruct struct {
	a int
	b string
}

//var gvar unsafe.Pointer // 全局变量gvar

func storePtr(s unsafe.Pointer) {
	ptr := &myStruct{a: 10, b: "Hello"} // 创建一个结构体类型的指针
	//oldPtr := unsafe.Pointer(s)

	atomic.StorePointer(&s, unsafe.Pointer(ptr)) // 原子地存储指针类型的值
}

func Test_ptr(t *testing.T) {
	oldStruct := &myStruct{
		a: 15,
	}
	Oldptr := unsafe.Pointer(oldStruct)
	storePtr(Oldptr)
	//ptr := (*myStruct)(atomic.LoadPointer(&gvar)) // 原子地加载指针类型的值

	ptr := (*myStruct)(atomic.LoadPointer(&Oldptr)) // 原子地加载指针类型的值

	fmt.Printf("pointer value: %p\n", ptr)
	fmt.Printf("a: %d, b: %s\n", ptr.a, ptr.b)
}

func test_def1() (r int) {
	i := 1
	defer func() {
		i = i + 1
	}()
	return i
}

func test_def2() (r int) {
	defer func(r int) {
		r = r + 3
	}(r)
	return 2
}

func test_def3() (r int) {
	defer func(r *int) {
		*r = *r + 2
	}(&r)
	return 2
}

func e1() {
	var err error
	defer fmt.Println(err)
	err = errors.New("e1 defer err")
}

func e2() {
	var err error
	defer func() {
		fmt.Println(err)
	}()
	err = errors.New("e2 defer err")
}

func e3() {
	var err error
	defer func(err error) {
		fmt.Println(err)
	}(err)
	err = errors.New("e3 defer err")
}

type Peo interface {
	Eat()
}

type A struct {
	Name string
}

func (a *A) Eat() {
	//TODO implement me
	fmt.Println("a Eat")
}

func NewA(name string) (a Peo) {
	b := &A{Name: name}
	a = b
	defer func() {
		b.Name = "改了"
	}()
	return a
}

type B struct {
	Name string
}

func (a B) Eat() {
	//TODO implement me
	fmt.Println("a Eat")
}

func NewB(name string) (a Peo) {
	b := B{Name: name}
	a = b
	defer func() {
		b.Name = "改了"
	}()
	return a
}

func NewA1(name string) Peo {
	b := &A{Name: name}

	defer func() {
		b.Name = "改了"
	}()
	return b
}

func NewB1(name string) Peo {
	b := B{Name: name}

	defer func() {
		b.Name = "改了"
	}()
	return b
}

func Test_def(t *testing.T) {
	//fmt.Println("实际输出:", test_def1()) //1
	//fmt.Println("实际输出:", test_def2()) // 2

	//fmt.Println("实际输出:", test_def3()) // 4

	//e1() // nil
	//e2() //e2 defer err
	//e3() // nil

	//fmt.Println(NewA("小宏")) // 指针的话&{改了}
	//fmt.Println(NewB("小宏")) //{小宏}
	fmt.Println(NewA1("小宏")) //&{改了}
	fmt.Println(NewB1("小宏")) //{小宏}
}
