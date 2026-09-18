package diagnostic

import (
	"fmt"
	"sort"
	"sync"

	"github.com/pranavms13/flux-lang/source"
)

// The code registry backs the diagnostic-code reference: every code a stage can
// emit is declared with a one-line description, so the reference is generated
// from the codes that actually exist rather than maintained by hand beside them.
var (
	registryMu sync.RWMutex
	registry   = map[Code]string{}
)

// Register declares a code and its description, returning the code so it can be
// assigned to a package-level variable:
//
//	var CodeArgumentType = diagnostic.Register("T_ARGUMENT_TYPE",
//		"an argument's type does not match the parameter it is passed to")
//
// It panics on a code without a reserved group prefix or on a duplicate
// declaration. Both are defects in Flux itself and are caught at init time,
// before any program is compiled.
func Register(code Code, description string) Code {
	if _, ok := code.Group(); !ok {
		panic(fmt.Sprintf("diagnostic: code %q does not start with a reserved group prefix", code))
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[code]; exists {
		panic(fmt.Sprintf("diagnostic: code %q is already registered", code))
	}
	registry[code] = description
	return code
}

// Describe returns the registered description of a code.
func Describe(code Code) (string, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	description, ok := registry[code]
	return description, ok
}

// Registered returns every declared code, sorted, so the reference lists them
// in a stable order.
func Registered() []Code {
	registryMu.RLock()
	defer registryMu.RUnlock()
	codes := make([]Code, 0, len(registry))
	for code := range registry {
		codes = append(codes, code)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	return codes
}

// CodeInternal reports a defect in Flux itself. It is the only code declared
// here; each stage declares its own as it is migrated onto diagnostics.
var CodeInternal = Register("X_INTERNAL", "an unexpected failure inside Flux; please report it")

// Internal returns a diagnostic for a defect in Flux itself. The span may be
// [github.com/pranavms13/flux-lang/source.NoSpan] when no construct is to blame.
func Internal(primary source.Span, format string, args ...any) Diagnostic {
	return Error(CodeInternal, primary, format, args...).
		WithNote("this is a bug in Flux, not in the program being compiled")
}
