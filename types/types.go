package types

import (
	"fmt"
	"strings"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/diagnostic"
	"github.com/pranavms13/flux-lang/resolver"
	"github.com/pranavms13/flux-lang/source"
)

// FluxType represents a type in the Flux language
type FluxType interface {
	String() string
	Equals(other FluxType) bool
}

// Basic types
type (
	IntType    struct{}
	StringType struct{}
	BoolType   struct{}
	VoidType   struct{}
)

func (IntType) String() string    { return "int" }
func (StringType) String() string { return "string" }
func (BoolType) String() string   { return "bool" }
func (VoidType) String() string   { return "void" }

func (t IntType) Equals(other FluxType) bool    { _, ok := other.(IntType); return ok }
func (t StringType) Equals(other FluxType) bool { _, ok := other.(StringType); return ok }
func (t BoolType) Equals(other FluxType) bool   { _, ok := other.(BoolType); return ok }
func (t VoidType) Equals(other FluxType) bool   { _, ok := other.(VoidType); return ok }

// Composite types
type ListType struct {
	ElementType FluxType
}

func (t ListType) String() string {
	return fmt.Sprintf("[%s]", t.ElementType.String())
}

func (t ListType) Equals(other FluxType) bool {
	if otherList, ok := other.(ListType); ok {
		return TypesEqual(t.ElementType, otherList.ElementType)
	}
	return false
}

type DictType struct {
	KeyType   FluxType
	ValueType FluxType
}

func (t DictType) String() string {
	return fmt.Sprintf("{%s: %s}", t.KeyType.String(), t.ValueType.String())
}

func (t DictType) Equals(other FluxType) bool {
	if otherDict, ok := other.(DictType); ok {
		return TypesEqual(t.KeyType, otherDict.KeyType) && TypesEqual(t.ValueType, otherDict.ValueType)
	}
	return false
}

// ParamInfo records where a parameter was declared, so that a diagnostic about
// an argument can point back at the parameter it disagrees with.
//
// It is provenance, not part of the type: [FunctionType.Equals] ignores it, and
// it is empty for a built-in or for a function type that came from an
// annotation rather than from a function literal.
type ParamInfo struct {
	Name string
	Span source.Span
}

type FunctionType struct {
	ParamTypes []FluxType
	ReturnType FluxType
	Params     []ParamInfo
}

// Param returns the declaration of the i-th parameter, and false when the
// function's type carries no record of where its parameters were written.
func (t FunctionType) Param(i int) (ParamInfo, bool) {
	if i < 0 || i >= len(t.Params) {
		return ParamInfo{}, false
	}
	return t.Params[i], t.Params[i].Span.IsValid()
}

func (t FunctionType) String() string {
	params := make([]string, len(t.ParamTypes))
	for i, p := range t.ParamTypes {
		params[i] = p.String()
	}
	return fmt.Sprintf("fn(%s) -> %s", strings.Join(params, ", "), t.ReturnType.String())
}

func (t FunctionType) Equals(other FluxType) bool {
	if otherFunc, ok := other.(FunctionType); ok {
		if len(t.ParamTypes) != len(otherFunc.ParamTypes) {
			return false
		}
		for i, param := range t.ParamTypes {
			if !TypesEqual(param, otherFunc.ParamTypes[i]) {
				return false
			}
		}
		return TypesEqual(t.ReturnType, otherFunc.ReturnType)
	}
	return false
}

// Add a new type for unknown/inferred types
type UnknownType struct{}

func (UnknownType) String() string { return "unknown" }
func (t UnknownType) Equals(other FluxType) bool {
	// Unknown type is compatible with any type during inference
	return true
}

func isUnknown(t FluxType) bool {
	_, ok := t.(UnknownType)
	return ok
}

// TypesEqual provides symmetric type equality checking.
// It returns true if either a.Equals(b) or b.Equals(a) is true.
// This fixes the non-symmetric equality relation caused by UnknownType.
func TypesEqual(a, b FluxType) bool {
	return a.Equals(b) || b.Equals(a)
}

// Type environment for variable bindings
type TypeEnv struct {
	bindings map[string]FluxType
	parent   *TypeEnv
}

func NewTypeEnv(parent *TypeEnv) *TypeEnv {
	return &TypeEnv{
		bindings: make(map[string]FluxType),
		parent:   parent,
	}
}

