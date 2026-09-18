//! rule: EVL-ORDER-LIST
//! about: list elements are evaluated left to right
//! status: implemented
//! all: output
//! stdout: "1\n2\n3\n1\n"
let trace = fn(v) => { print(v) v }
let xs = [trace(1), trace(2), trace(3)]
print(xs[0])
