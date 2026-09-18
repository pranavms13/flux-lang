//! rule: MOD-STRICT
//! about: differing branch types are an error only in strict mode
//! status: implemented
//! strict: static-error T_BRANCH_MISMATCH at 10:28..10:32
//! lenient: output
//! lenient-warning: T_BRANCH_MISMATCH at 10:28..10:32
//! warn-only: output
//! disabled: output
//! stdout: "42\n"
print(if true then 42 else "no")
