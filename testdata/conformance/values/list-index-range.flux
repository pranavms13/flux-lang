//! rule: VAL-LIST-INDEX
//! about: an index outside the list fails rather than returning a default
//! status: implemented
//! all: runtime-error R_INDEX_RANGE at 6:9..6:12
let xs = [1]
print(xs[5])
