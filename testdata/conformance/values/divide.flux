//! rule: VAL-DIVIDE
//! about: / and % are not yet operators
//! status: planned
//! milestone: phase-3
//! specified: output
//! specified-stdout: "3\n1\n"
//! all: static-error S_UNEXPECTED_TOKEN at 8:9..8:10
print(6 / 2)
print(7 % 2)
