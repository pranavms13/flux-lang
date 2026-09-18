//! rule: VAL-VOID
//! about: void cannot be added, and a runtime failure preserves earlier print output
//! status: implemented
//! stdout: "a\n"
//! all: static-error T_OPERAND_TYPE at 8:1..8:15
//! warn-only: runtime-error R_OPERAND_TYPE at 8:12..8:15
//! disabled: runtime-error R_OPERAND_TYPE at 8:12..8:15
print("a") + 1
