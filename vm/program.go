package vm

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

// ProgramFormatVersion is the version of the serialized program format.
//
// A generated executable carries a program compiled by whichever Flux built it.
// Without a version, a format change turns into a confusing decode error or,
// worse, a silently misread chunk. Adding a field that old readers can ignore
// does not change this number; removing or repurposing one does.
const ProgramFormatVersion = 1

// Program is what a compiled Flux program serializes to.
type Program struct {
	// Version is ProgramFormatVersion at the time of encoding.
	Version int
	// Chunk is the top-level bytecode.
	Chunk *Chunk
	// Sources is the text of the program's sources, keyed by the display
	// filename its locations use. It is present only when the program was
	// compiled with compiler.debug, and lets a generated executable print the
	// offending line rather than only its position.
	//
	// It makes the executable larger and puts the program's source inside it,
	// which is a disclosure a user should choose deliberately.
	Sources map[string]string
}

// registerGobTypes registers the concrete types that appear behind interfaces
// in a serialized program. Constants are an []interface{}, and a nested
// function is a *Chunk inside one, so gob has to be told the name to use.
//
// The names are explicit rather than derived from the Go import path, so moving
// a package does not invalidate every executable built before the move.
var registerGobTypes = func() func() {
	var once bool
	return func() {
		if once {
			return
		}
		once = true
		gob.RegisterName("flux.Chunk", &Chunk{})
	}
}()

// Encode serializes a program, stamping it with the current format version.
func Encode(chunk *Chunk, sources map[string]string) ([]byte, error) {
	registerGobTypes()
	var buffer bytes.Buffer
	program := Program{Version: ProgramFormatVersion, Chunk: chunk, Sources: sources}
	if err := gob.NewEncoder(&buffer).Encode(program); err != nil {
		return nil, fmt.Errorf("encode program: %w", err)
	}
	return buffer.Bytes(), nil
}

// Decode deserializes a program, refusing a format it does not understand
// rather than guessing at it.
func Decode(data []byte) (*Program, error) {
	registerGobTypes()
	var program Program
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&program); err != nil {
		return nil, fmt.Errorf("decode program: %w", err)
	}
	if program.Version != ProgramFormatVersion {
		return nil, fmt.Errorf(
			"this program uses Flux bytecode format %d, but this build understands format %d; recompile it",
			program.Version, ProgramFormatVersion)
	}
	if program.Chunk == nil {
		return nil, fmt.Errorf("decode program: it contains no code")
	}
	return &program, nil
}
