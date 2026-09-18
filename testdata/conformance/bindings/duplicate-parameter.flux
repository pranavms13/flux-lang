//! rule: BND-DUPLICATE-PARAMETER
//! about: a repeated parameter name is reported at the repetition
//! status: implemented
//! all: static-error B_DUPLICATE_PARAMETER at 8:15..8:16
//! warn-only: output
//! disabled: output
//! stdout: ""
let f = fn(x, x) => x
