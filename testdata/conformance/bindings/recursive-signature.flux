//! rule: BND-SELF-RECURSION
//! about: checking requires a complete recursive signature
//! status: implemented
//! all: static-error T_RECURSIVE_SIGNATURE at 8:1..8:40
//! warn-only: output
//! disabled: output
//! stdout: "0\n"
let f=fn(n)=>if n==0 then 0 else f(n-1)
print(f(3))
