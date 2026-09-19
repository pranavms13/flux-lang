//! rule: EVL-IF-TRUTHY
//! about: former truthy conditions are rejected in every execution mode
//! status: implemented
//! all: static-error T_CONDITION_TYPE at 8:10..8:11
//! warn-only: runtime-error R_CONDITION_TYPE at 8:10..8:11
//! disabled: runtime-error R_CONDITION_TYPE at 8:10..8:11
//! warn-only-warning: T_CONDITION_TYPE at 8:10..8:11
print(if 0 then "t" else "f")
print(if "" then "t" else "f")
print(if 5 then "t" else "f")
