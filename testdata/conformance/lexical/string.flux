//! rule: LEX-STRING
//! about: a string literal spans lines and accepts the specified escapes
//! status: implemented
//! all: output
//! stdout: "a\tb\n\"c\"\\\nover\ntwo\n"
print("a\tb")
print("\"c\"\\")
print("over
two")
