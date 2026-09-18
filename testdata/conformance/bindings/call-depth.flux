//! rule: VAL-FN-DEPTH
//! about: excessive recursion returns a located failure
//! status: implemented
//! all: runtime-error R_CALL_DEPTH at 5:25..5:30
let f=fn(n: int): int=>f(n+1)
f(0)
