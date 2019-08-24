package unsafewx

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"unsafe"
)

// TestClose tests that allocated blocks can be closed exactly once.
func TestClose(t *testing.T) {
	cases := []int{4 << 10, 8 << 10, 32 << 10, 8 << 20}
	for _, c := range cases {
		t.Run(fmt.Sprint(c), func(t *testing.T) {
			b := MustAlloc(c)
			if err := b.Close(); err != nil {
				t.Errorf("error while closing: %v", err)
			}
			if err := b.Close(); err == nil {
				t.Error("unexpected successful close")
			}
		})
	}
}

// TestWrite tests that data can be written correctly to a block.
func TestWrite(t *testing.T) {
	cases := []int{1, 4 << 10, 8 << 10, 8 << 20}
	for _, c := range cases {
		t.Run(fmt.Sprint(c), func(t *testing.T) {
			a := make([]byte, c)
			rand.Read(a)
			b := MustAlloc(c)
			defer b.Close()
			n, err := b.Write(a)
			if err != nil {
				t.Error(err)
			}
			if n != c {
				t.Errorf("wrote wrong number of bytes: wanted %d, have %d", c, n)
			}
			for i := 0; i < n; i++ {
				x := *(*byte)(unsafe.Pointer(b.v + uintptr(i)))
				if x != a[i] {
					t.Errorf("wrong value written at position %d: wanted %d, have %d", i, a[i], x)
				}
			}
		})
	}
}

// TestWriteMulti tests that multiple consecutive writes to a block work.
func TestWriteMulti(t *testing.T) {
	// These cases are the number of 1 kB writes, not the number of bytes like
	// in other tests.
	cases := []int{4, 8, 8 << 10}
	for _, c := range cases {
		t.Run(fmt.Sprint(c, "*1024"), func(t *testing.T) {
			a := make([]byte, c<<10)
			aa := a
			rand.Read(a)
			b := MustAlloc(c << 10)
			defer b.Close()
			var n int
			for len(aa) > 0 {
				wn, err := b.Write(aa[:1<<10])
				if err != nil {
					t.Error(err)
				}
				if wn != 1<<10 {
					t.Errorf("wrote wrong number of bytes: wanted 1024, have %d", n)
				}
				aa = aa[1<<10:]
				n += wn
			}
			if n != len(a) {
				t.Errorf("wrote wrong total number of bytes: wanted %d, have %d", len(a), n)
			}
		})
	}
}

// TestWriteTooMuch tests that attempting to write more data to a block than it
// can hold results in an error.
func TestWriteTooMuch(t *testing.T) {
	// NOTE: It is assumed that these test cases are larger than the page size.
	// They will fail with unexpected successful writes if that is not the
	// case.
	cases := []int{8 << 10, 8 << 20}
	for _, c := range cases {
		t.Run(fmt.Sprint(c), func(t *testing.T) {
			a := make([]byte, c+1<<10)
			rand.Read(a)
			b := MustAlloc(c)
			defer b.Close()
			n, err := b.Write(a)
			if err == nil {
				t.Error("expected error, got nil")
			}
			if n != c {
				t.Errorf("wrote wrong number of bytes: wanted %d, have %d", c, n)
				if n > c {
					// Make sure we don't try to read outside our allocated
					// memory in the upcoming correctness check.
					n = c
				}
			}
			for i := 0; i < n; i++ {
				x := *(*byte)(unsafe.Pointer(b.v + uintptr(i)))
				if x != a[i] {
					t.Errorf("wrong value written at position %d: wanted %d, have %d", i, a[i], x)
				}
			}
		})
	}
}

// TestWriteToExec tests that attempting to write to an executable block causes
// a panic.
func TestWriteToExec(t *testing.T) {
	b := MustAlloc(8 << 10)
	defer b.Close()
	if err := b.Exec(); err != nil {
		t.Fatalf("b.Exec failed: %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Error("writing to executable memory did not panic")
		}
	}()
	b.Write([]byte{0: 0})
}

// TestWriteTo tests that data written to a block can be read out of the block.
func TestWriteTo(t *testing.T) {
	cases := []int{1, 4 << 10, 8 << 10, 8 << 20}
	for _, c := range cases {
		t.Run(fmt.Sprint(c), func(t *testing.T) {
			a := make([]byte, c)
			rand.Read(a)
			b := MustAlloc(c)
			defer b.Close()
			if _, err := b.Write(a); err != nil {
				t.Fatalf("error filling block: %v", err)
			}
			var w bytes.Buffer
			n, err := b.WriteTo(&w)
			if err != nil {
				t.Error(err)
			}
			if n != int64(c) {
				t.Errorf("wrote wrong number of bytes: wanted %d, have %d", c, n)
			}
			for i := 0; i < int(n); i++ {
				x := *(*byte)(unsafe.Pointer(b.v + uintptr(i)))
				if x != a[i] {
					t.Errorf("wrong value written at position %d: wanted %d, have %d", i, x, a[i])
				}
			}
		})
	}
}

// TestFunc tests that a block can return a function of an arbitrary type. It
// does not attempt to call those functions.
func TestFunc(t *testing.T) {
	var f func(int, int) (int, int)
	var g = (*Block).Func
	b := MustAlloc(1)
	defer b.Close()
	if _, err := b.Write([]byte{0xc3}); err != nil {
		t.Fatalf("unable to write a byte: %v", err)
	}
	if err := b.Exec(); err != nil {
		t.Fatal("unable to exec block")
	}
	f = b.Func(0, reflect.TypeOf(f)).(func(int, int) (int, int))
	if f == nil {
		t.Error("creating func(int, int) (int, int) gave nil function")
	}
	g = b.Func(0, reflect.TypeOf(g)).(func(*Block, uintptr, reflect.Type) interface{})
	if g == nil {
		t.Error("creating func(*Block, uintptr, reflect.Type) interface{} gave nil function")
	}
}
