//! rule: EVL-IF-BOOL
//! about: conditions must be boolean even when type checking is disabled
//! status: implemented
//! all: static-error T_CONDITION_TYPE at 7:10..7:11
//! warn-only: runtime-error R_CONDITION_TYPE at 7:10..7:11
//! disabled: runtime-error R_CONDITION_TYPE at 7:10..7:11
print(if 0 then "t" else "f")
