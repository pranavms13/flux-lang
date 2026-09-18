//! rule: EVL-DISPLAY-OPAQUE
//! about: a void value currently displays as a host placeholder
//! status: planned
//! milestone: phase-3
//! specified: output
//! specified-stdout: "x\n<void>\n"
//! all: output
//! stdout: "x\n<nil>\n"
print(print("x"))
