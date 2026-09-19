//! rule: BND-SELF-RECURSION
//! about: a variable annotation supplies the complete recursive signature
//! status: implemented
//! all: output
//! stdout: "10\n"
let sum: fn(int)->int=fn(n)=>if n==0 then 0 else n+sum(n-1)
print(sum(4))
