//! rule: BND-FORWARD-REFERENCE
//! about: the checker rejects a forward reference, but relaxing it lets one resolve at run time
//! status: planned
//! milestone: phase-3
//! specified: static-error B_UNDEFINED_VARIABLE at 11:17..11:18
//! strict: static-error B_UNDEFINED_VARIABLE at 11:17..11:18
//! lenient: static-error B_UNDEFINED_VARIABLE at 11:17..11:18
//! warn-only: output
//! disabled: output
//! stdout: "10\n"
let g = fn() => y
let y = 10
print(g())
