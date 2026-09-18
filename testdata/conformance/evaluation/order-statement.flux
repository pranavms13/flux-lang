//! rule: EVL-ORDER-STATEMENT
//! about: statements run in source order
//! status: implemented
//! all: output
//! stdout: "one\ntwo\nthree\n"
print("one")
print("two")
print("three")
