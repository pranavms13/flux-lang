//! rule: BND-PARAMETER-SCOPE
//! about: a parameter is not visible outside the function that declares it
//! status: implemented
//! all: static-error B_UNDEFINED_VARIABLE at 8:7..8:8
//! warn-only: runtime-error R_UNDEFINED_VALUE at 8:7..8:8
//! disabled: runtime-error R_UNDEFINED_VALUE at 8:7..8:8
let f = fn(p) => p
print(p)
