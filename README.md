# merge-go

**WORK IN PROGRESS**

Deep clone and deep merge Go objects in an overlay manner. Results are full copies, not pointers to originals.

## Install

```bash
go get github.com/ppanyukov/merge-go
```

## Functions

```go
// Deep merge: overlay is applied on top of base.
MergeTagged[T any](base, overlay T) (T, error)
MustMergeTagged[T any](base, overlay T) T  // panics on error

// Deep clone.
DeepClone[T any](base T) (T, error)
MustDeepClone[T any](base T) T
```

## Merge semantics

| Type | Behavior |
|------|----------|
| Simple values (`int`, `string`, …) | Overlay wins |
| Pointers | Overlay wins if non-nil; otherwise base is used |
| Structs | Merged field-by-field recursively |
| Slices | Overlay appended to base |
| Maps | Keys merged; overlay keys overwrite base keys |
| Arrays | Merged element-by-element |

Private struct fields are skipped.

## Tags

```go
// Field is replaced atomically: overlay wins if non-nil, otherwise base is kept.
// Supported on pointer, slice, map, interface fields.
Field SomeType `merge:"atomic_object"`
```

## Example

```go
type Address struct {
    City  string
    State string
}

type Person struct {
    Name    string
    Age     *int   // pointer: nil means "not set"
    Tags    []string
    Address *Address
}

age := 30
base := Person{
    Name:    "Alice",
    Age:     &age,
    Tags:    []string{"admin"},
    Address: &Address{City: "Portland", State: "OR"},
}

overlay := Person{
    Name:    "Alice (updated)",
    Age:     nil,                    // not set: base wins
    Tags:    []string{"user"},
    Address: &Address{City: "Seattle"},
}

result := merge.MustMergeTagged(base, overlay)
// result.Name    => "Alice (updated)"           // overlay wins (simple type)
// result.Age     => &30                         // overlay nil, base kept
// result.Tags    => ["admin", "user"]           // appended
// result.Address => &{City:"Seattle", State:""}    // merged field-by-field; State overlay "" wins
```
