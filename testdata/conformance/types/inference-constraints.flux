//! rule: TYP-INFERENCE-CONSTRAINTS
//! about: an unannotated parameter is unconstrained, so a bad call reaches run time
//! status: planned
//! milestone: phase-4
//! specified: static-error T_ARGUMENT_TYPE at 10:9..10:12
//! specified-warn-only: runtime-error R_OPERAND_TYPE at 9:20..9:23
//! specified-disabled: runtime-error R_OPERAND_TYPE at 9:20..9:23
//! all: runtime-error R_OPERAND_TYPE at 9:20..9:23
let f = fn(x) => x + 1
print(f("a"))
