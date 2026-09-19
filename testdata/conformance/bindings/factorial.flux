//! rule: BND-SELF-RECURSION
//! about: typed factorial uses a local binding and recursive multiplication
//! status: implemented
//! all: output
//! stdout: "120\n"
let factorial = fn(n: int): int => {
  let base: bool = n <= 1
  if base then { 1 } else { n * factorial(n - 1) }
}
print(factorial(5))
