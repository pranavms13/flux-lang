//! rule: GRM-IF
//! about: else is required, because every conditional has a value
//! status: implemented
//! all: static-error S_UNEXPECTED_TOKEN at 5:21..5:22
print(if true then 1)
