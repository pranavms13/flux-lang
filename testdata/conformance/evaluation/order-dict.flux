//! rule: EVL-ORDER-DICT
//! about: pairs are evaluated in source order, and the key before its value
//! status: implemented
//! all: output
//! stdout: "k1\nv1\nk2\nv2\nv1\n"
let trace = fn(v) => { print(v) v }
let d = {trace("k1"): trace("v1"), trace("k2"): trace("v2")}
print(d["k1"])
