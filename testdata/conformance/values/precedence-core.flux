//! rule: GRM-PRECEDENCE
//! about: all operator levels and left associative arithmetic
//! status: implemented
//! all: output
//! stdout: "7\n5\n-6\nfalse\ntrue\n"
print(1+2*3)
print(10-3-2)
print(-2*3)
print(!(1<2))
print(1<2 == true && false || 3>=3)
