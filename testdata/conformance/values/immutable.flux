//! rule: VAL-IMMUTABLE
//! about: there is no assignment, so nothing can change a collection in place
//! status: implemented
//! all: static-error S_UNEXPECTED_TOKEN at 6:7..6:8
let xs = [1]
xs[0] = 9
