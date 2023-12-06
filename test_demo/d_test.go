package test_demo

import (
	"fmt"
	"testing"
)

func Test_99(t *testing.T) {

	defer fmt.Print("A")
	defer fmt.Print("B")
	fmt.Print("c")
	panic("demo")
	defer fmt.Print("D")
}
