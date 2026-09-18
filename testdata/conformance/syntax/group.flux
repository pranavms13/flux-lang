//! rule: GRM-GROUP
//! about: parentheses override the left grouping of equal precedence
//! status: implemented
//! all: output
//! stdout: "-4\n2\n"
print(1 - 2 - 3)
print(1 - (2 - 3))
