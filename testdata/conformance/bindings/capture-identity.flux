//! rule: BND-CAPTURE-IDENTITY
//! about: an inner binding must not change what an existing closure already resolved
//! status: planned
//! milestone: phase-3
//! specified: output
//! specified-stdout: "1\n"
//! all: static-error S_UNEXPECTED_TOKEN at 10:18..10:21
let y = 1
let g = fn() => y
let shadowed = { let y = 2
g() }
print(shadowed)
