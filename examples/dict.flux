// Create a dictionary
let dict = {
    "name": "John",
    "age": 30,
    "city": "New York"
}

// Access dictionary values
print(dict["name"])
print(dict["age"])
print(dict["city"])

// Dictionaries can have mixed value types
let mixed = {
    "number": 42,
    "text": "Hello",
    "boolean": true
}

print(mixed["number"])
print(mixed["text"])
print(mixed["boolean"])

// Nested dictionaries
let nested = {
    "person": {
        "name": "Alice",
        "age": 25
    },
    "location": {
        "city": "London",
        "country": "UK"
    }
}

print(nested["person"]["name"])
print(nested["location"]["city"])
