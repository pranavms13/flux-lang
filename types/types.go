package types

import (
	"fmt"
	"strings"

	"github.com/pranavms13/flux-lang/ast"
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
		return t.ElementType.Equals(otherList.ElementType)
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
		return t.KeyType.Equals(otherDict.KeyType) && t.ValueType.Equals(otherDict.ValueType)
	}
	return false
}

type FunctionType struct {
	ParamTypes []FluxType
	ReturnType FluxType
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
			if !param.Equals(otherFunc.ParamTypes[i]) {
				return false
			}
		}
		return t.ReturnType.Equals(otherFunc.ReturnType)
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
	env      *TypeEnv
	errors   []string
	warnings []string
	config   TypeCheckingMode
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

	return &TypeChecker{
		env:      env,
		errors:   []string{},
		warnings: []string{},
		config:   mode,
	}
}

func (tc *TypeChecker) Error(msg string) {
	if tc.config.WarnOnly {
		tc.warnings = append(tc.warnings, msg)
	} else {
		tc.errors = append(tc.errors, msg)
	}
}

func (tc *TypeChecker) Warning(msg string) {
	tc.warnings = append(tc.warnings, msg)
}

func (tc *TypeChecker) GetErrors() []string {
	return tc.errors
}

func (tc *TypeChecker) GetWarnings() []string {
	return tc.warnings
}

func (tc *TypeChecker) HasErrors() bool {
	return len(tc.errors) > 0
}

func (tc *TypeChecker) HasWarnings() bool {
	return len(tc.warnings) > 0
}

// Type checking methods
func (tc *TypeChecker) CheckProgram(prog *ast.Program) {
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
				tc.Error(fmt.Sprintf("invalid type annotation: %v", err))
				return
			}

			// Check if the expression type matches the annotation
			if !TypesEqual(exprType, annotatedType) {
				msg := fmt.Sprintf("type mismatch: variable %s declared as %s but assigned %s",
					stmt.Let.Name, annotatedType.String(), exprType.String())

				if tc.config.Strict {
					tc.Error(msg)
				} else {
					// In non-strict mode, allow compatible assignments or issue warnings
					if tc.canAssign(exprType, annotatedType) {
						tc.Warning(fmt.Sprintf("implicit type conversion: %s to %s for variable %s",
							exprType.String(), annotatedType.String(), stmt.Let.Name))
					} else {
						tc.Error(msg)
					}
				}
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
		tc.Error("unknown expression type")
		return VoidType{}
	}
}

func (tc *TypeChecker) CheckIfExpr(ifExpr *ast.IfExpr) FluxType {
	condType := tc.CheckExpr(ifExpr.Cond)
	if !TypesEqual(condType, BoolType{}) && !TypesEqual(condType, UnknownType{}) {
		msg := fmt.Sprintf("if condition must be bool, got %s", condType.String())
		if tc.config.Strict {
			tc.Error(msg)
		} else {
			tc.Warning(msg + " (treating as truthy)")
		}
	}

	thenType := tc.CheckExpr(ifExpr.ThenExpr)
	elseType := tc.CheckExpr(ifExpr.ElseExpr)

	if !TypesEqual(thenType, elseType) && !TypesEqual(thenType, UnknownType{}) && !TypesEqual(elseType, UnknownType{}) {
		msg := fmt.Sprintf("if branches must have same type: then=%s, else=%s",
			thenType.String(), elseType.String())

		if tc.config.Strict {
			tc.Error(msg)
			return VoidType{}
		} else {
			tc.Warning(msg + " (using union type)")
			// In non-strict mode, return the first non-void type or unknown
			if !TypesEqual(thenType, VoidType{}) {
				return thenType
			}
			return elseType
		}
	}

	return thenType
}

