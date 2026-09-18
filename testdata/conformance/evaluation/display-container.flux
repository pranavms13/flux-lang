//! rule: EVL-DISPLAY-CONTAINER
//! about: containers display in Flux syntax with quoted nested strings
//! status: implemented
//! all: output
//! stdout: "[1, 2]\n{\"a\": 1}\n"
print([1, 2])
print({"a": 1})
