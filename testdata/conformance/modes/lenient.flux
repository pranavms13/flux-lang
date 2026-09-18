//! rule: MOD-LENIENT
//! about: lenient still rejects a real mismatch; it relaxes only three rules
//! status: implemented
//! strict: static-error T_OPERAND_TYPE at 8:7..8:14
//! lenient: static-error T_OPERAND_TYPE at 8:7..8:14
//! warn-only: runtime-error R_OPERAND_TYPE at 8:11..8:14
//! disabled: runtime-error R_OPERAND_TYPE at 8:11..8:14
print("a" - 1)
