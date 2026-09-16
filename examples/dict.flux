// Dictionary keys and values each have a consistent type.
let person: {string: string} = {
    "name": "John",
    "city": "New York"
}
let ages: {string: int} = {"John": 30, "Alice": 25}
print(person["name"])
print(ages["John"])
print(person["city"])

let numbers = {"number": 42}
let messages = {"text": "Hello"}
let flags = {"boolean": true}
print(numbers["number"])
print(messages["text"])
print(flags["boolean"])

// Nested dictionaries also use consistent value types.
let nested = {
    "person": {"name": "Alice", "city": "London"},
    "location": {"city": "London", "country": "UK"}
}
print(nested["person"]["name"])
print(nested["location"]["city"])
