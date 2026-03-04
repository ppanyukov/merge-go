// Package merge provides functions to deep clone and deep merge objects in overlay manner.
// The results are full copies, not pointers to the originals.
package merge

import (
	"reflect"

	"github.com/pkg/errors"
)

// MustDeepClone returns a deep clone of baseInput, panicking on error.
func MustDeepClone[T any](baseInput T) T {
	result, err := DeepClone(baseInput)
	if err != nil {
		panic(err)
	}
	return result
}

// DeepClone returns a deep clone of baseInput.
func DeepClone[T any](baseInput T) (T, error) {
	base := reflect.ValueOf(baseInput)
	res, err := deepClone(base)
	if err != nil {
		var zero T
		return zero, err
	}
	if res.CanAddr() && reflect.TypeOf(baseInput).Kind() == reflect.Pointer {
		return res.Addr().Interface().(T), nil
	}
	return res.Interface().(T), nil
}

// MustMergeTagged does a DEEP MERGE of two types in an overlay manner.
//
// Basics:
//   - If base is nil, overlay is used.
//   - If base is not nil, it is deeply merged with overlay.
//   - Both base and overlay must be of the same type.
//   - Pointers and Pointer-To-Pointer types are supported.
//   - Both base and overlay can be nil pointers.
//   - Both base and overlay can be same objects, in which case the result
//     effectively is a deep clone in most cases, but see comments on slices
//     and maps below.
//
// The default "not set" semantics are as follows:
//
//   - For pointers, nil means "not set". Any non-nil pointer is set.
//   - Non-pointer simple types always have value.
//   - Slices can be nil and "non-nil but still empty".
//     This is important distinction for `atomic_object` tags.
//   - Maps have similar to Slices semantics. Completely nil maps are nil.
//     The allocated but unpopulated maps are treated as an object for overlay.
//
// Default behavior for merge:
//
//   - Internal private fields are skipped.
//     TODO: support in the future?
//
//   - The merge is always on underlying value, not on pointers.
//
//   - Pointer-To-Pointer-To-Pointer-Etc types are supported.
//
//   - Simple value types (int, string) are merged atomically: overlay always wins.
//
//   - Pointers to simple types are merged atomically: If overlay is non-nil, it wins.
//
//   - Slices and pointers to slices. Overlay is appended to base.
//     Each value of both slices is deep-copied recursively into the resulted slice.
//
//   - Maps and pointers to maps are merged on key: Base+Overlay. The Overlay
//     keys overwrite the base keys. Each object under the key is deep-copied.
//     TODO: Do we actually MERGE values under common keys?
//
//   - Structs and pointers to structs are deep-merged recursively field-by-field.
//     Only exported fields are merged.
//
// Customizing behavior with tags. This applies to structs: whether standalone,
// or in slices or as map values.
//
//   - Fields with `merge: atomic_object` are treated as atomic.
//     If overlay is non-nil, it completely replaces base. Otherwise base
//     is used.
//
//   - Currently only objects that can be nil are supported, e.g. pointers to
//     struct, slices and maps. Structs which are values are not supported for now.
//
// TODO: next possible features:
//
//   - Support pointer merge semantics for non-pointer values. E.g. if overlay is
//     a default value, base wins. Maybe have `merge:"omitempty"` tag for this like in YAML?
//     This would follow yaml/json semantics for scalar types.
//
//   - Support private internal fields, maybe via tags and reflection hacks.
//
//   - Support map_deep_values tag to deep-merge map values instead of overlay overwriting base keys.
//     E.g.Policies map[string]Policy `merge:"map_deep_values"`.
//
//   - Support atomic_object tag for value structs (current support is only on *Structs).
//     This is easy to do and useful for embedded structs.
func MustMergeTagged[T any](baseInput, overlayInput T) T {
	result, err := MergeTagged(baseInput, overlayInput)
	if err != nil {
		panic(err)
	}
	return result
}

// MergeTagged does a DEEP MERGE of two types in an overlay manner.
// It is the non-panicking equivalent of MustMergeTagged.
func MergeTagged[T any](baseInput, overlayInput T) (T, error) {
	base := reflect.ValueOf(baseInput)
	overlay := reflect.ValueOf(overlayInput)

	res, err := mergeTaggedReflect(base, overlay)
	if err != nil {
		var zero T
		return zero, err
	}
	if res.CanAddr() && reflect.TypeOf(baseInput).Kind() == reflect.Pointer {
		return res.Addr().Interface().(T), nil
	}

	return res.Interface().(T), nil
}

