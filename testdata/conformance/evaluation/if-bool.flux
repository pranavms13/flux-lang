//! rule: EVL-IF-BOOL
//! about: truthiness survives outside strict mode, where the rule requires a bool everywhere
//! status: planned
//! milestone: phase-3
//! specified: static-error T_CONDITION_TYPE at 13:10..13:11
//! specified-warn-only: runtime-error R_CONDITION_TYPE at 13:10..13:11
//! specified-disabled: runtime-error R_CONDITION_TYPE at 13:10..13:11
//! strict: static-error T_CONDITION_TYPE at 13:10..13:11
//! lenient: output
//! warn-only: output
//! disabled: output
//! stdout: "f\n"
print(if 0 then "t" else "f")
