print("Functions")
let double = fn(x) => x + x
let result = double(5)
print(result)

print("Add Strings")
let name = "Flux"
print("Hello, " + name)

let x = 5
let msg = if x > 0 then {
  print("x is positive")
  "yes"
} else {
  "no"
}
print(msg)