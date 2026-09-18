//! rule: GRM-FN
//! about: a function literal may carry parameter and return annotations
//! status: implemented
//! all: output
//! stdout: "a\n2\n"
let id = fn(x) => x
let inc = fn(x: int): int => x + 1
print(id("a"))
print(inc(1))
