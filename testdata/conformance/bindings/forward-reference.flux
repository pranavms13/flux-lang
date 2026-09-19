//! rule: BND-FORWARD-REFERENCE
//! about: forward references are rejected before optional checking
//! status: implemented
//! all: static-error B_UNDEFINED_VARIABLE at 5:17..5:18
let g = fn() => y
let y = 10
print(g())
