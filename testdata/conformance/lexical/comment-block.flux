//! rule: LEX-COMMENT-BLOCK
//! about: a block comment ends at the first */, so it does not nest
//! status: implemented
//! all: output
//! stdout: "1\n"
/* outer /* inner */ print(1)
