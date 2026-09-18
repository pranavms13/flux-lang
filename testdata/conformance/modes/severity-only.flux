//! rule: MOD-SEVERITY-ONLY
//! about: the code and span are the same whether the mode reports an error or a warning
//! status: implemented
//! strict: static-error T_COMPARISON_MISMATCH at 11:7..11:15
//! lenient: output
//! lenient-warning: T_COMPARISON_MISMATCH at 11:7..11:15
//! warn-only: output
//! warn-only-warning: T_COMPARISON_MISMATCH at 11:7..11:15
//! disabled: output
//! stdout: "false\n"
print(1 == "1")
