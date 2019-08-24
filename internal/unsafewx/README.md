# unsafewx

Package unsafewx provides management routines for memory that is either
writeable or executable. The "unsafe" part is there to remind you that any use
of unsafewx is unsafe, even though it typically doesn't require you to import
`unsafe` yourself.

Allocate a `Block` using `unsafewx.Alloc` or `unsafewx.MustAlloc`. The amount
of memory actually allocated is always rounded up to a multiple of the page
size. `Block`s implement the `io` interfaces `Writer` to write machine code to
the block, `WriterTo` to transfer out the contents of the block, and `Closer`
to free the memory. The backing memory is not garbage collected; losing all
references to a block without having called its `Close` method is a memory
leak, just like doing the same with `os.File` is a file descriptor leak.
However, a function obtained from a block can also spawn goroutines that use
the block's code, so it is impossible to know when a block is no longer in use
in the general case.

Once you're ready to execute memory in a Block, call its `Exec` method. After
doing so, any `Write` calls will panic, not return an error - trying to write
to executable memory is a programmer error, not a program error. If `Exec`
succeeds, you can call `Func` to obtain a function value. It is your
responsibility to ensure that the function address you provide points directly
to the beginning of a procedure that is fully ABI-compatible with the desired
function type, and that the block is not `Close`d while the returned function
is executing.

Blocks are not synchronized. Calling any of their methods from multiple
goroutines requires explicit synchronization mechanisms. The exception to this
is that any number of goroutines may obtain functions from the block, as long
as `Exec` was synchronized before doing so and `Close` is not called until
after all goroutines have finished using the functions obtained from the block.

Incorrect use of unsafewx is characterized by a unique ability to cause
unrecoverable panics in the best case scenario. Take care. 🙂

## Supported Platforms

Currently, unsafewx should work on any platform that Go targets, except plan9.
It doesn't appear that plan9 provides the mechanisms for W^X memory, so it
probably will not be supported in the future.

It wouldn't be unwarranted to add more versions of `func Example` in
`wx_amd64_test.go` for more arches, but otherwise, unsafewx itself works
regardless of the value of `$GOARCH`.
