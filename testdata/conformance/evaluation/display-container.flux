//! rule: EVL-DISPLAY-CONTAINER
//! about: containers currently display in the host's form, not the specified one
//! status: planned
//! milestone: phase-3
//! specified: output
//! specified-stdout: "[1, 2]\n{\"a\": 1}\n"
//! all: output
//! stdout: "[1 2]\nmap[a:1]\n"
print([1, 2])
print({"a": 1})
