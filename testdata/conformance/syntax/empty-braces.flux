//! rule: GRM-EMPTY-BRACES
//! about: {} is the empty dictionary, so indexing it looks for a missing key
//! status: implemented
//! all: runtime-error R_MISSING_KEY at 5:3..5:8
{}["a"]
