//! rule: GRM-CALL
//! about: an argument list does not permit a trailing comma
//! status: implemented
//! all: static-error S_UNEXPECTED_TOKEN at 6:5..6:6
let f = fn(x) => x
f(1,)
