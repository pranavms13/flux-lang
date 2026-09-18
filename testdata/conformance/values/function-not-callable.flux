//! rule: VAL-FN-NOT-CALLABLE
//! about: calling something that is not a function is reported at the callee
//! status: implemented
//! all: static-error T_NOT_CALLABLE at 8:7..8:8
//! warn-only: runtime-error R_NOT_CALLABLE at 8:8..8:11
//! disabled: runtime-error R_NOT_CALLABLE at 8:8..8:11
let x = 5
print(x(1))
