//! rule: EVL-DISPLAY-OPAQUE
//! about: void displays with a stable placeholder
//! status: implemented
//! all: output
//! stdout: "x\n<void>\n"
print(print("x"))
