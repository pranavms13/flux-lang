//! rule: TYP-ANNOTATION-LET
//! about: the value is what is wrong, so the value is what is underlined
//! status: implemented
//! all: static-error T_ANNOTATION_MISMATCH at 8:14..8:21
//! warn-only: output
//! disabled: output
//! stdout: ""
let x: int = "hello"
