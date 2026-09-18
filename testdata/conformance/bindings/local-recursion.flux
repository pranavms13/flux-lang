//! rule: BND-SELF-RECURSION
//! about: a local recursive closure survives its outer call
//! status: implemented
//! all: output
//! stdout: "16\n"
let make=fn(offset: int)=>{let sum=fn(n: int): int=>if n==0 then offset else n+sum(n-1);sum}
let a=make(10)
print(a(3))
