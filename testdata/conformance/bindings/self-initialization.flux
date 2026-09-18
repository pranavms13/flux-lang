//! rule: BND-LET
//! about: a binding cannot read its own unfinished initializer
//! status: implemented
//! all: static-error B_SELF_INITIALIZATION at 5:7..5:8
let x=x
