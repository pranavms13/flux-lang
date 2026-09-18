//! rule: BND-BUILTIN-PRINT
//! about: print is an ordinary binding, so it can be aliased and shadowed
//! status: implemented
//! all: output
//! stdout: "hello\ninner!\n"
let alias = print
alias("hello")
let shadow = fn(print) => print + "!"
print(shadow("inner"))
