//! rule: BND-REDECLARE
//! about: same-scope declarations cannot replace an immutable binding
//! status: implemented
//! all: static-error B_DUPLICATE_DECLARATION at 6:1..6:10
let x = 1
let x = 2
print(x)
