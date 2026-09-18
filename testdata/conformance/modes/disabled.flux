//! rule: MOD-DISABLED
//! about: with checking off, a program the checker rejects still runs
//! status: implemented
//! strict: static-error T_LIST_ELEMENT_TYPE at 9:17..9:22
//! lenient: static-error T_LIST_ELEMENT_TYPE at 9:17..9:22
//! warn-only: output
//! disabled: output
//! stdout: "two\n"
let mixed = [1, "two"]
print(mixed[1])
