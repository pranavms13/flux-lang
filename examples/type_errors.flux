// Example demonstrating type errors in Flux

// Type mismatch: trying to assign string to int variable
let x: int = "hello"

// Type mismatch in function call
let add = fn(a: int, b: int): int => a + b
let result = add("5", 10)

// Type mismatch in list elements
let mixed: [int] = [1, "two", 3]

// Type mismatch in dictionary values
let data: {string: int} = {"name": "Alice", "age": 25}

// Type mismatch in conditional branches
let status = if true then {
  42
} else {
  "inactive"
}

// Wrong argument count
let addNumbers = fn(a: int, b: int): int => a + b
let badResult = addNumbers(5)

// Indexing with wrong type
let numbers = [1, 2, 3]
let element = numbers["first"] 