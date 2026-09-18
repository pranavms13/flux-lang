//! rule: BND-SELF-RECURSION
//! about: the checker reports the name a function is being bound to as undefined
//! status: planned
//! milestone: phase-3
//! specified: output
//! specified-stdout: "6\n"
//! strict: static-error T_OPERAND_TYPE at 12:44..12:58
//! lenient: static-error T_OPERAND_TYPE at 12:44..12:58
//! warn-only: output
//! disabled: output
//! stdout: "6\n"
let sum = fn(n: int): int => if n > 0 then n + sum(n - 1) else 0
print(sum(3))
