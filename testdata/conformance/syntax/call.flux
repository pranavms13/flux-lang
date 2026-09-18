//! rule: GRM-CALL
//! about: a call passes its arguments in parentheses
//! status: implemented
//! all: output
//! stdout: "7\n"
let add = fn(a: int, b: int): int => a + b
print(add(3, 4))
