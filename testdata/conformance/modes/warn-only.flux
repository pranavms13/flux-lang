//! rule: MOD-WARN-ONLY
//! about: warn-only downgrades an error and lets the program fail for the reason it warned about
//! status: implemented
//! strict: static-error T_ANNOTATION_MISMATCH at 9:14..9:17
//! lenient: static-error T_ANNOTATION_MISMATCH at 9:14..9:17
//! warn-only: runtime-error R_OPERAND_TYPE at 10:9..10:12
//! warn-only-warning: T_ANNOTATION_MISMATCH at 9:14..9:17
//! disabled: runtime-error R_OPERAND_TYPE at 10:9..10:12
let n: int = "x"
print(n - 1)
