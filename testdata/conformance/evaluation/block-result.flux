//! rule: EVL-BLOCK-RESULT
//! about: a block produces its last expression's value and discards the rest
//! status: implemented
//! all: output
//! stdout: "a\nb\n3\n"
let v = { print("a") print("b") 3 }
print(v)
