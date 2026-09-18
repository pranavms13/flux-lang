//! rule: EVL-IF-BRANCH
//! about: only the selected branch is evaluated
//! status: implemented
//! all: output
//! stdout: "taken\ntaken\n"
let trace = fn(v) => { print(v) v }
print(if true then trace("taken") else trace("skipped"))
