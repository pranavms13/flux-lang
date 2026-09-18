//! rule: VAL-DICT-MISSING
//! about: reading an absent key fails rather than producing a default
//! status: implemented
//! all: runtime-error R_MISSING_KEY at 6:8..6:13
let d = {"a": 1}
print(d["b"])
