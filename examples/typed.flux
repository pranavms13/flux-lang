// Example demonstrating type safety in Flux

// Basic typed variable declarations
let x: int = 42
let name: string = "Flux"
let isActive: bool = true

print("Basic types:")
print(x)
print(name)

// Typed function declarations
let add: fn(int, int) -> int = fn(a: int, b: int): int => a + b
let result: int = add(10, 20)
print("Add result:")
print(result)

// List with type annotation
let numbers: [int] = [1, 2, 3, 4, 5]
print("Numbers list:")
print(numbers[0])

// Dictionary with type annotation
let person: {string: string} = {"name": "Alice", "city": "Tokyo"}
print("Person name:")
print(person["name"])

// Function with typed parameters and return type
let greet = fn(name: string): string => "Hello, " + name
let greeting: string = greet("World")
print(greeting)

// Conditional with type checking
let status: string = if x > 0 then {
  "positive"
} else {
  "non-positive"
}
print("Status:")
print(status) 