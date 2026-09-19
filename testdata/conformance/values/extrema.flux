//! rule: VAL-INT
//! about: both extrema and signed remainder are portable
//! status: implemented
//! all: output
//! stdout: "-9223372036854775808\n9223372036854775807\n-3\n-1\n0\n"
print(-9223372036854775808)
print(9223372036854775807)
print(-7/2)
print(-7%2)
print(-9223372036854775808%-1)