func (env *TypeEnv) Bind(name string, t FluxType) {
	env.bindings[name] = t
}

func (env *TypeEnv) Lookup(name string) (FluxType, bool) {
	if t, ok := env.bindings[name]; ok {
		return t, true
	}
	if env.parent != nil {
		return env.parent.Lookup(name)
	}
	return nil, false
}

// Type checker
type TypeChecker struct {
	env                *TypeEnv
	diagnostics        diagnostic.Bag
	config             TypeCheckingMode
	bindings           *resolver.Result
	bindingTypes       map[resolver.ID]FluxType
	signatures         map[*ast.FuncExpr]FunctionType
	signatureLocations map[*ast.FuncExpr]ast.Positioned
	// source and sourceID locate the diagnostics. They are unset for a checker
	// built without a source, in which case diagnostics carry codes and
	// messages but no location.
	source   *source.Source
	sourceID source.SourceID
}

// TypeCheckingMode controls how strict the type checker is
type TypeCheckingMode struct {
	Strict   bool
	WarnOnly bool
	Enabled  bool
}

func NewTypeChecker() *TypeChecker {
	return NewTypeCheckerWithConfig(TypeCheckingMode{
		Strict:   false,
		WarnOnly: false,
		Enabled:  true,
	})
}

// NewTypeCheckerWithConfig creates an isolated checker with the selected mode
// and the built-in print signature.
func NewTypeCheckerWithConfig(mode TypeCheckingMode) *TypeChecker {
	tc := &TypeChecker{config: mode}
	tc.reset()
	return tc
}

// printType is the built-in signature for print, which accepts any one value.
func printType() FunctionType {
	return FunctionType{ParamTypes: []FluxType{UnknownType{}}, ReturnType: VoidType{}}
}

// reset returns the checker to the state a fresh one would have, keeping only
// its mode and its source. Everything else is per program: resolver IDs
// restart at PrintID for each program, so a bindingTypes entry held over from
// an earlier one would answer for a binding it never described, and retained
// diagnostics would be reported against a program that did not produce them.
//
// It is called at the start of every CheckProgram rather than left to the
// caller, so that reusing a checker matches reusing a compiler or a runtime.
func (tc *TypeChecker) reset() {
	env := NewTypeEnv(nil)
	env.Bind("print", printType())

	tc.env = env
	tc.diagnostics = diagnostic.Bag{}
	tc.bindings = nil
	tc.bindingTypes = map[resolver.ID]FluxType{resolver.PrintID: printType()}
	tc.signatures = map[*ast.FuncExpr]FunctionType{}
	tc.signatureLocations = map[*ast.FuncExpr]ast.Positioned{}
}

// NewTypeCheckerForSource returns a checker whose diagnostics are located in
// src. The nodes it is given must have been parsed from that snapshot.
func NewTypeCheckerForSource(src *source.Source, mode TypeCheckingMode) *TypeChecker {
	tc := NewTypeCheckerWithConfig(mode)
	tc.source = src
	tc.sourceID = src.ID()
	return tc
}

// Type checking methods
func (tc *TypeChecker) CheckProgram(prog *ast.Program) {
	tc.reset()
	tc.bindings = resolver.Resolve(prog, tc.source)
	for _, d := range tc.bindings.Diagnostics {
		tc.diagnostics.Add(d)
	}
	if len(tc.bindings.Diagnostics) > 0 {
		return
	}
	if !tc.config.Enabled {
		return
	}
	for _, stmt := range prog.Statements {
		tc.CheckStatement(stmt)
	}
}

