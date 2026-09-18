//! rule: EVL-IF-TRUTHY
//! about: outside strict mode a non-bool condition warns and is read as truthy
//! status: implemented
//! strict: static-error T_CONDITION_TYPE at 11:10..11:11
//! lenient: output
//! lenient-warning: T_CONDITION_TYPE at 11:10..11:11
//! warn-only: output
//! warn-only-warning: T_CONDITION_TYPE at 11:10..11:11
//! disabled: output
//! stdout: "f\nf\nt\n"
print(if 0 then "t" else "f")
print(if "" then "t" else "f")
print(if 5 then "t" else "f")
