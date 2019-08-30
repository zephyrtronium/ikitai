//go:generate go run ./mkenc
//go:generate gofmt -s -w table.go idcs.go

// Package x86enc provides instruction encoding for x86 assembly.
//
// x86enc is based on, and supports all opcodes in,
// golang.org/x/arch/x86/x86asm.
//
package x86enc

import (
	"fmt"
	"io"

	"golang.org/x/arch/x86/x86asm"
	"golang.org/x/xerrors"
)

// A Prog is a list of x86 instructions to assemble.
type Prog struct {
	// Insn is the list of instructions.
	Insn []x86asm.Inst
	// Mode is the processor mode in bits: 16, 32, or 64.
	Mode int
}

// WriteTo assembles the program into the given writer.
func (p Prog) WriteTo(w io.Writer) (n int64, err error) {
	b := make([]byte, 16)
	for k, insn := range p.Insn {
		wn, err := enc(insn, b, p.Mode)
		if err != nil {
			return n, xerrors.Errorf("error encoding %v (insn %d): %w", insn, k, err)
		}
		wn, err = w.Write(b[:wn])
		n += int64(wn)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// Len calculates the length in bytes of the encoded form of the Prog. More
// precisely, Len finds the number of bytes that p.WriteTo will attempt to
// write; it stops if it encounters an erroneous instruction.
func (p Prog) Len() (n int) {
	b := make([]byte, 16)
	for _, insn := range p.Insn {
		wn, err := enc(insn, b, p.Mode)
		if err != nil {
			break
		}
		n += wn
	}
	return n
}

// Verify verifies that all instructions in the Prog are valid and encodeable.
func (p Prog) Verify() error {
	b := make([]byte, 16)
	for _, insn := range p.Insn {
		_, err := enc(insn, b, p.Mode)
		if err != nil {
			return err
		}
	}
	return nil
}

// enc encodes a single insn and its arguments, returning its length in bytes
// or an error if it cannot be encoded.
func enc(insn x86asm.Inst, b []byte, mode int) (n int, err error) {
	k := insn.Op - x86asm.AAA
	if int(k) >= len(table) {
		return 0, InvalidInsnError{insn}
	}
	idcs := tableIdcs[k]
	ops := table[idcs[0]:idcs[1]]
	// Find the instruction that matches the insn.
	var op instruction
	for _, p := range ops {
		
	}
}

// InvalidInsnError is the type of error returned for an invalid instruction.
type InvalidInsnError struct {
	Insn x86asm.Inst
}

func (err InvalidInsnError) Error() string {
	return fmt.Sprintf("invalid instruction: %v", err.Insn)
}
