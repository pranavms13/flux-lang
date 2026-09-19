//! rule: VAL-DICT-KEY-TYPE
//! about: a dictionary key is an int, a string or a bool, never a function
//! status: implemented
//! all: static-error T_INVALID_DICT_KEY at 9:10..9:11
//! warn-only: runtime-error R_INVALID_DICT_KEY at 9:10..9:11
//! disabled: runtime-error R_INVALID_DICT_KEY at 9:10..9:11
//! stdout: ""
let f = fn(x) => x
let d = {f: 1}
