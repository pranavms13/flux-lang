package vm

import (
	"fmt"
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
)

type Chunk struct {
	Code      []byte
	Constants []interface{}
	Params    []string
}

type Closure struct {
	Chunk *Chunk
	Args  []interface{}
}

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
			index := vm.readByte()
			vm.push(vm.chunk.Constants[index])
		case OpIndex:
			index := vm.pop().(int)
			val := vm.pop()
			arr, ok := val.([]interface{})
			if !ok {
				panic(fmt.Sprintf("Cannot index into non-array value: %v", val))
			}
			if index < 0 || index >= len(arr) {
				panic("Array index out of bounds")
			}
			vm.push(arr[index])
		case OpArray:
			size := vm.readByte()
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
			vm.push(a == b)
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
			fmt.Println(val)
		case OpDefineGlobal:
			nameIdx := vm.readByte()
			name := vm.chunk.Constants[nameIdx].(string)
			val := vm.pop()
			vm.globals[name] = val
		case OpGetGlobal:
			nameIdx := vm.readByte()
			name := vm.chunk.Constants[nameIdx].(string)
			if val, ok := vm.locals[name]; ok {
				vm.push(val)
			} else if val, ok := vm.globals[name]; ok {
				vm.push(val)
			} else if name == "print" {
				// Special handling for print function
				vm.push("print")
			} else {
				panic(fmt.Sprintf("Undefined variable: %s", name))
			}
		case OpJumpIfFalse:
			offset := vm.readByte()
			if !vm.truthy(vm.peek()) {
				vm.ip = int(offset)
			}
		case OpJumpIfTrue:
			offset := vm.readByte()
			if vm.truthy(vm.peek()) {
				vm.ip = int(offset)
			}
		case OpJump:
			offset := vm.readByte()
			vm.ip = int(offset)
		case OpCall:
			nargs := vm.readByte()
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
				subVM := New(fn.Chunk)
				subVM.globals = vm.globals
				for i, param := range fn.Chunk.Params {
					subVM.locals[param] = args[i]
				}
				subVM.Run()
				if len(subVM.stack) > 0 {
					vm.push(subVM.stack[len(subVM.stack)-1])
				} else {
					vm.push(nil)
				}
			case string:
				if fn == "print" {
					// For print, just push the last argument without printing
					if len(args) > 0 {
						vm.push(args[len(args)-1])
					} else {
						vm.push(nil)
					}
				} else {
					// Look up function in globals
					if val, ok := vm.globals[fn]; ok {
						if closure, ok := val.(*Closure); ok {
							subVM := New(closure.Chunk)
							subVM.globals = vm.globals
							for i, param := range closure.Chunk.Params {
								subVM.locals[param] = args[i]
							}
							subVM.Run()
							if len(subVM.stack) > 0 {
								vm.push(subVM.stack[len(subVM.stack)-1])
							} else {
								vm.push(nil)
							}
						} else {
							panic(fmt.Sprintf("Cannot call non-function: %v", val))
						}
					} else {
						panic(fmt.Sprintf("Undefined function: %s", fn))
					}
				}
			default:
				panic(fmt.Sprintf("Cannot call non-function: %v", fn))
			}
		case OpClosure:
			fnIdx := vm.readByte()
			fnChunk := vm.chunk.Constants[fnIdx].(*Chunk)
			vm.push(&Closure{Chunk: fnChunk})
		case OpReturn:
			if len(vm.stack) > 0 {
				retVal := vm.stack[len(vm.stack)-1]
				vm.stack = vm.stack[:len(vm.stack)-1]
				vm.push(retVal)
			}
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

func (vm *VM) peek() interface{} {
	if len(vm.stack) == 0 {
		return nil
	}
	return vm.stack[len(vm.stack)-1]
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
