//! rule: BND-SELF-RECURSION
//! about: a function with a complete signature can call itself
//! status: implemented
//! all: output
//! stdout: "6\n"
let sum = fn(n: int): int => if n > 0 then n + sum(n - 1) else 0
print(sum(3))
