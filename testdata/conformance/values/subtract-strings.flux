//! rule: VAL-SUB
//! about: - has no string overload, unlike +
//! status: implemented
//! all: static-error T_OPERAND_TYPE at 7:7..7:16
//! warn-only: runtime-error R_OPERAND_TYPE at 7:11..7:16
//! disabled: runtime-error R_OPERAND_TYPE at 7:11..7:16
print("a" - "b")
