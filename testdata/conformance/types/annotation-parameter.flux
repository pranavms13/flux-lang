//! rule: TYP-ANNOTATION-PARAM
//! about: an annotated parameter constrains the argument at every call
//! status: implemented
//! all: static-error T_ARGUMENT_TYPE at 8:11..8:14
//! warn-only: runtime-error R_OPERAND_TYPE at 7:32..7:35
//! disabled: runtime-error R_OPERAND_TYPE at 7:32..7:35
let inc = fn(x: int): int => x + 1
print(inc("a"))
