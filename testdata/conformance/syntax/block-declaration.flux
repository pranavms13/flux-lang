//! rule: GRM-BLOCK-DECLARATION
//! about: blocks support scoped declarations and return their last item
//! status: implemented
//! all: output
//! stdout: "3\n"
print({ let x = 3
x })
