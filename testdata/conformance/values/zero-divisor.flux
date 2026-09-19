//! rule: VAL-DIVIDE
//! about: a required right operand still reports division by zero
//! status: implemented
//! all: runtime-error R_ZERO_DIVISOR at 5:17..5:19
print(true && (1/0>0))
