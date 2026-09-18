//! rule: LEX-IDENT
//! about: identifiers may start with _ , carry digits, and are case-sensitive
//! status: implemented
//! all: output
//! stdout: "1\n2\n"
let _value1 = 1
let Value1 = 2
print(_value1)
print(Value1)