func mergeTaggedReflect(base, overlay reflect.Value) (reflect.Value, error) {
	if base.Type() != overlay.Type() {
		return reflect.Value{}, errors.Errorf("type mismatch: base is %v, overlay is %v", base.Type(), overlay.Type())
	}

	switch base.Kind() {
	case reflect.Pointer:
		if base.IsNil() && overlay.IsNil() {
			return overlay, nil
		}

		if overlay.IsNil() {
			// Ensure deep copy of base
			val, err := deepClone(base.Elem())
			if err != nil {
				return reflect.Value{}, err
			}
			res := reflect.New(base.Type().Elem())
			res.Elem().Set(val)
			return res, nil
		}
		if base.IsNil() {
			// Ensure deep copy of overlay
			val, err := deepClone(overlay.Elem())
			if err != nil {
				return reflect.Value{}, err
			}
			res := reflect.New(base.Type().Elem())
			res.Elem().Set(val)
			return res, nil
		}

		merged, err := mergeTaggedReflect(base.Elem(), overlay.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		result := reflect.New(base.Type().Elem())
		result.Elem().Set(merged)
		return result, nil

	case reflect.Struct:
		return mergeStructs(base, overlay)

	case reflect.Map:
		return mergeMaps(base, overlay)

	case reflect.Slice:
		return mergeSlices(base, overlay)

	case reflect.Array:
		return mergeArrays(base, overlay)

	default:
		// All simple types: overlay wins
		return overlay, nil
	}
}

// deepClone performs a deep clone of the given value, handling pointers, structs, and maps.
// Resulted object is a complete copy of the input.
func deepClone(base reflect.Value) (reflect.Value, error) {
	switch base.Kind() {
	case reflect.Pointer:
		if base.IsNil() {
			return base, nil
		}
		// Always wrap in a new pointer
		cloned, err := deepClone(base.Elem())
		if err != nil {
			return reflect.Value{}, err
		}
		res := reflect.New(base.Type().Elem())
		res.Elem().Set(cloned)
		return res, nil

	case reflect.Struct:
		typ := base.Type()
		res := reflect.New(typ).Elem()
		for i := 0; i < typ.NumField(); i++ {
			val, err := deepClone(base.Field(i))
			if err != nil {
				return reflect.Value{}, err
			}
			res.Field(i).Set(val)
		}
		return res, nil

	case reflect.Map:
		if base.IsNil() {
			return base, nil
		}

		res := reflect.MakeMap(base.Type())
		for _, key := range base.MapKeys() {
			elem := base.MapIndex(key)
			elem, err := deepClone(elem)
			if err != nil {
				return reflect.Value{}, err
			}
			res.SetMapIndex(key, elem)
		}
		return res, nil

	case reflect.Slice:
		if base.IsNil() {
			return base, nil
		}

		res := reflect.MakeSlice(base.Type(), 0, base.Len())
		for i := 0; i < base.Len(); i++ {
			elem, err := deepClone(base.Index(i))
			if err != nil {
				return reflect.Value{}, err
			}
			res = reflect.Append(res, elem)
		}
		return res, nil

	case reflect.Array:
		res := reflect.New(base.Type()).Elem()
		for i := 0; i < base.Len(); i++ {
			val, err := deepClone(base.Index(i))
			if err != nil {
				return reflect.Value{}, err
			}
			res.Index(i).Set(val)
		}
		return res, nil

	default:
		// All simple types here, just return
		return base, nil
	}
}

func mergeStructs(base reflect.Value, overlay reflect.Value) (reflect.Value, error) {
	if base.Type() != overlay.Type() {
		return reflect.Value{}, errors.Errorf("type mismatch: base is %v, overlay is %v", base.Type(), overlay.Type())
	}
	if base.Kind() != reflect.Struct {
		return reflect.Value{}, errors.Errorf("expected reflect.Struct type here, got %v", base.Kind())
	}

	// Assume struct here. Recursively merge each field.
	typ := base.Type()
	out := reflect.New(typ).Elem()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tag := field.Tag.Get("merge")
		baseField := base.Field(i)
		overlayField := overlay.Field(i)

		if !field.IsExported() {
			// overlayField = reflect.NewAt(overlayField.Type(), unsafe.Pointer(overlayField.UnsafeAddr())).Elem()
			continue
		}

		switch tag {
		case "atomic_object":
			valToSet, err := mergeAtomicField(baseField, overlayField, field)
			if err != nil {
				return reflect.Value{}, err
			}
			out.Field(i).Set(valToSet)
		default:
			valToSet, err := mergeTaggedReflect(baseField, overlayField)
			if err != nil {
				return reflect.Value{}, err
			}
			out.Field(i).Set(valToSet)
		}
	}

	return out, nil
}

