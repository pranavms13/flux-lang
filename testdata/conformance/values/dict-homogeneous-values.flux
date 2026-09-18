//! rule: VAL-DICT-HOMOGENEOUS
//! about: every value of a dictionary has the type of the first
//! status: implemented
//! all: static-error T_DICT_VALUE_TYPE at 8:34..8:36
//! warn-only: output
//! disabled: output
//! stdout: "Alice\n"
let d = {"name": "Alice", "age": 25}
print(d["name"])
