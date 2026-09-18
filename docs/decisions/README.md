# Decision records

One file per decision about what the language means. A record exists so that a
rule in [the specification](../SPEC.md) can be read together with the reason it
is that way, and so that a decision cannot be quietly reversed by an
implementation change: `TestDecisionsAreLinked` checks that every rule a record
names still exists, that every planned rule has a record, and that every
migration a record points at is written.

Each record states its status, the rules it decides, the phase that implements
it, the slice of [the plan](../PLAN.md) that carries it, and where its migration
notes are. A record for behavior Flux already has says `Phase: implemented`; its
job is to stop the behavior from being changed by accident.

| # | Decision | Phase | Rules |
| --- | --- | --- | --- |
| D-01 | [Top-level expression output](top-level-output.md) | implemented | `EVL-TOP-LEVEL-DISPLAY`, `EVL-PRINT` |
| D-02 | [Blocks and their result](blocks.md) | 3 | `GRM-BLOCK`, `GRM-BLOCK-DECLARATION`, `EVL-BLOCK-RESULT` |
| D-03 | [What `{}` means](empty-braces.md) | implemented | `GRM-EMPTY-BRACES` |
| D-04 | [Statement boundaries](statement-boundaries.md) | 3 | `LEX-WHITESPACE`, `GRM-ADJACENCY`, `GRM-STATEMENT-SEPARATOR` |
| D-05 | [Evaluation order](evaluation-order.md) | implemented | the five `EVL-ORDER-…` and `EVL-IF-BRANCH` rules |
| D-06 | [Bindings are immutable and declared once](bindings.md) | 3 | `BND-LET`, `BND-REDECLARE`, `BND-PARAMETER-SCOPE` |
| D-07 | [A captured name keeps its binding](closure-lookup.md) | 3 | `BND-CAPTURE`, `BND-CAPTURE-IDENTITY` |
| D-08 | [Forward references and self-recursion](forward-references.md) | 3 | `BND-FORWARD-REFERENCE`, `BND-SELF-RECURSION` |
| D-09 | [Integers are 64-bit and checked](integers.md) | 3 | `VAL-INT`, `VAL-INT-OVERFLOW` |
| D-10 | [Division and remainder](division-and-remainder.md) | 3 | `VAL-DIVIDE` |
| D-11 | [Unary negation](unary-negation.md) | 3 | `VAL-NEG` |
| D-12 | [Booleans and conditions](booleans.md) | 3 | `VAL-LOGICAL`, `EVL-IF-TRUTHY`, `EVL-IF-BOOL` |
| D-13 | [Equality](equality.md) | 3 | `VAL-EQUALITY`, `VAL-EQUALITY-FUNCTION` |
| D-14 | [Collections](collections.md) | implemented | the `VAL-LIST-…`, `VAL-DICT-…` and `VAL-IMMUTABLE` rules |
| D-15 | [No implicit conversion](conversion.md) | implemented | `TYP-NO-COERCION` |
| D-16 | [How values are displayed](display.md) | 3 | the three `EVL-DISPLAY-…` rules |
| D-17 | [Inference constrains unannotated parameters](inference.md) | 4 | `TYP-INFERRED`, `TYP-INFERENCE-CONSTRAINTS` |

D-01 through D-16 correspond to the decision table in section 6 of
[the plan](../PLAN.md), with two additions that writing the specification
produced: D-11, because a grammar with subtraction and no negation could not be
stated without looking like an omission, and D-17, because D-08 commits to
typing a recursive function from its annotation and Phase 4 has to keep that
working once annotations are optional.
