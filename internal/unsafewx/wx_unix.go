// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package unsafewx

import (
	"fmt"
	"reflect"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Alloc allocates a block of W^X memory. Panics if n < 0.
func Alloc(n int) (*Block, error) {
	if n < 0 {
		panic(fmt.Errorf("wx: cannot allocate %d bytes: negative values are illegal", n))
	}
	ps := unix.Getpagesize()
	c := (n + ps - 1) / ps * ps
	if c == 0 {
		// It is crucial that we do not try to mmap zero bytes, because Mmap
		// uses a special region for zero-byte allocations, and we don't want
		// to change its protections.
		c = ps
	}
	logv("allocating", n, "bytes rounded up to", c)
	v, err := unix.Mmap(-1, 0, c, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANONYMOUS)
	if err != nil {
		logv("error during alloc:", err)
		return nil, err
	}
	// Mmap returns a slice over the memory we requested, but we want the
	// actual pointer, mostly for Windows compatibility. It should be safe (in
	// the garbage collection sense) for us to unwrap the slice because Mmap
	// keeps it alive in a private map.
	p := (*(*reflect.SliceHeader)(unsafe.Pointer(&v))).Data
	logv("obtained", c, "bytes at", fmt.Sprintf("%#x", p))
	return &Block{v: p, c: uintptr(c)}, nil
}

// Exec marks the block as executable. Following this, any write operations
// panic, and functions assembled within may be called.
func (b *Block) Exec() error {
	logv("marking data at", fmt.Sprintf("%#x", b.v), "with len", b.n, "cap", b.c, "executable")
	// While converting from pointer-to-slice to pointer-to-reflect.SliceHeader
	// is among the valid use cases for unsafe.Pointer, the documentation for
	// unsafe says not to create SliceHeader values. Oh well.
	if err := unix.Mprotect(*(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{Data: b.v, Len: int(b.c), Cap: int(b.c)})), unix.PROT_READ|unix.PROT_EXEC); err != nil {
		logv("error during protect:", err)
		return err
	}
	b.x = true
	return nil
}

// Close releases the block's memory. Following this, b.IsValid returns false.
func (b *Block) Close() error {
	if !b.IsValid() {
		return ErrInvalidClose
	}
	logv("freeing data at", fmt.Sprintf("%#x", b.v), "with len", b.n, "cap", b.c)
	if err := unix.Munmap(*(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{Data: b.v, Len: int(b.c), Cap: int(b.c)}))); err != nil {
		logv("error during free:", err)
		return err
	}
	b.v = 0
	return nil
}
