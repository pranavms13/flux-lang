//! rule: VAL-INT
//! about: int spans the signed 64-bit range
//! status: implemented
//! all: output
//! stdout: "9223372036854775807\n-9223372036854775808\n"
print(9223372036854775807)
print(0 - 9223372036854775807 - 1)
