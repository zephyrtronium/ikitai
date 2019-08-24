// Package unsafewx provides management routines for memory that is either
// writeable or executable.
//
// W^X memory as implemented in package unsafewx is writeable exactly until it
// becomes executable. Once execute permission is added, write permission is
// removed, and there is no way to transition back. This helps prevent certain
// classes of arbitrary code execution attacks.
//
// The "unsafe" part of unsafewx is there because using this package is
// inherently unsafe, as it lets you execute arbitrary code with absolutely no
// safety checks. Despite this, using unsafewx typically doesn't require that
// you import unsafe directly. "unsafewx" is a mnemonic. Be mindful.
//
package unsafewx

import (
	"errors"
	"io"
	"log"
	"reflect"
	"unsafe"
)

// A Block represents a block of writeable or executable memory, or W^X.
type Block struct {
	v    uintptr // pointer to data
	n, c uintptr // len and cap
	x    bool    // executable flag
}

// MustAlloc is like Alloc but panics if the block could not be allocated.
func MustAlloc(n int) *Block {
	b, err := Alloc(n)
	if err != nil {
		panic(err)
	}
	return b
}

// IsValid returns true if the block refers to committed memory.
func (b *Block) IsValid() bool {
	return b != nil && b.v != 0
}

// Available returns the number of unwritten bytes in the block. Panics if the
// block is not valid.
func (b *Block) Available() int {
	if !b.IsValid() {
		panic("wx: use of invalid block")
	}
	return int(b.c - b.n)
}

// Len returns the number of bytes written in the block. Panics if the block is
// not valid.
func (b *Block) Len() int {
	if !b.IsValid() {
		panic("wx: use of invalid block")
	}
	return int(b.n)
}

// Cursor returns the current write-to position in the block. This may be
// useful to keep track of the addresses of function pointers when writing
// multiple functions to the same block. Panics if the block is not valid.
func (b *Block) Cursor() uintptr {
	if !b.IsValid() {
		panic("wx: use of invalid block")
	}
	return b.n
}

// Write writes bytes into the block. If the number of bytes to write exceeds
// the capacity of the block, Write ignores the excess and returns
// CapacityExceeded. Panics if the block is not valid or if b.Exec has
// succeeded.
func (b *Block) Write(p []byte) (n int, err error) {
	if b.x {
		panic("wx: attempted to write to executable memory")
	}
	if len(p) == 0 {
		return 0, nil
	}
	n = len(p)
	if c := b.Available(); n > c {
		// Writing too much data.
		err = CapacityExceeded
		n = c
	}
	memmove(unsafe.Pointer(b.v+b.n), unsafe.Pointer(&p[0]), uintptr(n))
	b.n += uintptr(n)
	return
}

// WriteTo copies out the written contents of the block. This may call w.Write
// multiple times. Panics if the block is not valid.
func (b *Block) WriteTo(w io.Writer) (n int64, err error) {
	const ps = 4096
	bn := uintptr(b.Len())
	if bn <= ps {
		// Avoid wasteful allocation when writing a small block.
		p := make([]byte, bn)
		memmove(unsafe.Pointer(&p[0]), unsafe.Pointer(b.v), bn)
		wn, err := w.Write(p)
		return int64(wn), err
	}
	p := make([]byte, ps)
	var o uintptr
	var wn int
	for bn-o > ps {
		memmove(unsafe.Pointer(&p[0]), unsafe.Pointer(b.v+o), ps)
		o += ps
		wn, err = w.Write(p)
		n += int64(wn)
		if err != nil {
			return
		}
	}
	memmove(unsafe.Pointer(&p[0]), unsafe.Pointer(b.v+o), bn-o)
	wn, err = w.Write(p[:bn-o])
	n += int64(wn)
	return
}

// Func returns a function that executes the code at the given address in the
// block. The function has the type given in the typ parameter. The caller is
// responsible for ensuring that the address points directly to executable code
// that is ABI-compatible with the desired function type, and that the block is
// not closed while the function is executing. Panics if the block is invalid,
// has not been marked executable, or if addr is outside the block's bounds
// (but not if the function leaves the block's bounds; that will result in an
// unrecoverable panic).
func (b *Block) Func(addr uintptr, typ reflect.Type) interface{} {
	if !b.IsValid() {
		panic("wx: attempted to create function without committed memory")
	}
	if !b.x {
		panic("wx: attempted to create function in writeable memory")
	}
	if addr >= b.n {
		panic("wx: function pointer out of bounds")
	}
	// Create a zero value of the function type, then set its pointer unsafely.
	// KEEP IN SYNC WITH reflect.Value:
	// https://github.com/golang/go/blob/master/src/reflect/value.go#L36
	type rvalue struct {
		rtype unsafe.Pointer
		ptr   unsafe.Pointer
		flag  uintptr
	}
	z := reflect.Zero(typ)
	// z.Interface() dereferences the function pointer we use here because in
	// gc, function values (i.e., uses of functions other than by static,
	// package-level names) are pointers to pointers to code. See
	// https://golang.org/s/go11func.
	x := b.v + addr
	(*rvalue)(unsafe.Pointer(&z)).ptr = unsafe.Pointer(&x)
	return z.Interface()
}

// CapacityExceeded is the error returned when attempting to write more data
// than a block can hold.
var CapacityExceeded = errors.New("wx: write exceeded block availability")

// InvalidClose is the error returned when attempting to close a block that is
// nil or already closed.
var InvalidClose = errors.New("wx: close on invalid block")

// Verbose, if non-nil, is used to log every memory operation.
var Verbose *log.Logger

func logv(args ...interface{}) {
	if Verbose != nil {
		Verbose.Println(args...)
	}
}

// memmove is the internal implementation of the copy builtin.
// KEEP IN SYNC WITH runtime.memmove:
// https://github.com/golang/go/blob/master/src/runtime/stubs.go#L88
//go:linkname memmove runtime.memmove
func memmove(to, from unsafe.Pointer, n uintptr)
