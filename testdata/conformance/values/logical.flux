//! rule: VAL-LOGICAL
//! about: ! && and || are not yet operators
//! status: planned
//! milestone: phase-3
//! specified: output
//! specified-stdout: "false\ntrue\nfalse\n"
//! all: static-error S_UNEXPECTED_TOKEN at 8:7..8:8
print(!true)
print(true || false)
print(true && false)
