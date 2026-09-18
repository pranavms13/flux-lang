//! rule: EVL-ORDER-CALL
//! about: the callee is evaluated first, then the arguments left to right
//! status: implemented
//! all: output
//! stdout: "callee\nfirst\nsecond\n0\n"
let trace = fn(label) => { print(label) label }
let table = {"callee": fn(a, b) => 0}
print(table[trace("callee")](trace("first"), trace("second")))
