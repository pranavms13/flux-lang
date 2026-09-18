//! rule: GRM-STATEMENT-SEPARATOR
//! about: ; is not yet a token, so it is rejected as text that forms none
//! status: planned
//! milestone: phase-3
//! specified: output
//! specified-stdout: "1\n2\n"
//! all: static-error S_INVALID_CHARACTER at 8:9..8:10
print(1); print(2)
