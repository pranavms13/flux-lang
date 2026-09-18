//! rule: EVL-TOP-LEVEL-DISPLAY
//! about: a non-void expression statement displays its value; a declaration and a void one do not
//! status: implemented
//! all: output
//! stdout: "1\n2\np\n"
let x = 1
x
1 + 1
print("p")
