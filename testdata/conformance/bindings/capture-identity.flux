//! rule: BND-CAPTURE-IDENTITY
//! about: a later inner binding does not change an existing closure
//! status: implemented
//! all: output
//! stdout: "1\n"
let y = 1
let g = fn() => y
let shadowed = { let y = 2
g() }
print(shadowed)
