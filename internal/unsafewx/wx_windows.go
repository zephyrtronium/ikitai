package unsafewx

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// Alloc allocates a block of W^X memory. Panics if n < 0.
func Alloc(n int) (*Block, error) {
	if n < 0 {
		panic(fmt.Errorf("wx: cannot allocate %d bytes: negative values are illegal", n))
	}
	ps := windows.Getpagesize()
	c := (n + ps - 1) / ps * ps
	logv("allocating", n, "bytes rounded up to", c)
	p, err := windows.VirtualAlloc(0, uintptr(c), windows.MEM_RESERVE|windows.MEM_COMMIT, windows.PAGE_READWRITE)
	if err != nil {
		logv("error during alloc:", err)
		return nil, err
	}
	logv("obtained", c, "bytes at", fmt.Sprintf("%#x", p))
	b := &Block{v: p, c: uintptr(c)}
	return b, nil
}

// Exec marks the block as executable. Following this, any write operations
// panic, and functions assembled within may be called.
func (b *Block) Exec() error {
	logv("marking data at", fmt.Sprintf("%#x", b.v), "with len", b.n, "cap", b.c, "executable")
	var x uint32
	if err := windows.VirtualProtect(b.v, b.c, windows.PAGE_EXECUTE_READ, &x); err != nil {
		logv("error during protect:", err)
		return err
	}
	b.x = true
	// MSDN says we should call FlushInstructionCache to ensure that the CPU
	// sees the new executable memory, but sys/windows doesn't provide that
	// function, and I don't see other JIT examples using it.
	return nil
}

// Close releases the block's memory. Following this, b.IsValid returns false.
func (b *Block) Close() error {
	if !b.IsValid() {
		return ErrInvalidClose
	}
	logv("freeing data at", fmt.Sprintf("%#x", b.v), "with len", b.n, "cap", b.c)
	if err := windows.VirtualFree(b.v, 0, windows.MEM_RELEASE); err != nil {
		logv("error during free:", err)
		return err
	}
	b.v = 0
	return nil
}
