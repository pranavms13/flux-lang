// Checked arithmetic, boolean control flow, local bindings and recursion.
let factorial = fn(n: int): int => {
  let base: bool = n <= 1
  if base then { 1 } else { n * factorial(n - 1) }
}
print(factorial(5))
print(false && (1 / 0 > 0))

// Each call creates an independent captured cell for a local recursive closure.
let makeSum = fn(offset: int): fn(int) -> int => {
  let sum: fn(int) -> int = fn(n) =>
    if n <= 0 then offset else n + sum(n - 1)
  sum
}
let sum = makeSum(10)
print(sum(3))
print([1 + 2 * 3, -7 % 2, -9223372036854775808])

// A later inner binding cannot change an earlier closure's lexical lookup.
let x = 1
let read = {
  let earlier = fn(): int => x
  let x = 2
  earlier
}
print(read())