// CheckStatement checks an expression or validates a binding against its
// annotation before storing its type.
func (tc *TypeChecker) CheckStatement(stmt *ast.Statement) {
	if stmt.Let != nil {
		id := tc.bindings.Declarations[stmt.Let]
		if fn := ast.FunctionLiteral(stmt.Let.Expr); fn != nil {
			signature, complete := tc.functionSignature(stmt.Let, fn)
			tc.bindingTypes[id] = signature
			tc.signatures[fn] = signature
			if stmt.Let.TypeAnno != nil {
				tc.signatureLocations[fn] = stmt.Let.TypeAnno
			}
			if tc.bindings.Recursive[stmt.Let] && !complete {
				tc.errorAt(always, CodeRecursiveSignature, stmt.Let, "recursive function %s requires every parameter and its return type to be annotated", stmt.Let.Name)
			}
		}
		exprType := tc.CheckExpr(stmt.Let.Expr)

		// Check if there's a type annotation
		if stmt.Let.TypeAnno != nil {
			annotatedType, err := ConvertASTType(stmt.Let.TypeAnno.Type)
			if err != nil {
				tc.errorAt(always, CodeInvalidAnnotation, stmt.Let.TypeAnno, "invalid type annotation: %v", err)
				return
			}

			// The mistake is in the value, so that is where the diagnostic
			// points; the annotation it disagrees with is attached to it.
			if !TypesEqual(exprType, annotatedType) {
				tc.reportWithRelated(always, CodeAnnotationMismatch, stmt.Let.Expr,
					tc.span(stmt.Let.TypeAnno), "%s is declared as %s here",
					[]any{stmt.Let.Name, annotatedType.String()},
					"type mismatch: variable %s declared as %s but assigned %s",
					stmt.Let.Name, annotatedType.String(), exprType.String())
			}

			// Use the annotated type for binding
			tc.env.Bind(stmt.Let.Name, annotatedType)
			tc.bindingTypes[id] = annotatedType
		} else {
			// Use inferred type
			tc.env.Bind(stmt.Let.Name, exprType)
			tc.bindingTypes[id] = exprType
		}
	} else if stmt.Expr != nil {
		tc.CheckExpr(stmt.Expr)
	}
}

// CheckExpr returns an expression type and records diagnostics for invalid
// constructs.
func (tc *TypeChecker) CheckExpr(expr *ast.Expr) FluxType {
	switch {
	case expr.If != nil:
		return tc.CheckIfExpr(expr.If)
	case expr.Bin != nil:
		return tc.CheckBinaryExpr(expr.Bin)
	case expr.Func != nil:
		return tc.CheckFuncExpr(expr.Func)
	default:
		tc.report(always, diagnostic.Internal(tc.span(expr), "expression matched no grammar alternative"))
		return VoidType{}
	}
}

// CheckIfExpr checks the condition and branch types, applying the configured
// tolerance for non-boolean conditions and differing branches.
func (tc *TypeChecker) CheckIfExpr(ifExpr *ast.IfExpr) FluxType {
	condType := tc.CheckExpr(ifExpr.Cond)
	if !TypesEqual(condType, BoolType{}) && !isUnknown(condType) {
		d := diagnostic.Error(CodeConditionType, tc.span(ifExpr.Cond),
			"if condition must be bool, got %s", condType.String())
		tc.report(always, d)
	}

	thenType := tc.CheckExpr(ifExpr.ThenExpr)
	elseType := tc.CheckExpr(ifExpr.ElseExpr)

	if !TypesEqual(thenType, elseType) && !isUnknown(thenType) && !isUnknown(elseType) {
		// The else branch is where the disagreement is noticed, and the then
		// branch is what it disagrees with.
		d := diagnostic.Error(CodeBranchMismatch, tc.span(ifExpr.ElseExpr),
			"if branches must have same type: then=%s, else=%s",
			thenType.String(), elseType.String())
		if span := tc.span(ifExpr.ThenExpr); span.IsValid() {
			d = d.WithRelated(span, "the then branch produces %s", thenType.String())
		}
		if tc.config.Strict {
			tc.report(strictOnly, d)
			return VoidType{}
		}
		// Outside strict mode the result is unknown rather than void, which is
		// existing inference behavior that Phase 4 revisits.
		tc.report(strictOnly, d.WithNote("using unknown type"))
		return UnknownType{}
	}

	return thenType
}

