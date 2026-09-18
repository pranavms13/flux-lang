//! rule: GRM-DICT
//! about: a dictionary literal is comma-separated key: value pairs in braces
//! status: implemented
//! all: output
//! stdout: "1\n2\n"
let d = {"a": 1, "b": 2}
print(d["a"])
print(d["b"])