func (tc *TypeChecker) CheckBinaryExpr(binExpr *ast.Binary) FluxType {
	leftType := tc.CheckExpr(&ast.Expr{Primary: binExpr.Left})

	if binExpr.Operator == nil || binExpr.Right == nil {
		return leftType
	}

	rightType := tc.CheckExpr(binExpr.Right)

	switch *binExpr.Operator {
	case "+":
		// Allow unknown types for inference
		if TypesEqual(leftType, UnknownType{}) || TypesEqual(rightType, UnknownType{}) {
			// Try to infer based on the known type
			if !TypesEqual(leftType, UnknownType{}) {
				return leftType
			}
			if !TypesEqual(rightType, UnknownType{}) {
				return rightType
			}
			return UnknownType{} // Both unknown, return unknown
		}

		if TypesEqual(leftType, IntType{}) && TypesEqual(rightType, IntType{}) {
			return IntType{}
		}
		if TypesEqual(leftType, StringType{}) && TypesEqual(rightType, StringType{}) {
			return StringType{}
		}

		msg := fmt.Sprintf("invalid operands for +: %s and %s", leftType.String(), rightType.String())
		if tc.config.Strict {
			tc.Error(msg)
		} else {
			// In non-strict mode, be more lenient
			if (TypesEqual(leftType, IntType{}) || TypesEqual(leftType, StringType{})) &&
				(TypesEqual(rightType, IntType{}) || TypesEqual(rightType, StringType{})) {
				tc.Warning(fmt.Sprintf("mixed type addition: %s + %s (converting to string)",
					leftType.String(), rightType.String()))
				return StringType{} // Default to string for mixed additions
			} else {
				tc.Error(msg)
			}
		}
		return VoidType{}
	case "-":
		// Allow unknown types for inference
		if TypesEqual(leftType, UnknownType{}) || TypesEqual(rightType, UnknownType{}) {
			return IntType{} // Assume int for arithmetic
		}

		if TypesEqual(leftType, IntType{}) && TypesEqual(rightType, IntType{}) {
			return IntType{}
		}

		msg := fmt.Sprintf("invalid operands for -: %s and %s", leftType.String(), rightType.String())
		if tc.config.Strict {
			tc.Error(msg)
		} else {
			tc.Warning(msg + " (assuming int)")
			return IntType{}
		}
		return VoidType{}
	case "==":
		// Allow comparison of unknown types
		if TypesEqual(leftType, UnknownType{}) || TypesEqual(rightType, UnknownType{}) {
			return BoolType{}
		}

		if TypesEqual(leftType, rightType) {
			return BoolType{}
		}

		msg := fmt.Sprintf("cannot compare different types: %s and %s", leftType.String(), rightType.String())
		if tc.config.Strict {
			tc.Error(msg)
		} else {
			tc.Warning(msg + " (allowing comparison)")
		}
		return BoolType{}
	case ">", "<":
		// Allow unknown types for comparison
		if TypesEqual(leftType, UnknownType{}) || TypesEqual(rightType, UnknownType{}) {
			return BoolType{}
		}

		if TypesEqual(leftType, IntType{}) && TypesEqual(rightType, IntType{}) {
			return BoolType{}
		}

		msg := fmt.Sprintf("invalid operands for %s: %s and %s", *binExpr.Operator, leftType.String(), rightType.String())
		if tc.config.Strict {
			tc.Error(msg)
		} else {
			tc.Warning(msg + " (assuming numeric comparison)")
		}
		return BoolType{}
	default:
		tc.Error(fmt.Sprintf("unknown binary operator: %s", *binExpr.Operator))
		return VoidType{}
	}
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

	// Apply postfixes
	currentType := baseType
	for _, postfix := range primary.Postfix {
		if postfix.Call != nil {
			currentType = tc.CheckCallExpr(currentType, postfix.Call)
		} else if postfix.Index != nil {
			currentType = tc.CheckIndexExpr(currentType, postfix.Index)
		}
	}

	return currentType
}

func (tc *TypeChecker) CheckBaseExpr(base *ast.BaseExpr) FluxType {
	if base.Term != nil {
		return tc.CheckTerm(base.Term)
	} else if base.List != nil {
		return tc.CheckListExpr(base.List)
	} else if base.Dict != nil {
		return tc.CheckDictExpr(base.Dict)
	}

	tc.Error("unknown base expression")
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
		tc.Error(fmt.Sprintf("undefined variable: %s", *term.Ident))
		return VoidType{}
	}

	tc.Error("unknown term")
	return VoidType{}
}

func (tc *TypeChecker) CheckListExpr(list *ast.ListExpr) FluxType {
	if len(list.Elems) == 0 {
		// Empty list - we'll infer the type later or use a generic type
		return ListType{ElementType: VoidType{}}
	}

	elemType := tc.CheckExpr(list.Elems[0])
	for i, elem := range list.Elems[1:] {
		t := tc.CheckExpr(elem)
		if !TypesEqual(t, elemType) {
			tc.Error(fmt.Sprintf("list element %d has type %s, expected %s",
				i+1, t.String(), elemType.String()))
		}
	}

	return ListType{ElementType: elemType}
}

