//! rule: LEX-INT-RANGE
//! about: an integer literal too wide for 64 bits is rejected while reading
//! status: implemented
//! all: static-error S_INT_RANGE at 5:7..5:27
print(99999999999999999999)
