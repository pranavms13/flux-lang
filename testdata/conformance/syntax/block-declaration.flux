//! rule: GRM-BLOCK-DECLARATION
//! about: a block cannot yet contain a declaration
//! status: planned
//! milestone: phase-3
//! specified: output
//! specified-stdout: "3\n"
//! all: static-error S_UNEXPECTED_TOKEN at 8:9..8:12
print({ let x = 3
x })
