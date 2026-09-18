//! rule: GRM-ADJACENCY
//! about: a newline does not end a statement, so a following ( is a call
//! status: implemented
//! all: output
//! stdout: "1\n"
let f = fn(x) => x
print(f
(1))
