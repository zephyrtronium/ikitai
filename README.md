# Ikitai - 生きたい！！！

Package ikitai is an optimizing just-in-time compiler for SSA-transformed Go.

Ikitai is intended to allow Go programs to be extensible and hot-swappable in a
Cgo-free environment. Transforming Go to SSA form is already easy; Ikitai
provides the compilation from SSA form to executable functions.

The vision of Ikitai is to become a Go compiler that rivals gc and gccgo in
output code performance and generates code which passes all tests. Part of this
vision is completely compiling the runtime itself.

This project is probably too ambitious.

## Planned Features

- Extensions: Run Go code from any trusted source without needing to compile statically.
- Hot-swapping: Load extension functions, edit the source code, and then load the new implementations.
- Configurable optimization passes: Put vectorization before hoisting. Try really hard to eliminate bounds checks. Inline specific functions. Optimize for execution speed or for output size.
- Live PGO: Save expensive optimizations for code in the hot path. Correctly predict branches. Discover which functions are optimal to inline at each individual call site. Inline hot portions of functions with cold paths. Specialize functions for frequently used interface types.

## Roadmap

- AMD64 assembling
	+ arch/x86/x86asm to machine code
	+ Code generation
		- SSA form to x86asm
		- assembly source to x86asm (possibly a separate module)
		- Include runtime functions
		- Generate DWARF and information for runtime.FuncForPC?
- Optimization
- Additional arches
	+ x86?
	+ arm64?
