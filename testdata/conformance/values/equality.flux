//! rule: VAL-EQUALITY
//! about: scalars compare by value and containers compare element by element
//! status: implemented
//! all: output
//! stdout: "true\ntrue\nfalse\n"
print([1, 2] == [1, 2])
print({"a": 1} == {"a": 1})
print(1 == 2)
