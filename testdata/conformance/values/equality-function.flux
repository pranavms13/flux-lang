//! rule: VAL-EQUALITY-FUNCTION
//! about: functions are rejected by static and runtime equality checks
//! status: implemented
//! all: static-error T_INCOMPARABLE at 9:7..9:13
//! warn-only: runtime-error R_INCOMPARABLE at 9:7..9:13
//! disabled: runtime-error R_INCOMPARABLE at 9:7..9:13
let f = fn(x) => x
let g = fn(x) => x
print(f == g)
print(f == f)
