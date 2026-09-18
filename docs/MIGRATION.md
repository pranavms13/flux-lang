# Migration notes

Every decision that changes what an existing Flux program does has a section
here, with the program before and after. A decision that only adds syntax —
`/`, `%`, unary `-`, `!`, `&&`, `||`, `;`, `let` inside a block — is not listed:
nothing that parses today means something different afterwards.

Each section names the [decision](decisions/) it comes from and the
[specification rule](SPEC.md) it implements. `TestDecisionsAreLinked` fails if a
decision points at a section that does not exist here.

Nothing in this file has happened yet. These are the changes Phase 3 and Phase 4
carry out, written down while the reasons are fresh.

## Redeclaring a name

From [D-06](decisions/bindings.md) — rule `BND-REDECLARE`, phase 3.

A name may be declared once per scope. A second `let` for the same name in the
same scope is rejected instead of silently replacing the first.

```flux
// before — the second declaration replaces the first
let total = 1
let total = total + 1
print(total)          // 2
```

```flux
// after — name the second value, or shadow in an inner scope
let total = 1
let adjusted = total + 1
print(adjusted)       // 2
```

Shadowing in an inner scope is still allowed, and is a different name rather
than a new value for the same one.

## Late name rebinding

From [D-07](decisions/closure-lookup.md) — rule `BND-CAPTURE-IDENTITY`, phase 3.

A function keeps the binding each of its free names resolved to when the
function was written. Redeclaring that name later cannot change what the
function sees.

```flux
// before — g sees whichever y is current when it runs
let y = 1
let g = fn() => y
let y = 2
print(g())            // 2
```

```flux
// after — the outer y is rejected as a redeclaration (see above); once it is
// shadowed in an inner scope instead, g still sees the binding it captured
let y = 1
let g = fn() => y
let shadowed = { let y = 2
                 g() }
print(shadowed)       // 1
```

In practice most programs never noticed the old behavior, because noticing it
requires redeclaring a name a closure has already captured.

## Recursive functions

From [D-08](decisions/forward-references.md) — rules `BND-FORWARD-REFERENCE` and
`BND-SELF-RECURSION`, phase 3.

A function may call itself by the name it is being bound to, if that binding
carries a return type annotation. An ordinary forward reference to a name
declared later stays an error, in every mode.

```flux
// before — rejected by the checker, and only runs with checking relaxed
let sum = fn(n: int): int => if n > 0 then n + sum(n - 1) else 0
print(sum(3))
```

```flux
// after — the same program, accepted in every mode
let sum = fn(n: int): int => if n > 0 then n + sum(n - 1) else 0
print(sum(3))         // 6
```

This is a widening: a program that worked under `warnOnly` or with checking
disabled keeps working, and now works under strict checking too. The change that
can break a program is the other half — a forward reference to a non-function
name no longer resolves at run time when checking is relaxed.

```flux
// before, with checking disabled — resolved when g was called
let g = fn() => y
let y = 10
print(g())            // 10

// after — y must be declared before the function that refers to it
let y = 10
let g = fn() => y
print(g())            // 10
```

## Integer overflow

From [D-09](decisions/integers.md) — rule `VAL-INT-OVERFLOW`, phase 3.

An operation whose result does not fit a signed 64-bit integer is reported as
`R_INT_OVERFLOW` at the operator, instead of wrapping.

```flux
// before
print(9223372036854775807 + 1)   // -9223372036854775808
```

```flux
// after
print(9223372036854775807 + 1)
// error[R_INT_OVERFLOW]: + overflowed the range of int
```

`int` also becomes 64-bit on every target rather than following the build
machine, so a program compiled for a 32-bit target stops wrapping at a different
place from the same program on a 64-bit one.

## Truthy conditions

From [D-12](decisions/booleans.md) — rule `EVL-IF-BOOL`, phase 3.

A condition must be a `bool` in every mode. Outside strict mode a non-`bool`
condition is currently accepted with a warning and read for truthiness.

```flux
// before — 0 and "" are false, anything else is true
print(if count then "some" else "none")
```

```flux
// after — say what the test is
print(if count > 0 then "some" else "none")
```

Strict mode already rejected these, so a program that passes strict checking
today is unaffected.

## Comparing functions

From [D-13](decisions/equality.md) — rule `VAL-EQUALITY-FUNCTION`, phase 3.

Comparing two functions, or two containers holding functions, is rejected as
`T_INCOMPARABLE` by the checker, or `R_INCOMPARABLE` at run time when checking
is relaxed or disabled, instead of producing an answer derived from host structure.

```flux
// before
let f = fn(x) => x
let g = fn(x) => x
print(f == g)         // false
print(f == f)         // true
```

```flux
// after
print(f == g)
// error[T_INCOMPARABLE]: fn(unknown) -> unknown cannot be compared
```

Neither of the old answers was usable: `f == f` was true because the comparison
reached the same host pointer, and `f == g` was false because it reached a
different one.

## Displayed containers

From [D-16](decisions/display.md) — rules `EVL-DISPLAY-CONTAINER` and
`EVL-DISPLAY-OPAQUE`, phase 3.

A displayed list or dictionary takes the shape a Flux program would write, and a
displayed function or void value becomes a fixed placeholder with no host
address in it.

```flux
// before
print([1, 2])         // [1 2]
print({"a": 1})       // map[a:1]
print(print("x"))     // x, then <nil>
```

```flux
// after
print([1, 2])         // [1, 2]
print({"a": 1})       // {"a": 1}
print(print("x"))     // x, then <void>
```

Scalar output does not change, so a program that only prints numbers, strings
and booleans is unaffected. Anything that reads container output — a test, a
script parsing another script's output — has to be updated.

A displayed function changes from a host address such as `&{0x14000100300 map[]}`
to `<function>`. The old text was not stable between runs, and the interpreter
and the VM did not even produce the same one, so nothing could have depended on
it.

## Unannotated parameters

From [D-17](decisions/inference.md) — rule `TYP-INFERENCE-CONSTRAINTS`, phase 4.

An unannotated parameter is constrained by how the body uses it, instead of
being treated as compatible with every type.

```flux
// before — accepted by the checker, fails inside the body at run time
let f = fn(x) => x + 1
print(f("a"))
// error[R_OPERAND_TYPE]: cannot apply + to string and int
```

```flux
// after — reported where the wrong argument is written
print(f("a"))
// error[T_ARGUMENT_TYPE]: argument 1 has type string, expected int
```

The programs affected are the ones that were already wrong. A correct program
gains earlier and better-placed diagnostics; it does not need to be changed.
