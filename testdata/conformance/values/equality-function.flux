//! rule: VAL-EQUALITY-FUNCTION
//! about: comparing functions currently compares host structure instead of failing
//! status: planned
//! milestone: phase-3
//! specified: static-error T_INCOMPARABLE at 10:7..10:13
//! all: output
//! stdout: "false\ntrue\n"
let f = fn(x) => x
let g = fn(x) => x
print(f == g)
print(f == f)
