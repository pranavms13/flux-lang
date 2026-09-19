//! rule: BND-PARAMETER-SCOPE
//! about: a parameter is not visible outside the function that declares it
//! status: implemented
//! all: static-error B_UNDEFINED_VARIABLE at 6:7..6:8
let f = fn(p) => p
print(p)
