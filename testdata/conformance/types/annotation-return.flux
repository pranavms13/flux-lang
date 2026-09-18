//! rule: TYP-ANNOTATION-RETURN
//! about: the body is what disagrees with the declared return type
//! status: implemented
//! all: static-error T_RETURN_MISMATCH at 8:22..8:25
//! warn-only: output
//! disabled: output
//! stdout: ""
let f = fn(): int => "s"
