//! rule: DIA-CODE
//! about: the prefix names the stage, so one mistake is B_ statically and R_ at run time
//! status: implemented
//! all: static-error B_UNDEFINED_VARIABLE at 7:7..7:14
//! warn-only: runtime-error R_UNDEFINED_VALUE at 7:7..7:14
//! disabled: runtime-error R_UNDEFINED_VALUE at 7:7..7:14
print(missing)