// checkValue traverses precedence levels while checking both logical operands,
// including the statically skipped side of a short-circuit expression.
func (tc *TypeChecker) checkValue(n ast.Positioned) FluxType {
	if left, rest := ast.Chain(n); left != nil {
		result := tc.checkValue(left)
		leftSpan := tc.span(left)
		for _, op := range rest {
			right := tc.checkValue(op.Right)
			result = tc.checkOperator(op.Operator, result, right, tc.span(op.At), leftSpan, tc.span(op.Right))
			leftSpan = leftSpan.Union(tc.span(op.At))
		}
		return result
	}
	switch n := n.(type) {
	case *ast.Unary:
		if n.Primary != nil {
			return tc.CheckPrimaryExpr(n.Primary)
		}
		actual := tc.checkValue(n.Operand)
		var want FluxType = IntType{}
		if n.Operator == "!" {
			want = BoolType{}
		}
		if !TypesEqual(actual, want) {
			tc.errorAt(always, CodeOperandType, n, "invalid operand for %s: %s", n.Operator, actual.String())
		}
		return want
	case *ast.PrimaryExpr:
		return tc.CheckPrimaryExpr(n)
	}
	tc.report(always, diagnostic.Internal(tc.span(n), "invalid precedence node"))
	return UnknownType{}
}
func (tc *TypeChecker) CheckBinaryExpr(n *ast.Binary) FluxType { return tc.checkValue(n) }

// checkOperator reports on an operator application. opSpan covers the operator
// and its right operand, which is where the node begins; leftSpan covers
// everything the operator is applied to on the left, so that a diagnostic can
// name both operands.
func (tc *TypeChecker) checkOperator(operator string, left, right FluxType,
	opSpan, leftSpan, rightSpan source.Span) FluxType {
	switch operator {
	case "+":
		if isUnknown(left) && isUnknown(right) {
			return UnknownType{}
		}
		if TypesEqual(left, IntType{}) && TypesEqual(right, IntType{}) {
			return IntType{}
		}
		if TypesEqual(left, StringType{}) && TypesEqual(right, StringType{}) {
			return StringType{}
		}
	case "-", "*", "/", "%", "<", ">", "<=", ">=":
		if TypesEqual(left, IntType{}) && TypesEqual(right, IntType{}) {
			if operator == "-" || operator == "*" || operator == "/" || operator == "%" {
				return IntType{}
			}
			return BoolType{}
		}
	case "&&", "||":
		if TypesEqual(left, BoolType{}) && TypesEqual(right, BoolType{}) {
			return BoolType{}
		}
	case "==", "!=":
		if !comparableType(left) || !comparableType(right) {
			tc.report(always, diagnostic.Error(CodeIncomparable, leftSpan.Union(opSpan), "function values and containers holding functions cannot be compared"))
			return BoolType{}
		}
		if !TypesEqual(left, right) {
			d := diagnostic.Error(CodeComparisonMismatch, leftSpan.Union(opSpan),
				"cannot compare different types: %s and %s", left.String(), right.String())
			if rightSpan.IsValid() {
				d = d.WithRelated(rightSpan, "this operand is %s", right.String())
			}
			if !tc.config.Strict {
				d = d.WithNote("allowing comparison")
			}
			tc.report(strictOnly, d)
		}
		return BoolType{}
	}
	d := diagnostic.Error(CodeOperandType, leftSpan.Union(opSpan),
		"invalid operands for %s: %s and %s", operator, left.String(), right.String())
	if rightSpan.IsValid() {
		d = d.WithRelated(rightSpan, "this operand is %s", right.String())
	}
	tc.report(always, d)
	return UnknownType{}
}

func (tc *TypeChecker) CheckBlockExpr(blockExpr *ast.BlockExpr) FluxType {
	var lastType FluxType = VoidType{}
	oldEnv := tc.env
	tc.env = NewTypeEnv(oldEnv)
	defer func() { tc.env = oldEnv }()
	for _, stmt := range blockExpr.Statements {
		if stmt.Let != nil {
			tc.CheckStatement(stmt)
			lastType = VoidType{}
		} else {
			lastType = tc.CheckExpr(stmt.Expr)
		}
	}
	return lastType
}

// CheckPrimaryExpr checks a base and its postfix operations, extending the
// callee or indexed-value span at each step.
func (tc *TypeChecker) CheckPrimaryExpr(primary *ast.PrimaryExpr) FluxType {
	var baseType FluxType

	if primary.Base != nil {
		baseType = tc.CheckBaseExpr(primary.Base)
	}

	// Apply postfixes. The span of what is being called or indexed grows with
	// each one, so that a diagnostic points at the whole callee rather than at
	// the identifier that started it.
	currentType := baseType
	currentSpan := tc.span(primary.Base)
	for _, postfix := range primary.Postfix {
		if postfix.Call != nil {
			currentType = tc.CheckCallExpr(currentType, currentSpan, postfix.Call)
		} else if postfix.Index != nil {
			currentType = tc.CheckIndexExpr(currentType, currentSpan, postfix.Index)
		}
		currentSpan = currentSpan.Union(tc.span(postfix))
	}

	return currentType
}

