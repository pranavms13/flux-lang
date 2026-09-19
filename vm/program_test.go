package vm_test

import (
	"bytes"
	"encoding/gob"
	"strings"
	"sync"
	"testing"

	"github.com/pranavms13/flux-lang/vm"
)

// TestProgramConcurrentEncodeDecode exercises registration from both public
// entry points; run it alone with -race to check concurrent first use.
func TestProgramConcurrentEncodeDecode(t *testing.T) {
	chunk, _ := compile(t, "concurrent.flux", "let f = fn(x) => x\nprint(f(1))\n")
	// Build an independent payload without calling vm.Encode, so Decode can
	// participate in first use of the registration path too.
	var fixture bytes.Buffer
	if err := gob.NewEncoder(&fixture).Encode(vm.Program{
		Version: vm.ProgramFormatVersion, Chunk: &vm.Chunk{Code: []byte{byte(vm.OpReturn)}},
	}); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var workers sync.WaitGroup
	for i := 0; i < 32; i++ {
		workers.Add(1)
		go func(i int) {
			defer workers.Done()
			<-start
			if i%2 == 0 {
				if _, err := vm.Decode(fixture.Bytes()); err != nil {
					t.Errorf("initial decode: %v", err)
					return
				}
			}
			data, err := vm.Encode(chunk, nil)
			if err != nil {
				t.Errorf("encode: %v", err)
				return
			}
			program, err := vm.Decode(data)
			if err != nil {
				t.Errorf("decode: %v", err)
				return
			}
			out, err := run(t, program.Chunk)
			if err != nil || out != "1\n" {
				t.Errorf("decoded program produced %q, %v", out, err)
			}
		}(i)
	}
	close(start)
	workers.Wait()
}

// Numeric values and binding operands changed in format 2; a prior payload
// must be rejected before any of its instructions are interpreted.
func TestRejectPhase2ProgramFormat(t *testing.T) {
	var encoded bytes.Buffer
	if err := gob.NewEncoder(&encoded).Encode(vm.Program{Version: 1, Chunk: &vm.Chunk{Code: []byte{byte(vm.OpReturn)}, Constants: []any{int(42)}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := vm.Decode(encoded.Bytes()); err == nil || !strings.Contains(err.Error(), "recompile") {
		t.Fatalf("format 1: %v", err)
	}
}
