//! rule: EVL-PRINT
//! about: print writes one line at the moment it is called
//! status: implemented
//! all: output
//! stdout: "before\ninside\nafter\n"
let f = fn() => { print("inside") 0 }
print("before")
let r = f()
print("after")
