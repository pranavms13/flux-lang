package vm_test

import (
	"bytes"
	"encoding/gob"
	"strings"
	"testing"

	"github.com/pranavms13/flux-lang/compiler"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/fault"
	"github.com/pranavms13/flux-lang/parser"
	"github.com/pranavms13/flux-lang/source"
	"github.com/pranavms13/flux-lang/vm"
)

func compile(t *testing.T, name, text string) (*vm.Chunk, *source.Source) {
	t.Helper()
	result := parser.ParseSource(source.New(1, name, text))
	if result.Failed() {
		t.Fatalf("parse: %v", result.Diagnostics)
	}
	return compiler.NewFluxCompilerForSource(result.Source).Compile(result.Program), result.Source
}

func run(t *testing.T, chunk *vm.Chunk) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := vm.NewWithOutput(chunk, &out).Run()
	return out.String(), err
}

// TestFailureAfterVariableLengthInstructions is the reason the VM captures an
// instruction's offset before reading its operands. Every instruction before
// the failing one here carries a four-byte operand, so a location keyed by the
// instruction pointer after the operands were read would name a later
// instruction, or a byte in the middle of one.
func TestFailureAfterVariableLengthInstructions(t *testing.T) {
	const text = "let a = 1\nlet b = 2\nlet c = 3\nlet xs = [a, b, c]\nprint(xs[\"key\"])\n"
	chunk, src := compile(t, "operands.flux", text)

	_, err := run(t, chunk)
	failure, ok := err.(*fault.Error)
	if !ok {
		t.Fatalf("got %v (%T), want a *fault.Error", err, err)
	}
	if failure.Code != fault.CodeIndexType {
		t.Fatalf("code = %q, want %q", failure.Code, fault.CodeIndexType)
	}
	if got, want := src.Text()[failure.Where.Start:failure.Where.End], `["key"]`; got != want {
		t.Errorf("points at %q, want %q", got, want)
	}
	if failure.Where.Line != 5 {
		t.Errorf("reported on line %d, want 5", failure.Where.Line)
	}
}

// TestSourceMapIsKeyedAtInstructionStarts walks the bytecode and checks that
// every recorded offset is the first byte of an instruction. A key landing
// inside an operand would still look up successfully and report a position for
// the wrong construct.
func TestSourceMapIsKeyedAtInstructionStarts(t *testing.T) {
	const text = `let add = fn(a, b) => a + b
let xs = [1, 2, 3]
let d = {"k": 1}
print(add(xs[0], d["k"]))
print(if true then { 1 } else { 2 })
`
	chunk, _ := compile(t, "boundaries.flux", text)

	var checked int
	var walk func(c *vm.Chunk, name string)
	walk = func(c *vm.Chunk, name string) {
		starts := map[int]bool{}
		for ip := 0; ip < len(c.Code); {
			starts[ip] = true
			ip += 1 + 4*vm.Opcode(c.Code[ip]).Operands()
		}
		if len(c.Locations) == 0 {
			t.Errorf("%s has no source map", name)
		}
		for offset := range c.Locations {
			checked++
			if !starts[offset] {
				t.Errorf("%s: location keyed at offset %d, which is not an instruction start", name, offset)
			}
			if location := c.Locate(offset); !location.IsValid() {
				t.Errorf("%s: offset %d maps to an invalid location", name, offset)
			}
		}
		// Nested functions get their own maps, so a failure inside one reports
		// its own line rather than the line of the call.
		for i, constant := range c.Constants {
			if nested, ok := constant.(*vm.Chunk); ok {
				walk(nested, name+" constant "+string(rune('0'+i)))
			}
		}
	}
	walk(chunk, "top level")
	if checked == 0 {
		t.Fatal("no locations were checked")
	}
}

