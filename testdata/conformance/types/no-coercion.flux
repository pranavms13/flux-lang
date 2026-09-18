//! rule: TYP-NO-COERCION
//! about: nothing converts between int and string, in either direction
//! status: implemented
//! all: static-error T_OPERAND_TYPE at 7:7..7:14
//! warn-only: runtime-error R_OPERAND_TYPE at 7:9..7:14
//! disabled: runtime-error R_OPERAND_TYPE at 7:9..7:14
print(1 + "1")
