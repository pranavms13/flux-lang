//! rule: VAL-VOID
//! about: void has no value, so no operator accepts it
//! status: implemented
//! all: static-error T_OPERAND_TYPE at 7:1..7:15
//! warn-only: runtime-error R_OPERAND_TYPE at 7:12..7:15
//! disabled: runtime-error R_OPERAND_TYPE at 7:12..7:15
print("a") + 1
