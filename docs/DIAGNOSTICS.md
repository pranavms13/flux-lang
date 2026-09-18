# Diagnostic codes

Every diagnostic Flux reports carries a code. The code is the stable part: the
wording of a message may improve, but a code keeps its meaning, so it is what to
search for, assert on in a test, or filter by in an editor.

A code's prefix names the stage that reports it, which usually tells you what
kind of mistake it is before you look it up.

This file is generated from the code registry. To update it, run:

```sh
FLUX_UPDATE_DOCS=1 go test -run TestDiagnosticReferenceIsCurrent .
```

## Syntax — `S_`

The source could not be read as Flux. Reported before anything is checked or run.

| Code | Meaning |
| --- | --- |
| `S_INVALID_CHARACTER` | the source contains text that is not a Flux token |
| `S_UNEXPECTED_EOF` | the source ends in the middle of a construct |
| `S_UNEXPECTED_TOKEN` | a token appears where the grammar does not allow it |

## Binding — `B_`

A name does not resolve, or is declared more than once. Reported while checking, or while running for a name that only some paths define.

| Code | Meaning |
| --- | --- |
| `B_DUPLICATE_PARAMETER` | a function declares the same parameter name twice |
| `B_UNDEFINED_VARIABLE` | a name was used where nothing declares it |

## Type — `T_`

A value does not have the type its use requires. Reported while checking, and downgraded to a warning or suppressed entirely by the configured mode.

| Code | Meaning |
| --- | --- |
| `T_ANNOTATION_MISMATCH` | a value does not have the type its declaration annotates |
| `T_ARGUMENT_COUNT` | a function was called with the wrong number of arguments |
| `T_ARGUMENT_TYPE` | an argument's type does not match the parameter it is passed to |
| `T_BRANCH_MISMATCH` | the branches of a conditional produce different types |
| `T_COMPARISON_MISMATCH` | two values of different types were compared |
| `T_CONDITION_TYPE` | the condition of a conditional is not a bool |
| `T_DICT_KEY_TYPE` | a dictionary key does not have the type of the other keys |
| `T_DICT_VALUE_TYPE` | a dictionary value does not have the type of the other values |
| `T_INDEX_TYPE` | a collection was indexed with the wrong type |
| `T_INVALID_ANNOTATION` | a type annotation does not name a type |
| `T_INVALID_DICT_KEY` | a dictionary key is not an int, string, or bool |
| `T_LIST_ELEMENT_TYPE` | a list element does not have the type of the other elements |
| `T_NOT_CALLABLE` | a value that is not a function was called |
| `T_NOT_INDEXABLE` | a value that is not a list or dictionary was indexed |
| `T_OPERAND_TYPE` | an operator was applied to types it does not accept |
| `T_RETURN_MISMATCH` | a function body does not produce the return type it declares |

## Runtime — `R_`

The program was accepted and then failed while running. Reported identically by the interpreter, the VM, and a generated executable.

| Code | Meaning |
| --- | --- |
| `R_ARGUMENT_COUNT` | a function was called with the wrong number of arguments |
| `R_INDEX_RANGE` | a list index is outside the list |
| `R_INDEX_TYPE` | a list was indexed with something other than an int |
| `R_MISSING_KEY` | a dictionary has no entry for the key it was given |
| `R_NOT_CALLABLE` | a value that is not a function was called |
| `R_NOT_INDEXABLE` | a value that is not a list or dictionary was indexed |
| `R_OPERAND_TYPE` | an operator was applied to values it does not accept |
| `R_UNDEFINED_VALUE` | a name was evaluated that nothing has bound |

## Internal — `X_`

A defect in Flux itself. A program should not be able to provoke one; if yours does, that is worth reporting.

| Code | Meaning |
| --- | --- |
| `X_INTERNAL` | an unexpected failure inside Flux; please report it |
