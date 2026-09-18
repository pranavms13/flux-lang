//! rule: VAL-LIST-HOMOGENEOUS
//! about: every element of a list has the type of the first
//! status: implemented
//! all: static-error T_LIST_ELEMENT_TYPE at 8:14..8:19
//! warn-only: output
//! disabled: output
//! stdout: "1\n"
let xs = [1, "two", 3]
print(xs[0])
