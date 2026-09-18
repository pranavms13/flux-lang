//! rule: BND-TOP-LEVEL-SCOPE
//! about: a top-level binding is visible inside functions declared after it
//! status: implemented
//! all: output
//! stdout: "15\n"
let base = 10
let add = fn(n) => n + base
print(add(5))
