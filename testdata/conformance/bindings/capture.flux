//! rule: BND-CAPTURE
//! about: a returned function keeps the parameters of the function that made it
//! status: implemented
//! all: output
//! stdout: "8\n"
let adder = fn(a) => fn(b) => a + b
let add5 = adder(5)
print(add5(3))
