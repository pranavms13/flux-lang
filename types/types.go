package types

import (
	"fmt"
	"strings"

	"github.com/pranavms13/flux-lang/ast"
	"github.com/pranavms13/flux-lang/diagnostic"
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
	env         *TypeEnv
	diagnostics diagnostic.Bag
	config      TypeCheckingMode
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

func NewTypeCheckerWithConfig(mode TypeCheckingMode) *TypeChecker {
	env := NewTypeEnv(nil)

	// Add built-in functions with more flexible typing
	env.Bind("print", FunctionType{
		ParamTypes: []FluxType{UnknownType{}}, // Accept any type
		ReturnType: VoidType{},
	})

	return &TypeChecker{env: env, config: mode}
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
	if !tc.config.Enabled {
		return
	}
	for _, stmt := range prog.Statements {
		tc.CheckStatement(stmt)
	}
}

func (tc *TypeChecker) CheckStatement(stmt *ast.Statement) {
	if stmt.Let != nil {
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
		} else {
			// Use inferred type
			tc.env.Bind(stmt.Let.Name, exprType)
		}
	} else if stmt.Expr != nil {
		tc.CheckExpr(stmt.Expr)
	}
}

func (tc *TypeChecker) CheckExpr(expr *ast.Expr) FluxType {
	switch {
	case expr.If != nil:
		return tc.CheckIfExpr(expr.If)
	case expr.Bin != nil:
		return tc.CheckBinaryExpr(expr.Bin)
	case expr.Block != nil:
		return tc.CheckBlockExpr(expr.Block)
	case expr.Primary != nil:
		return tc.CheckPrimaryExpr(expr.Primary)
	case expr.Func != nil:
		return tc.CheckFuncExpr(expr.Func)
	default:
		tc.report(always, diagnostic.Internal(tc.span(expr), "expression matched no grammar alternative"))
		return VoidType{}
	}
}

func (tc *TypeChecker) CheckIfExpr(ifExpr *ast.IfExpr) FluxType {
	condType := tc.CheckExpr(ifExpr.Cond)
	if !TypesEqual(condType, BoolType{}) && !isUnknown(condType) {
		d := diagnostic.Error(CodeConditionType, tc.span(ifExpr.Cond),
			"if condition must be bool, got %s", condType.String())
		if !tc.config.Strict {
			d = d.WithNote("treating as truthy")
		}
		tc.report(strictOnly, d)
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

func (tc *TypeChecker) CheckBinaryExpr(expr *ast.Binary) FluxType {
	result := tc.checkAdditive(expr.Left)
	leftSpan := tc.span(expr.Left)
	for _, rest := range expr.Rest {
		right := tc.checkAdditive(rest.Right)
		result = tc.checkOperator(rest.Operator, result, right,
			tc.span(rest), leftSpan, tc.span(rest.Right))
		leftSpan = leftSpan.Union(tc.span(rest))
	}
	return result
}

func (tc *TypeChecker) checkAdditive(expr *ast.Additive) FluxType {
	result := tc.CheckPrimaryExpr(expr.Left)
	leftSpan := tc.span(expr.Left)
	for _, rest := range expr.Rest {
		right := tc.CheckPrimaryExpr(rest.Right)
		result = tc.checkOperator(rest.Operator, result, right,
			tc.span(rest), leftSpan, tc.span(rest.Right))
		leftSpan = leftSpan.Union(tc.span(rest))
	}
	return result
}

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
	case "-", "<", ">":
		if TypesEqual(left, IntType{}) && TypesEqual(right, IntType{}) {
			if operator == "-" {
				return IntType{}
			}
			return BoolType{}
		}
	case "==":
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
	for _, expr := range blockExpr.Exprs {
		lastType = tc.CheckExpr(expr)
	}
	return lastType
}

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

func (tc *TypeChecker) CheckTerm(term *ast.Term) FluxType {
	if term.Number != nil {
		return IntType{}
	} else if term.String != nil {
		return StringType{}
	} else if term.Bool != nil {
		return BoolType{}
	} else if term.Ident != nil {
		if t, ok := tc.env.Lookup(*term.Ident); ok {
			return t
		}
		tc.errorAt(always, CodeUndefinedVariable, term, "undefined variable: %s", *term.Ident)
		return VoidType{}
	}

	tc.report(always, diagnostic.Internal(tc.span(term), "term matched no grammar alternative"))
	return VoidType{}
}

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
				"list element %d has type %s, expected %s", i+1, t.String(), elemType.String())
		}
	}

	return ListType{ElementType: elemType}
}

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
				"dictionary key %d has type %s, expected %s", i+1, kt.String(), keyType.String())
		}
		if !TypesEqual(vt, valueType) {
			tc.reportWithRelated(always, CodeDictValueType, pair.Value,
				tc.span(first.Value), "the first value is %s", []any{valueType.String()},
				"dictionary value %d has type %s, expected %s", i+1, vt.String(), valueType.String())
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

func (tc *TypeChecker) CheckFuncExpr(funcExpr *ast.FuncExpr) FluxType {
	// Create new scope for function parameters
	funcEnv := NewTypeEnv(tc.env)
	oldEnv := tc.env
	tc.env = funcEnv

	// Process parameters with type annotations
	paramTypes := make([]FluxType, len(funcExpr.Params))
	params := make([]ParamInfo, len(funcExpr.Params))
	declaredAt := map[string]*ast.FuncParam{}
	for i, param := range funcExpr.Params {
		if first, repeated := declaredAt[param.Name]; repeated {
			d := diagnostic.Error(CodeDuplicateParameter, tc.span(param),
				"duplicate parameter: %s", param.Name)
			if span := tc.span(first); span.IsValid() {
				d = d.WithRelated(span, "%s is already declared here", param.Name)
			}
			tc.report(always, d)
		} else {
			declaredAt[param.Name] = param
		}
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
		}

		paramTypes[i] = paramType
		params[i] = ParamInfo{Name: param.Name, Span: tc.span(param)}
		tc.env.Bind(param.Name, paramType)
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
		// Use inferred return type
		returnType = bodyType
	}

	// Restore old environment
	tc.env = oldEnv

	return FunctionType{
		ParamTypes: paramTypes,
		ReturnType: returnType,
		Params:     params,
	}
}

func (tc *TypeChecker) checkDictionaryKey(t FluxType, at ast.Positioned) {
	switch t.(type) {
	case IntType, StringType, BoolType, UnknownType:
	default:
		tc.errorAt(always, CodeInvalidDictKey, at,
			"dictionary key must be int, string, or bool, got %s", t.String())
	}
}
