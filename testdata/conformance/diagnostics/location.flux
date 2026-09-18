//! rule: DIA-LOCATION
//! about: the failing index is underlined, not the expression that contains it
//! status: implemented
//! all: runtime-error R_INDEX_RANGE at 6:17..6:20
let xs = [1, 2]
print(xs[0] - xs[9])
