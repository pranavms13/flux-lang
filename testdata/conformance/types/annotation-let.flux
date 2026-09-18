//! rule: TYP-ANNOTATION-LET
//! about: an annotated binding accepts a value of the annotated type
//! status: implemented
//! all: output
//! stdout: "ab\n0\n"
let n: int = 1
let s: string = "a"
print(s + "b")
print(n - 1)
