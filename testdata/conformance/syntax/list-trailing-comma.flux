//! rule: GRM-LIST
//! about: a list literal does not permit a trailing comma
//! status: implemented
//! all: static-error S_UNEXPECTED_TOKEN at 5:16..5:17
let xs = [1, 2,]
