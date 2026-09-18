//! rule: VAL-VOID
//! about: void compares equal to void, and both operands are evaluated
//! status: implemented
//! all: output
//! stdout: "a\nb\ntrue\n"
print(print("a") == print("b"))
