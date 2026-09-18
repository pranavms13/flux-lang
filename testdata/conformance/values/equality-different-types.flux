//! rule: VAL-EQUALITY
//! about: values of different types are never equal, and strict refuses to ask
//! status: implemented
//! strict: static-error T_COMPARISON_MISMATCH at 9:7..9:15
//! lenient: output
//! warn-only: output
//! disabled: output
//! stdout: "false\n"
print(1 == "1")
