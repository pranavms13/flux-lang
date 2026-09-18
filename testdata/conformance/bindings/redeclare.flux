//! rule: BND-REDECLARE
//! about: a second declaration of a name currently replaces the first in silence
//! status: planned
//! milestone: phase-3
//! specified: static-error B_DUPLICATE_DECLARATION at 9:1..9:10
//! all: output
//! stdout: "2\n"
let x = 1
let x = 2
print(x)
