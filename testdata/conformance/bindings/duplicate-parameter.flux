//! rule: BND-DUPLICATE-PARAMETER
//! about: a repeated parameter name is reported at the repetition
//! status: implemented
//! all: static-error B_DUPLICATE_PARAMETER at 5:15..5:16
let f = fn(x, x) => x
