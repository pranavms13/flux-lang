//! rule: TYP-INFERRED
//! about: a binding takes its value's type, and an unannotated parameter has none
//! status: implemented
//! all: output
//! stdout: "2\n2\n"
let n = 1
let apply = fn(f, v) => f(v)
print(n + 1)
print(apply(fn(x: int): int => x + 1, 1))
