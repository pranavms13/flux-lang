//! rule: GRM-LIST
//! about: [] is the empty list, so indexing it finds no element rather than no collection
//! status: implemented
//! all: runtime-error R_INDEX_RANGE at 5:3..5:6
[][0]