func TestNestedFunctionChunksCarryTheirOwnNameAndMap(t *testing.T) {
	chunk, _ := compile(t, "nested.flux", "let add = fn(a, b) => a + b\nprint(add(1, 2))\n")

	var nested *vm.Chunk
	for _, constant := range chunk.Constants {
		if candidate, ok := constant.(*vm.Chunk); ok {
			nested = candidate
		}
	}
	if nested == nil {
		t.Fatal("no nested function chunk was emitted")
	}
	if got, want := nested.Label(), "add"; got != want {
		t.Errorf("label = %q, want %q; a call trace would say %q", got, want, got)
	}
	if len(nested.Locations) == 0 {
		t.Error("the nested chunk has no source map of its own")
	}
	if anonymous := (&vm.Chunk{}).Label(); anonymous != fault.Anonymous {
		t.Errorf("an unnamed chunk is labelled %q, want %q", anonymous, fault.Anonymous)
	}
}

func TestStackShortfallIsReportedAsAnInternalDefect(t *testing.T) {
	// OpAdd with nothing on the stack cannot be produced by the compiler, so
	// reaching it means Flux is broken, not the program.
	_, err := run(t, &vm.Chunk{Code: []byte{byte(vm.OpAdd)}})

	failure, ok := err.(*fault.Error)
	if !ok {
		t.Fatalf("got %v (%T), want a *fault.Error", err, err)
	}
	if failure.Code != diagnostic.CodeInternal {
		t.Errorf("code = %q, want %q; a VM defect must not look like a mistake in the program",
			failure.Code, diagnostic.CodeInternal)
	}
}

func TestUnknownOpcodeIsAnInternalDefect(t *testing.T) {
	_, err := run(t, &vm.Chunk{Code: []byte{200}})
	failure, ok := err.(*fault.Error)
	if !ok || failure.Code != diagnostic.CodeInternal {
		t.Errorf("got %v, want an internal failure", err)
	}
}

func TestProgramFormatRoundTripsAndRefusesOtherVersions(t *testing.T) {
	chunk, src := compile(t, "format.flux", "let f = fn(x) => x\nprint(f(1))\n")

	encoded, err := vm.Encode(chunk, map[string]string{src.Name(): src.Text()})
	if err != nil {
		t.Fatal(err)
	}
	program, err := vm.Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if program.Version != vm.ProgramFormatVersion {
		t.Errorf("version = %d, want %d", program.Version, vm.ProgramFormatVersion)
	}
	if got := program.Sources[src.Name()]; got != src.Text() {
		t.Errorf("embedded source = %q, want the program text", got)
	}
	// The nested function chunk has to survive as a *Chunk, not as whatever gob
	// would have guessed from an interface.
	var nested int
	for _, constant := range program.Chunk.Constants {
		if _, ok := constant.(*vm.Chunk); ok {
			nested++
		}
	}
	if nested != 1 {
		t.Errorf("decoded %d nested chunks, want 1", nested)
	}
	out, err := run(t, program.Chunk)
	if err != nil || out != "1\n" {
		t.Errorf("decoded program produced %q, %v", out, err)
	}

	// A program from another format version is refused with an instruction to
	// recompile, not with a decode error a user cannot act on.
	var future bytes.Buffer
	if err := gob.NewEncoder(&future).Encode(vm.Program{
		Version: vm.ProgramFormatVersion + 1,
		Chunk:   chunk,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.Decode(future.Bytes()); err == nil || !strings.Contains(err.Error(), "recompile") {
		t.Errorf("decoding a future format gave %v, want an instruction to recompile", err)
	}

	// A payload with no code at all is refused rather than run as an empty
	// program that silently succeeds.
	var empty bytes.Buffer
	if err := gob.NewEncoder(&empty).Encode(vm.Program{Version: vm.ProgramFormatVersion}); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.Decode(empty.Bytes()); err == nil {
		t.Error("a program containing no code was accepted")
	}
}

func TestOperandTableMatchesWhatTheCompilerEmits(t *testing.T) {
	// Walking a chunk with the operand table must land exactly on its end. If
	// the table is wrong for any opcode the compiler emits, this overshoots.
	chunk, _ := compile(t, "walk.flux", `let f = fn(a) => if a > 0 then { [1, 2] } else { {"k": 1} }
print(f(1))
`)
	ip := 0
	for ip < len(chunk.Code) {
		ip += 1 + 4*vm.Opcode(chunk.Code[ip]).Operands()
	}
	if ip != len(chunk.Code) {
		t.Errorf("walking the chunk ended at %d, want %d", ip, len(chunk.Code))
	}
}
