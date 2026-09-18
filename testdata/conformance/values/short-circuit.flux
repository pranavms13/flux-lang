//! rule: VAL-LOGICAL
//! about: skipped operands neither divide nor print
//! status: implemented
//! all: output
//! stdout: "false\ntrue\nfalse\n"
print(false && (1/0>0))
print(true || (1/0>0))
print(false && {print("bad");true})
