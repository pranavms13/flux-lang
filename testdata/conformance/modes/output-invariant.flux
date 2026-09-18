//! rule: MOD-OUTPUT-INVARIANT
//! about: every mode that runs this program prints exactly the same thing
//! status: implemented
//! all: output
//! stdout: "1\ntwo\n"
let xs = [1]
print(xs[0])
print("two")
