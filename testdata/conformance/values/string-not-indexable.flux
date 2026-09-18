//! rule: VAL-STRING
//! about: a string is not a collection, so it cannot be indexed
//! status: implemented
//! all: static-error T_NOT_INDEXABLE at 7:7..7:12
//! warn-only: runtime-error R_NOT_INDEXABLE at 7:12..7:15
//! disabled: runtime-error R_NOT_INDEXABLE at 7:12..7:15
print("abc"[0])
