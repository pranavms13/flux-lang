//! rule: VAL-LIST-INDEX
//! about: a list is indexed by an int and by nothing else
//! status: implemented
//! all: static-error T_INDEX_TYPE at 8:10..8:13
//! warn-only: runtime-error R_INDEX_TYPE at 8:9..8:14
//! disabled: runtime-error R_INDEX_TYPE at 8:9..8:14
let xs = [1]
print(xs["a"])
