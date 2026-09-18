//! rule: LEX-INT-RANGE
//! about: an integer literal too wide for 64 bits is rejected while reading
//! status: implemented
//! all: static-error S_UNEXPECTED_TOKEN at 5:7..5:7
print(99999999999999999999)
