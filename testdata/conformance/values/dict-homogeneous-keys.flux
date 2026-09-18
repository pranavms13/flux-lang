//! rule: VAL-DICT-HOMOGENEOUS
//! about: every key of a dictionary has the type of the first
//! status: implemented
//! all: static-error T_DICT_KEY_TYPE at 8:20..8:21
//! warn-only: output
//! disabled: output
//! stdout: "a\n"
let d = {"k": "a", 1: "b"}
print(d["k"])
