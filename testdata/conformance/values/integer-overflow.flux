//! rule: VAL-INT-OVERFLOW
//! about: arithmetic outside the signed 64-bit range reports overflow
//! status: implemented
//! all: runtime-error R_INT_OVERFLOW at 5:27..5:30
print(9223372036854775807 + 1)