// CheckBaseExpr checks the term, group, block, or collection underlying a
// primary expression.
func (tc *TypeChecker) CheckBaseExpr(base *ast.BaseExpr) FluxType {
	if base.Term != nil {
		return tc.CheckTerm(base.Term)
	} else if base.Group != nil {
		return tc.CheckExpr(base.Group.Expr)
	} else if base.Block != nil {
		return tc.CheckBlockExpr(base.Block)
	} else if base.List != nil {
		return tc.CheckListExpr(base.List)
	} else if base.Dict != nil {
		return tc.CheckDictExpr(base.Dict)
	}

	tc.report(always, diagnostic.Internal(tc.span(base), "base expression matched no grammar alternative"))
	return VoidType{}
}

// CheckTerm returns a literal or bound name type, reporting unknown names at
// their use.
func (tc *TypeChecker) CheckTerm(term *ast.Term) FluxType {
	if term.Number != nil {
		return IntType{}
	} else if term.String != nil {
		return StringType{}
	} else if term.Bool != nil {
		return BoolType{}
	} else if term.Ident != nil {
		if tc.bindings != nil {
			if t, ok := tc.bindingTypes[tc.bindings.Uses[term]]; ok {
				return t
			}
		}
		if t, ok := tc.env.Lookup(*term.Ident); ok {
			return t
		}
		tc.errorAt(always, CodeUndefinedVariable, term, "undefined variable: %s", *term.Ident)
		return VoidType{}
	}

	tc.report(always, diagnostic.Internal(tc.span(term), "term matched no grammar alternative"))
	return VoidType{}
}

// CheckListExpr checks each element against the first and returns an unknown
// element type for an empty list.
func (tc *TypeChecker) CheckListExpr(list *ast.ListExpr) FluxType {
	if len(list.Elems) == 0 {
		// Empty list - we'll infer the type later or use a generic type
		return ListType{ElementType: UnknownType{}}
	}

	elemType := tc.CheckExpr(list.Elems[0])
	for i, elem := range list.Elems[1:] {
		t := tc.CheckExpr(elem)
		if !TypesEqual(t, elemType) {
			tc.reportWithRelated(always, CodeListElementType, elem,
				tc.span(list.Elems[0]), "the first element is %s", []any{elemType.String()},
				"list element %d has type %s, expected %s", i+2, t.String(), elemType.String())
		}
	}

	return ListType{ElementType: elemType}
}

// CheckDictExpr validates dictionary keys and checks later key and value types
// against the first pair.
func (tc *TypeChecker) CheckDictExpr(dict *ast.DictExpr) FluxType {
	if len(dict.Pairs) == 0 {
		// Empty dictionary
		return DictType{KeyType: UnknownType{}, ValueType: UnknownType{}}
	}

	first := dict.Pairs[0]
	keyType := tc.CheckExpr(first.Key)
	tc.checkDictionaryKey(keyType, first.Key)
	valueType := tc.CheckExpr(first.Value)

	for i, pair := range dict.Pairs[1:] {
		kt := tc.CheckExpr(pair.Key)
		tc.checkDictionaryKey(kt, pair.Key)
		vt := tc.CheckExpr(pair.Value)

		if !TypesEqual(kt, keyType) {
			tc.reportWithRelated(always, CodeDictKeyType, pair.Key,
				tc.span(first.Key), "the first key is %s", []any{keyType.String()},
				"dictionary key %d has type %s, expected %s", i+2, kt.String(), keyType.String())
		}
		if !TypesEqual(vt, valueType) {
			tc.reportWithRelated(always, CodeDictValueType, pair.Value,
				tc.span(first.Value), "the first value is %s", []any{valueType.String()},
				"dictionary value %d has type %s, expected %s", i+2, vt.String(), valueType.String())
		}
	}

	return DictType{KeyType: keyType, ValueType: valueType}
}

