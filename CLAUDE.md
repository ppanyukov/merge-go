# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run a single test
go test -v -run TestName ./merge/

# Run tests with coverage
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

## Architecture

Single-package library (`merge/`) implementing deep clone and deep merge via Go's `reflect` package.

**Public API** (`merge/merge.go`):
- `MergeTagged[T](base, overlay T) (T, error)` — deep merge; overlay wins for most types
- `MustMergeTagged[T](base, overlay T) T` — panics on error
- `DeepClone[T](base T) (T, error)` — deep clone
- `MustDeepClone[T](base T) T` — panics on error

**Merge semantics:**
- Simple values: overlay wins
- Pointers: overlay wins if non-nil; base used otherwise (nil = "not set")
- Structs: field-by-field recursion; private fields skipped
- Slices: overlay appended to base (concatenation)
- Maps: merged by key; overlay key wins
- Arrays: element-by-element merge
- Struct tag `merge:"atomic_object"` on a pointer field: overlay wins completely if non-nil (no recursive merge); only valid on pointer fields

**Key invariant:** All outputs are deep copies — no shared references between inputs and the result.

The core dispatch is `mergeTaggedReflect` which switches on `reflect.Kind` and delegates to type-specific helpers. `deepClone` follows the same pattern for cloning only.
