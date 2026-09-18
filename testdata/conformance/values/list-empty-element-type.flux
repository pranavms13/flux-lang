//! rule: VAL-LIST-HOMOGENEOUS
//! about: an empty list has no element type yet, so it fits any list annotation
//! status: implemented
//! all: output
//! stdout: "both accepted\n"
let a: [int] = []
let b: [string] = []
print("both accepted")
