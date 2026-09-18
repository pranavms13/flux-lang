//! rule: VAL-DICT-DUPLICATE
//! about: the last pair with a repeated key wins, and both values are evaluated
//! status: implemented
//! all: output
//! stdout: "evaluated\nevaluated\n2\n"
let seen = fn(v) => { print("evaluated") v }
let d = {"k": seen(1), "k": seen(2)}
print(d["k"])
