package unsafewx

import (
	"fmt"
	"reflect"
)

func Example() {
	code := []byte{
		// func(x, y int) (int, int) {
		// 	return x, y
		// }
		0x48, 0x8b, 0x44, 0x24, 0x08, // MOVQ 0x08(SP), AX
		0x48, 0x89, 0x44, 0x24, 0x18, // MOVQ AX, 0x18(SP)
		0x48, 0x8b, 0x44, 0x24, 0x10, // MOVQ 0x10(SP), AX
		0x48, 0x89, 0x44, 0x24, 0x20, // MOVQ AX, 0x20(SP)
		0xc3, // RET
	}
	b := MustAlloc(len(code))
	defer b.Close()
	b.Write(code)
	b.Exec()
	var f func(int, int) (int, int)
	f = b.Func(0, reflect.TypeOf(f)).(func(int, int) (int, int))
	x, y := f(1, 2)
	fmt.Println(x, y)
	// Output: 1 2
}
