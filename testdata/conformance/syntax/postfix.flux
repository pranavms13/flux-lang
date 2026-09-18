//! rule: GRM-POSTFIX
//! about: calls and indexes apply left to right to what precedes them
//! status: implemented
//! all: output
//! stdout: "2\n"
let table = {"a": [1, 2]}
let pick = fn(k) => table[k]
print(pick("a")[1])