// CheckCallExpr checks a call. calleeSpan covers the expression being called,
// which is what an arity or callability diagnostic points at; an argument
// mismatch points at the argument instead.
func (tc *TypeChecker) CheckCallExpr(fnType FluxType, calleeSpan source.Span, call *ast.CallExpr) FluxType {
	if isUnknown(fnType) {
		for _, arg := range call.Args {
			tc.CheckExpr(arg)
		}
		return UnknownType{}
	}
	funcType, ok := fnType.(FunctionType)
	if !ok {
		tc.report(always, diagnostic.Error(CodeNotCallable, calleeSpan,
			"cannot call non-function type: %s", fnType.String()))
		return VoidType{}
	}

	if len(call.Args) != len(funcType.ParamTypes) {
		d := diagnostic.Error(CodeArgumentCount, tc.span(call),
			"function expects %d arguments, got %d", len(funcType.ParamTypes), len(call.Args))
		if calleeSpan.IsValid() {
			d = d.WithRelated(calleeSpan, "this is %s", funcType.String())
		}
		tc.report(always, d)
		return funcType.ReturnType
	}

	for i, arg := range call.Args {
		argType := tc.CheckExpr(arg)
		expectedType := funcType.ParamTypes[i]

		// Allow unknown types to be compatible
		if !isUnknown(expectedType) && !isUnknown(argType) {
			if !TypesEqual(argType, expectedType) {
				tc.reportArgumentMismatch(funcType, i, arg, argType, expectedType)
			}
		}
	}

	return funcType.ReturnType
}

// reportArgumentMismatch points at the argument and attaches the parameter it
// disagrees with, which is the pair a reader needs in order to decide which of
// the two is wrong.
func (tc *TypeChecker) reportArgumentMismatch(funcType FunctionType, i int, arg *ast.Expr, argType, expectedType FluxType) {
	// Arguments are numbered from one, the way a person counts them.
	d := diagnostic.Error(CodeArgumentType, tc.span(arg),
		"argument %d has type %s, expected %s", i+1, argType.String(), expectedType.String())
	if param, ok := funcType.Param(i); ok {
		d = d.WithRelated(param.Span, "parameter %q is declared as %s here",
			param.Name, expectedType.String())
	}
	tc.report(always, d)
}

// CheckIndexExpr checks an index. baseSpan covers the collection being indexed.
func (tc *TypeChecker) CheckIndexExpr(baseType FluxType, baseSpan source.Span, index *ast.IndexExpr) FluxType {
	indexType := tc.CheckExpr(index.Index)

	switch bt := baseType.(type) {
	case UnknownType:
		return UnknownType{}
	case ListType:
		if !TypesEqual(indexType, IntType{}) {
			tc.errorAt(always, CodeIndexType, index.Index,
				"list index must be int, got %s", indexType.String())
		}
		return bt.ElementType
	case DictType:
		tc.checkDictionaryKey(indexType, index.Index)
		if !TypesEqual(indexType, bt.KeyType) {
			tc.errorAt(always, CodeIndexType, index.Index,
				"dictionary key must be %s, got %s", bt.KeyType.String(), indexType.String())
		}
		return bt.ValueType
	default:
		tc.report(always, diagnostic.Error(CodeNotIndexable, baseSpan,
			"cannot index into type: %s", baseType.String()))
		return VoidType{}
	}
}

