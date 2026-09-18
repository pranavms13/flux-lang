//! rule: VAL-NEG
//! about: there is no unary minus, so -5 is not a literal or an expression
//! status: planned
//! milestone: phase-3
//! specified: output
//! specified-stdout: "-5\n"
//! all: static-error S_UNEXPECTED_TOKEN at 8:7..8:8
print(-5)
