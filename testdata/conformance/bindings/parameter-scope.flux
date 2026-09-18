//! rule: BND-PARAMETER-SCOPE
//! about: a parameter shadows a top-level binding only inside its own body
//! status: implemented
//! all: output
//! stdout: "param\nglobal\n"
let x = "global"
let f = fn(x) => x
print(f("param"))
print(x)
