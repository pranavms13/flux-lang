//! rule: VAL-INT-OVERFLOW
//! about: arithmetic past the range wraps around instead of being reported
//! status: planned
//! milestone: phase-3
//! specified: runtime-error R_INT_OVERFLOW at 8:27..8:30
//! all: output
//! stdout: "-9223372036854775808\n"
print(9223372036854775807 + 1)
