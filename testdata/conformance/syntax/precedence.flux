//! rule: GRM-PRECEDENCE
//! about: comparison binds looser than addition, so a - b < c groups as (a - b) < c
//! status: implemented
//! all: output
//! stdout: "true\nfalse\n"
print(5 - 2 < 4)
print(1 + 1 == 3)
