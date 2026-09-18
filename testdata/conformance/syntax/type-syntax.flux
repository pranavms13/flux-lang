//! rule: GRM-TYPE
//! about: every type form may be written in an annotation
//! status: implemented
//! all: output
//! stdout: "3\nx\ntrue\n"
let n: int = 1
let s: string = "x"
let b: bool = true
let xs: [int] = [1]
let d: {string: int} = {"k": 2}
let f: fn(int) -> int = fn(y: int): int => y
print(f(n) + d["k"])
print(s)
print(b)
