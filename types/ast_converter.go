package types

import (
	"fmt"

	"github.com/pranavms13/flux-lang/ast"
)

// Convert AST type annotations to internal FluxType
func ConvertASTType(astType *ast.Type) (FluxType, error) {
	if astType == nil {
		return nil, fmt.Errorf("nil AST type")
	}

	switch {
	case astType.Basic != nil:
		switch *astType.Basic {
		case "int":
			return IntType{}, nil
		case "string":
			return StringType{}, nil
		case "bool":
			return BoolType{}, nil
		case "void":
			return VoidType{}, nil
		default:
			return nil, fmt.Errorf("unknown basic type: %s", *astType.Basic)
		}

	case astType.List != nil:
		elemType, err := ConvertASTType(astType.List.ElemType)
		if err != nil {
			return nil, fmt.Errorf("error converting list element type: %w", err)
		}
		return ListType{ElementType: elemType}, nil

	case astType.Dict != nil:
		keyType, err := ConvertASTType(astType.Dict.KeyType)
		if err != nil {
			return nil, fmt.Errorf("error converting dict key type: %w", err)
		}
		valueType, err := ConvertASTType(astType.Dict.ValueType)
		if err != nil {
			return nil, fmt.Errorf("error converting dict value type: %w", err)
		}
		return DictType{KeyType: keyType, ValueType: valueType}, nil

	case astType.Function != nil:
		paramTypes := make([]FluxType, len(astType.Function.ParamTypes))
		for i, paramType := range astType.Function.ParamTypes {
			pt, err := ConvertASTType(paramType)
			if err != nil {
				return nil, fmt.Errorf("error converting function parameter %d type: %w", i, err)
			}
			paramTypes[i] = pt
		}

		returnType, err := ConvertASTType(astType.Function.ReturnType)
		if err != nil {
			return nil, fmt.Errorf("error converting function return type: %w", err)
		}

		return FunctionType{ParamTypes: paramTypes, ReturnType: returnType}, nil

	default:
		return nil, fmt.Errorf("unknown AST type")
	}
}

// Convert internal FluxType to AST type (for error messages, etc.)
func ConvertFluxTypeToAST(fluxType FluxType) *ast.Type {
	switch t := fluxType.(type) {
	case IntType:
		basic := "int"
		return &ast.Type{Basic: &basic}
	case StringType:
		basic := "string"
		return &ast.Type{Basic: &basic}
	case BoolType:
		basic := "bool"
		return &ast.Type{Basic: &basic}
	case VoidType:
		basic := "void"
		return &ast.Type{Basic: &basic}
	case ListType:
		return &ast.Type{
			List: &ast.ListType{
				ElemType: ConvertFluxTypeToAST(t.ElementType),
			},
		}
	case DictType:
		return &ast.Type{
			Dict: &ast.DictType{
				KeyType:   ConvertFluxTypeToAST(t.KeyType),
				ValueType: ConvertFluxTypeToAST(t.ValueType),
			},
		}
	case FunctionType:
		paramTypes := make([]*ast.Type, len(t.ParamTypes))
		for i, pt := range t.ParamTypes {
			paramTypes[i] = ConvertFluxTypeToAST(pt)
		}
		return &ast.Type{
			Function: &ast.FuncType{
				ParamTypes: paramTypes,
				ReturnType: ConvertFluxTypeToAST(t.ReturnType),
			},
		}
	default:
		// Default to void for unknown types
		basic := "void"
		return &ast.Type{Basic: &basic}
	}
}
