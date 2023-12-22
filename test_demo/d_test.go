package test_demo

import (
	"fmt"
	"testing"
	"unsafe"
)

func Test_99(t *testing.T) {
	aa := "a你"
	fmt.Println(len(aa))           //4 获取aa 的字节长度
	fmt.Println(unsafe.Sizeof(aa)) // 16
	b := []string{}
	fmt.Println(unsafe.Sizeof(b)) // 16
	defer fmt.Print("A")
	defer fmt.Print("B")
	fmt.Print("c")
	panic("demo")
	defer fmt.Print("D")
}

func Test_string(t *testing.T) {
	a := Age{
		Name: "nihao",
		Age:  16,
	}

	//结构体Age 中Name 的内存地址
	ptr := unsafe.Pointer(&a)
	fmt.Println(*(*string)(ptr))

	acturlStr := unsafe.Add(ptr, 8)

	fmt.Println(*(*int)(acturlStr)) // Name 字段的长度
	agePtr := unsafe.Add(ptr, 16)
	fmt.Println("a 实例的 Age:", *(*int)(agePtr))
	//操作方法2
	//直接获取a 实例的Age 字段 unsafe.Sizeof("") 增加 16字节
	res := (*int)(unsafe.Pointer(uintptr(unsafe.Pointer(&a)) + unsafe.Sizeof("")))
	//修改值
	*res = 8
	fmt.Println(a)
}
