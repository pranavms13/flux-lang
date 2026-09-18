//! rule: BND-LET
//! about: let evaluates its value once and binds it to a name
//! status: implemented
//! all: output
//! stdout: "2\n"
let x = 1
let y = x + 1
print(y)