func mergeSlices(base reflect.Value, overlay reflect.Value) (reflect.Value, error) {
	if base.Kind() != reflect.Slice {
		return reflect.Value{}, errors.Errorf("expected reflect.Slice type here, got %v", base.Kind())
	}

	// We need to make a "deep" recursive copy of all elements in the slice, not
	// just copy the slice elements into another slice.
	res := reflect.MakeSlice(base.Type(), 0, base.Len()+overlay.Len())

	for i := 0; i < base.Len(); i++ {
		elem, err := deepClone(base.Index(i))
		if err != nil {
			return reflect.Value{}, err
		}
		res = reflect.Append(res, elem)
	}

	for i := 0; i < overlay.Len(); i++ {
		elem, err := deepClone(overlay.Index(i))
		if err != nil {
			return reflect.Value{}, err
		}
		res = reflect.Append(res, elem)
	}

	return res, nil
}

func mergeArrays(base reflect.Value, overlay reflect.Value) (reflect.Value, error) {
	if base.Kind() != reflect.Array {
		return reflect.Value{}, errors.Errorf("expected reflect.Array type here, got %v", base.Kind())
	}
	if base.Type() != overlay.Type() {
		return reflect.Value{}, errors.Errorf("type mismatch: base is %v, overlay is %v", base.Type(), overlay.Type())
	}

	// Result has the same type as the inputs. Merge element-by-element.
	res := reflect.New(base.Type()).Elem()
	for i := 0; i < base.Len(); i++ {
		merged, err := mergeTaggedReflect(base.Index(i), overlay.Index(i))
		if err != nil {
			return reflect.Value{}, err
		}
		res.Index(i).Set(merged)
	}

	return res, nil
}

func mergeMaps(base reflect.Value, overlay reflect.Value) (reflect.Value, error) {
	if base.Type() != overlay.Type() {
		return reflect.Value{}, errors.Errorf("type mismatch: base is %v, overlay is %v", base.Type(), overlay.Type())
	}

	if base.Kind() != reflect.Map {
		return reflect.Value{}, errors.Errorf("expected reflect.Map type here, got %v", base.Kind())
	}

	if base.IsNil() && overlay.IsNil() {
		return reflect.MakeMap(base.Type()), nil
	}

	result := reflect.MakeMap(base.Type())

	if !base.IsNil() {
		for _, key := range base.MapKeys() {
			elem := base.MapIndex(key)
			//elem = mergeTaggedReflect(elem, elem)
			elem, err := deepClone(elem)
			if err != nil {
				return reflect.Value{}, err
			}
			result.SetMapIndex(key, elem)
		}
	}

	if !overlay.IsNil() {
		for _, key := range overlay.MapKeys() {
			elem := overlay.MapIndex(key)
			//elem = mergeTaggedReflect(elem, elem)
			elem, err := deepClone(elem)
			if err != nil {
				return reflect.Value{}, err
			}
			result.SetMapIndex(key, elem)
		}
	}

	return result, nil
}

func mergeAtomicField(baseField reflect.Value, overlayField reflect.Value, field reflect.StructField) (reflect.Value, error) {
	// Overlay wins if non-nil (pointer) or non-zero.
	switch overlayField.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		if !overlayField.IsNil() {
			return deepClone(overlayField)
		}
		return deepClone(baseField)
	//case reflect.Struct:
	//	// TODO: rethink this
	//	if !overlayField.IsZero() {
	//		return deepClone(overlayField)
	//	}
	//
	//	return deepClone(baseField)
	default:
		// TODO: Should we support value types?
		return reflect.Value{}, errors.Errorf("atomic_object: unsupported field type: %v", overlayField.Kind())
		//valToSet := func() reflect.Value {
		//	if !overlayField.IsZero() {
		//		return deepClone(overlayField)
		//	}
		//	return deepClone(baseField)
		//}()
		//out.Field(i).Set(valToSet)
	}
}
