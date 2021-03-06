// mkenc generates the encoding table for x86enc.
package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"golang.org/x/xerrors"
)

const hdr = `package x86enc

// Code generated by go generate; DO NOT EDIT
`

const insns = `
type instruction struct {
	op, a1, a2, a3, a4, encoding, valid32, valid64, feature, tags string
}

var table = [...]instruction{
`

const idcs = `
var tableIdcs = [...][2]int16{
`

func main() {
	resp, err := http.Get("https://raw.githubusercontent.com/golang/arch/master/x86/x86.csv")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	out, err := os.Create("table.go")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	iut, err := os.Create("idcs.go")
	if err != nil {
		panic(err)
	}
	defer iut.Close()
	must(out.WriteString(hdr + insns))
	must(iut.WriteString(hdr + idcs))
	defer iut.WriteString("}\n")
	defer out.WriteString("}\n")
	r := csv.NewReader(resp.Body)
	r.Comment = '#'
	r.FieldsPerRecord = 6
	cur := ""
	l := [][]string{}
	n := 0
	defer func() {
		args := make([]string, 0, len(argtypes))
		for arg := range argtypes {
			args = append(args, arg)
		}
		sort.Strings(args)
		for _, arg := range args {
			fmt.Fprintln(os.Stderr, arg)
		}
	}()
	for {
		insn, err := r.Read()
		switch {
		default:
			op, a1, a2, a3, a4 := getop(insn[0])
			if op != cur {
				n = emit(out, iut, l, n)
				l = l[:0]
				cur = op
			}
			l = append(l, append([]string{op, a1, a2, a3, a4}, insn[1:]...))
		case xerrors.Is(err, io.EOF):
			emit(out, iut, l, n)
			return
		case xerrors.Is(err, csv.ErrFieldCount):
			continue
		case err != nil:
			panic(err)
		}
	}
}

func getop(m string) (op, a1, a2, a3, a4 string) {
	f := strings.FieldsFunc(m, func(c rune) bool {
		return c == ' ' || c == ','
	})
	switch len(f) {
	case 5:
		a4 = f[4]
		fallthrough
	case 4:
		// VPAND has a typo in the manual where "ymm3/m256" is instead
		// "ymm3/.m256". No other insns have "." in the argument fields.
		a3 = strings.Replace(f[3], ".", "", -1)
		if strings.HasPrefix(a4, "/") {
			// VPCMPEQ[DQW] have some incorrect forms where "ymm3/m256" is
			// split into "ymm3","m256".
			a3 += a4
			a4 = ""
		}
		fallthrough
	case 3:
		a2 = f[2]
		fallthrough
	case 2:
		a1 = f[1]
		fallthrough
	case 1:
		op = f[0]
	default:
		panic(m)
	}
	return
}

func emit(out, iut io.Writer, l [][]string, n int) int {
	if len(l) == 0 {
		return n
	}
	o := make([]string, len(l))
	args := make([]string, 0, 4)
	var b strings.Builder
	ok := false
	for i, insn := range l {
		if !ok && !strings.Contains(insn[len(insn)-1], "pseudo") {
			// We want to remove pseudo-opcodes, but only if all insns with
			// this op are pseudo. Perhaps more precisely, we still want 1-byte
			// NOP.
			ok = true
		}
		b.WriteString("instruction{")
		for _, s := range insn {
			b.WriteString(fmt.Sprintf("%q,", s))
		}
		b.WriteByte('}')
		o[i] = b.String()
		b.Reset()
		args = append(args, insn[1:5]...)
	}
	if ok {
		for _, arg := range args {
			argtypes[arg] = true
		}
		must(fmt.Fprintf(out, "\t%s,\n", strings.Join(o, ", ")))
		must(fmt.Fprintf(iut, "\t[2]int16{%d, %d},\n", n, n+len(l)))
		return n + len(l)
	}
	return n
}

var argtypes = map[string]bool{}

func must(_ int, err error) {
	if err != nil {
		panic(err)
	}
}
