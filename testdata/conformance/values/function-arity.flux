//! rule: VAL-FN-ARITY
//! about: a call supplies exactly as many arguments as the function declares
//! status: implemented
//! all: static-error T_ARGUMENT_COUNT at 8:10..8:13
//! warn-only: runtime-error R_ARGUMENT_COUNT at 8:10..8:13
//! disabled: runtime-error R_ARGUMENT_COUNT at 8:10..8:13
let add = fn(a: int, b: int): int => a + b
print(add(5))
