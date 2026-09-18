//! rule: GRM-BLOCK
//! about: a block runs its expressions in order and keeps the last value
//! status: implemented
//! all: output
//! stdout: "first\n2\n"
let value = { print("first") 2 }
print(value)