// CheckFuncExpr checks parameters and the body in a nested scope, validates
// annotations, and retains parameter locations for call diagnostics.
func (tc *TypeChecker) CheckFuncExpr(funcExpr *ast.FuncExpr) FluxType {
	// Create new scope for function parameters
	funcEnv := NewTypeEnv(tc.env)
	oldEnv := tc.env
	tc.env = funcEnv

	// Process parameters with type annotations
	paramTypes := make([]FluxType, len(funcExpr.Params))
	params := make([]ParamInfo, len(funcExpr.Params))
	signature, hasSignature := tc.signatures[funcExpr]
	for i, param := range funcExpr.Params {
		var paramType FluxType

		if param.TypeAnno != nil {
			// Use explicit type annotation
			annotatedType, err := ConvertASTType(param.TypeAnno.Type)
			if err != nil {
				tc.errorAt(always, CodeInvalidAnnotation, param.TypeAnno,
					"invalid type annotation for parameter %s: %v", param.Name, err)
				paramType = UnknownType{} // fallback
			} else {
				paramType = annotatedType
			}
		} else {
			// Use unknown type for inference
			paramType = UnknownType{}
			if hasSignature && i < len(signature.ParamTypes) {
				paramType = signature.ParamTypes[i]
			}
		}

		paramTypes[i] = paramType
		params[i] = ParamInfo{Name: param.Name, Span: tc.span(param)}
		tc.env.Bind(param.Name, paramType)
		if tc.bindings != nil {
			tc.bindingTypes[tc.bindings.Parameters[param]] = paramType
		}
	}

	// Check function body
	bodyType := tc.CheckExpr(funcExpr.Body)

	// Check return type annotation if present
	var returnType FluxType
	if funcExpr.ReturnAnno != nil {
		annotatedReturnType, err := ConvertASTType(funcExpr.ReturnAnno.Type)
		if err != nil {
			tc.errorAt(always, CodeInvalidAnnotation, funcExpr.ReturnAnno,
				"invalid return type annotation: %v", err)
			returnType = bodyType // use inferred type
		} else {
			// The body is what has the wrong type, so it is where this points.
			if !isUnknown(bodyType) && !TypesEqual(bodyType, annotatedReturnType) {
				tc.reportWithRelated(always, CodeReturnMismatch, funcExpr.Body,
					tc.span(funcExpr.ReturnAnno), "the return type is declared %s here",
					[]any{annotatedReturnType.String()},
					"return type mismatch: declared %s but body returns %s",
					annotatedReturnType.String(), bodyType.String())
			}
			returnType = annotatedReturnType
		}
	} else {
		// A variable function annotation also checks the body before the
		// function is exposed, including its recursive self-reference.
		returnType = bodyType
		if hasSignature && !isUnknown(signature.ReturnType) {
			returnType = signature.ReturnType
			if !TypesEqual(bodyType, returnType) {
				related := tc.signatureLocations[funcExpr]
				tc.reportWithRelated(always, CodeReturnMismatch, funcExpr.Body, tc.span(related), "the function signature is declared here", nil, "return type mismatch: declared %s but body returns %s", returnType.String(), bodyType.String())
			}
		}
	}

	// Restore old environment
	tc.env = oldEnv

	return FunctionType{
		ParamTypes: paramTypes,
		ReturnType: returnType,
		Params:     params,
	}
}

// checkDictionaryKey rejects concrete key types other than int, string, and
// bool, leaving unknown types for later checking.
func (tc *TypeChecker) checkDictionaryKey(t FluxType, at ast.Positioned) {
	switch t.(type) {
	case IntType, StringType, BoolType, UnknownType:
	default:
		tc.errorAt(always, CodeInvalidDictKey, at,
			"dictionary key must be int, string, or bool, got %s", t.String())
	}
}

func comparableType(t FluxType) bool {
	switch t := t.(type) {
	case FunctionType:
		return false
	case ListType:
		return comparableType(t.ElementType)
	case DictType:
		return comparableType(t.KeyType) && comparableType(t.ValueType)
	default:
		return true
	}
}

// functionSignature prebinds a complete declared signature for self-recursion.
// Unknowns remain placeholders for nonrecursive functions until Phase 4.
func (tc *TypeChecker) functionSignature(binding *ast.LetStatement, fn *ast.FuncExpr) (FunctionType, bool) {
	sig := FunctionType{ReturnType: UnknownType{}}
	if binding.TypeAnno != nil {
		if t, err := ConvertASTType(binding.TypeAnno.Type); err == nil {
			if f, ok := t.(FunctionType); ok {
				return f, true
			}
		}
	}
	complete := fn.ReturnAnno != nil
	for _, p := range fn.Params {
		var t FluxType = UnknownType{}
		if p.TypeAnno == nil {
			complete = false
		} else if v, err := ConvertASTType(p.TypeAnno.Type); err == nil {
			t = v
		} else {
			complete = false
		}
		sig.ParamTypes = append(sig.ParamTypes, t)
		sig.Params = append(sig.Params, ParamInfo{Name: p.Name, Span: tc.span(p)})
	}
	if fn.ReturnAnno != nil {
		if t, err := ConvertASTType(fn.ReturnAnno.Type); err == nil {
			sig.ReturnType = t
		} else {
			complete = false
		}
	}
	return sig, complete
}