func (tc *TypeChecker) CheckDictExpr(dict *ast.DictExpr) FluxType {
	if len(dict.Pairs) == 0 {
		// Empty dictionary
		return DictType{KeyType: VoidType{}, ValueType: VoidType{}}
	}

	keyType := tc.CheckExpr(dict.Pairs[0].Key)
	valueType := tc.CheckExpr(dict.Pairs[0].Value)

	for i, pair := range dict.Pairs[1:] {
		kt := tc.CheckExpr(pair.Key)
		vt := tc.CheckExpr(pair.Value)

		if !TypesEqual(kt, keyType) {
			tc.Error(fmt.Sprintf("dictionary key %d has type %s, expected %s",
				i+1, kt.String(), keyType.String()))
		}
		if !TypesEqual(vt, valueType) {
			tc.Error(fmt.Sprintf("dictionary value %d has type %s, expected %s",
				i+1, vt.String(), valueType.String()))
		}
	}

	return DictType{KeyType: keyType, ValueType: valueType}
}

func (tc *TypeChecker) CheckCallExpr(fnType FluxType, call *ast.CallExpr) FluxType {
	funcType, ok := fnType.(FunctionType)
	if !ok {
		tc.Error(fmt.Sprintf("cannot call non-function type: %s", fnType.String()))
		return VoidType{}
	}

	if len(call.Args) != len(funcType.ParamTypes) {
		tc.Error(fmt.Sprintf("function expects %d arguments, got %d",
			len(funcType.ParamTypes), len(call.Args)))
		return funcType.ReturnType
	}

	for i, arg := range call.Args {
		argType := tc.CheckExpr(arg)
		expectedType := funcType.ParamTypes[i]

		// Allow unknown types to be compatible
		if !TypesEqual(expectedType, UnknownType{}) && !TypesEqual(argType, UnknownType{}) {
			if !TypesEqual(argType, expectedType) {
				tc.Error(fmt.Sprintf("argument %d has type %s, expected %s",
					i, argType.String(), expectedType.String()))
			}
		}
	}

	return funcType.ReturnType
}

func (tc *TypeChecker) CheckIndexExpr(baseType FluxType, index *ast.IndexExpr) FluxType {
	indexType := tc.CheckExpr(index.Index)

	switch bt := baseType.(type) {
	case ListType:
		if !TypesEqual(indexType, IntType{}) {
			tc.Error(fmt.Sprintf("list index must be int, got %s", indexType.String()))
		}
		return bt.ElementType
	case DictType:
		if !TypesEqual(indexType, bt.KeyType) {
			tc.Error(fmt.Sprintf("dictionary key must be %s, got %s",
				bt.KeyType.String(), indexType.String()))
		}
		return bt.ValueType
	default:
		tc.Error(fmt.Sprintf("cannot index into type: %s", baseType.String()))
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
	for i, param := range funcExpr.Params {
		var paramType FluxType

		if param.TypeAnno != nil {
			// Use explicit type annotation
			annotatedType, err := ConvertASTType(param.TypeAnno.Type)
			if err != nil {
				tc.Error(fmt.Sprintf("invalid type annotation for parameter %s: %v", param.Name, err))
				paramType = UnknownType{} // fallback
			} else {
				paramType = annotatedType
			}
		} else {
			// Use unknown type for inference
			paramType = UnknownType{}
		}

		paramTypes[i] = paramType
		tc.env.Bind(param.Name, paramType)
	}

	// Check function body
	bodyType := tc.CheckExpr(funcExpr.Body)

	// Check return type annotation if present
	var returnType FluxType
	if funcExpr.ReturnAnno != nil {
		annotatedReturnType, err := ConvertASTType(funcExpr.ReturnAnno.Type)
		if err != nil {
			tc.Error(fmt.Sprintf("invalid return type annotation: %v", err))
			returnType = bodyType // use inferred type
		} else {
			// Check if body type matches return annotation
			if !TypesEqual(bodyType, UnknownType{}) && !TypesEqual(bodyType, annotatedReturnType) {
				tc.Error(fmt.Sprintf("return type mismatch: declared %s but body returns %s",
					annotatedReturnType.String(), bodyType.String()))
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
	}
}

// canAssign checks if a value of one type can be assigned to another in non-strict mode
func (tc *TypeChecker) canAssign(from, to FluxType) bool {
	if TypesEqual(from, to) {
		return true
	}

	// In non-strict mode, allow some implicit conversions
	switch to.(type) {
	case UnknownType:
		return true
	case IntType:
		// Allow string to int conversion in non-strict mode (would need runtime parsing)
		return false // For now, don't allow this
	case StringType:
		// Allow most types to string conversion
		return true
	default:
		return false
	}
}
