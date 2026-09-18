//! rule: TYP-INFERENCE-CONSTRAINTS
//! about: an unannotated parameter is unconstrained, so a bad call reaches run time
//! status: planned
//! milestone: phase-4
//! specified: static-error T_ARGUMENT_TYPE at 8:9..8:12
//! all: runtime-error R_OPERAND_TYPE at 7:20..7:23
let f = fn(x) => x + 1
print(f("a"))
