package vm

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

type Opcode byte

const (
	OpConstant Opcode = iota
	OpAdd
	OpSub
	OpEqual
	OpGreater
	OpLess
	OpPop
	OpPrint
	OpReturn
	OpDefineGlobal
	OpGetGlobal
	OpCall
	OpClosure
	OpJumpIfFalse
	OpJump
	OpJumpIfTrue
	OpIndex
	OpArray
	OpDict
)

type Chunk struct {
	Code      []byte
	Constants []interface{}
	Params    []string
}

type Closure struct {
	Chunk  *Chunk
	Locals map[string]interface{}
}

type builtinPrint struct{}

type VM struct {
	chunk   *Chunk
	ip      int
	stack   []interface{}
	globals map[string]interface{}
	locals  map[string]interface{}
}

func New(chunk *Chunk) *VM {
	return &VM{
		chunk:   chunk,
		ip:      0,
		stack:   []interface{}{},
		globals: map[string]interface{}{},
		locals:  map[string]interface{}{},
	}
}

func (vm *VM) Run() {
	for vm.ip < len(vm.chunk.Code) {
		op := Opcode(vm.readByte())
		switch op {
		case OpConstant:
			index := vm.readOperand()
			vm.push(vm.chunk.Constants[index])
		case OpIndex:
			index := vm.pop()
			value := vm.pop()
			switch v := value.(type) {
			case []interface{}:
				idx, ok := index.(int)
				if !ok {
					panic("Array index must be an integer")
				}
				if idx < 0 || idx >= len(v) {
					panic("Array index out of bounds")
				}
				vm.push(v[idx])
			case map[interface{}]interface{}:
				val, exists := v[index]
				if !exists {
					panic(fmt.Sprintf("Key not found in dictionary: %v", index))
				}
				vm.push(val)
			default:
				panic(fmt.Sprintf("Cannot index into value of type %T", value))
			}
		case OpDict:
			size := vm.readOperand()
			keys := make([]interface{}, size)
			values := make([]interface{}, size)
			for i := size - 1; i >= 0; i-- {
				values[i] = vm.pop()
				keys[i] = vm.pop()
			}
			dict := make(map[interface{}]interface{}, size)
			for i, key := range keys {
				dict[key] = values[i]
			}
			vm.push(dict)
		case OpArray:
			size := vm.readOperand()
			elems := make([]interface{}, size)
			for i := int(size) - 1; i >= 0; i-- {
				elems[i] = vm.pop()
			}
			vm.push(elems)
		case OpAdd:
			b := vm.pop()
			a := vm.pop()
			switch aVal := a.(type) {
			case int:
				if bVal, ok := b.(int); ok {
					vm.push(aVal + bVal)
				} else {
					panic("Cannot add non-integer to integer")
				}
			case string:
				if bVal, ok := b.(string); ok {
					vm.push(aVal + bVal)
				} else {
					panic("Cannot add non-string to string")
				}
			default:
				panic("Cannot add non-numeric, non-string values")
			}
		case OpSub:
			b := vm.pop().(int)
			a := vm.pop().(int)
			vm.push(a - b)
		case OpEqual:
			b := vm.pop()
			a := vm.pop()
			vm.push(reflect.DeepEqual(a, b))
		case OpGreater:
			b := vm.pop().(int)
			a := vm.pop().(int)
			vm.push(a > b)
		case OpLess:
			b := vm.pop().(int)
			a := vm.pop().(int)
			vm.push(a < b)
		case OpPop:
			vm.pop()
		case OpPrint:
			val := vm.pop()
			if val != nil {
				fmt.Println(val)
			}
		case OpDefineGlobal:
			nameIdx := vm.readOperand()
			name := vm.chunk.Constants[nameIdx].(string)
			val := vm.pop()
			vm.globals[name] = val
		case OpGetGlobal:
			nameIdx := vm.readOperand()
			name := vm.chunk.Constants[nameIdx].(string)
			if val, ok := vm.locals[name]; ok {
				vm.push(val)
			} else if val, ok := vm.globals[name]; ok {
				vm.push(val)
			} else if name == "print" {
				// Special handling for print function
				vm.push(builtinPrint{})
			} else {
				panic(fmt.Sprintf("Undefined variable: %s", name))
			}
		case OpJumpIfFalse:
			offset := vm.readOperand()
			if !vm.truthy(vm.pop()) {
				vm.ip = int(offset)
			}
		case OpJumpIfTrue:
			offset := vm.readOperand()
			if vm.truthy(vm.pop()) {
				vm.ip = int(offset)
			}
		case OpJump:
			offset := vm.readOperand()
			vm.ip = int(offset)
		case OpCall:
			nargs := vm.readOperand()
			args := make([]interface{}, int(nargs))
			for i := int(nargs) - 1; i >= 0; i-- {
				args[i] = vm.pop()
			}
			callee := vm.pop()
			if callee == nil {
				panic("Cannot call nil")
			}
			switch fn := callee.(type) {
			case *Closure:
				if len(args) != len(fn.Chunk.Params) {
					panic("argument count mismatch")
				}
				subVM := New(fn.Chunk)
				subVM.globals = vm.globals
				for name, value := range fn.Locals {
					subVM.locals[name] = value
				}
				for i, param := range fn.Chunk.Params {
					subVM.locals[param] = args[i]
				}
				subVM.Run()
				vm.push(subVM.pop())
			case builtinPrint:
				if len(args) != 1 {
					panic("print expects 1 argument")
				}
				fmt.Println(args[0])
				vm.push(nil)
			default:
				panic(fmt.Sprintf("Cannot call non-function: %v", fn))
			}

		case OpClosure:
			fnIdx := vm.readOperand()
			fnChunk := vm.chunk.Constants[fnIdx].(*Chunk)
			locals := make(map[string]interface{}, len(vm.locals))
			for name, value := range vm.locals {
				locals[name] = value
			}
			vm.push(&Closure{Chunk: fnChunk, Locals: locals})
		case OpReturn:
			return
		default:
			panic(fmt.Sprintf("Unknown opcode: %d", op))
		}
	}
}

func (vm *VM) push(val interface{}) {
	vm.stack = append(vm.stack, val)
}

func (vm *VM) pop() interface{} {
	if len(vm.stack) == 0 {
		return nil
	}
	val := vm.stack[len(vm.stack)-1]
	vm.stack = vm.stack[:len(vm.stack)-1]
	return val
}

func (vm *VM) readByte() byte {
	b := vm.chunk.Code[vm.ip]
	vm.ip++
	return b
}

func (vm *VM) truthy(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case string:
		return val != ""
	default:
		return val != nil
	}
}

func (vm *VM) readOperand() int {
	operand := binary.BigEndian.Uint32(vm.chunk.Code[vm.ip : vm.ip+4])
	vm.ip += 4
	return int(operand)
}
