package merge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Ptr[T any](v T) *T { return &v }

func TestMergeTagged(t *testing.T) {
	t.Run("simple types", func(t *testing.T) {
		t.Run("non-pointer simple type overlay always wins", func(t *testing.T) {
			// This is even if the overlay is the empty default value
			assert.Equal(t, 0, MustMergeTagged(0, 0))
			assert.Equal(t, 0, MustMergeTagged(20, 0))
			assert.Equal(t, 11, MustMergeTagged(20, 11))
			assert.Equal(t, 11, MustMergeTagged(0, 11))

			assert.Equal(t, "", MustMergeTagged("", ""))
			assert.Equal(t, "", MustMergeTagged("base", ""))
			assert.Equal(t, "overlay", MustMergeTagged("base", "overlay"))
		})

		// Non-nil overlay always wins, even if its dereference value is empty default value.
		t.Run("pointer to simple types overlay always wins if not nil", func(t *testing.T) {
			assert.Equal(t, 0, *MustMergeTagged(Ptr(0), Ptr(0)))
			assert.Equal(t, 0, *MustMergeTagged(Ptr(10), Ptr(0)))
			assert.Equal(t, 0, *MustMergeTagged(nil, Ptr(0)))
		})

		// When overlay is nil, base is used.
		t.Run("pointer to simple types overlay always wins if not nil", func(t *testing.T) {
			assert.Equal(t, 10, *MustMergeTagged(Ptr(10), nil))
		})

		t.Run("merged result is always a new pointer", func(t *testing.T) {
			t.Run("exception: both base and overlay are nil", func(t *testing.T) {
				var base *string = nil
				var overlay *string = nil
				out := MustMergeTagged(base, overlay)
				assert.Nil(t, out)
			})
			t.Run("base not nil, overlay is nil", func(t *testing.T) {
				var base *string = Ptr("base")
				var overlay *string = nil
				out := MustMergeTagged(base, overlay)
				assert.NotSame(t, base, out)
				assert.NotSame(t, overlay, out)
			})
			t.Run("base is nil, overlay not nil", func(t *testing.T) {
				var base *string = nil
				var overlay *string = Ptr("overlay")
				out := MustMergeTagged(base, overlay)
				assert.NotSame(t, base, out)
				assert.NotSame(t, overlay, out)
			})
			t.Run("base not nil, overlay not nil", func(t *testing.T) {
				var base *string = Ptr("overlay")
				var overlay *string = Ptr("overlay")
				out := MustMergeTagged(base, overlay)
				assert.NotSame(t, base, out)
				assert.NotSame(t, overlay, out)
			})
		})

		t.Run("NOT A BUG: merge is correct for pointer-to-pointer **int and ***int values", func(t *testing.T) {
			base := Ptr(Ptr(Ptr(1)))
			var overlay ***int = nil
			expected := Ptr(Ptr(Ptr(1)))
			out := MustMergeTagged(base, overlay)
			assert.Equal(t, ***expected, ***out, "BUG: merge is incorrect pointer-to-pointer **int values")
		})

		t.Run("NOT A BUG: merge is correct for pointer-to-pointer **int and ***int values", func(t *testing.T) {
			var base ***int = nil
			var overlay ***int = Ptr(Ptr(Ptr(10)))
			expected := Ptr(Ptr(Ptr(10)))
			out := MustMergeTagged(base, overlay)
			assert.Equal(t, ***expected, ***out, "BUG: merge is incorrect pointer-to-pointer **int values")
		})
	})

	t.Run("slices", func(t *testing.T) {
		t.Run("simple types", func(t *testing.T) {
			assert.Equal(t, []int{}, MustMergeTagged([]int{}, []int{}))
			assert.Equal(t, []int{1, 2, 3}, MustMergeTagged([]int{1, 2, 3}, []int{}))
			assert.Equal(t, []int{1, 2, 3}, MustMergeTagged([]int{}, []int{1, 2, 3}))
			assert.Equal(t, []int{1, 2, 3, 1, 2, 3}, MustMergeTagged([]int{1, 2, 3}, []int{1, 2, 3}))
		})

		t.Run("pointer types", func(t *testing.T) {
			base := []*int{Ptr(1)}
			overlay := []*int{Ptr(10)}
			expected := []*int{Ptr(1), Ptr(10)}
			out := MustMergeTagged(base, overlay)
			assert.Equal(t, expected, out)

			t.Run("modifying merged result does not affect base or overlay", func(t *testing.T) {
				// Modify merged result and check that base and overlay are not modified.
				// Actually write to the value pointed by the pointer
				for i := 0; i < len(out); i++ {
					*out[i] = 999
				}

				expectedBase := []*int{Ptr(1)}
				expectedOverlay := []*int{Ptr(10)}
				expectedMerged := []*int{Ptr(999), Ptr(999)}
				assert.Equal(t, expectedBase, base)
				assert.Equal(t, expectedOverlay, overlay)
				assert.Equal(t, expectedMerged, out)
			})
		})

		t.Run("pointer structs", func(t *testing.T) {
			type MyStruct struct {
				IntPtr    *int
				StringPtr *string
			}

			base := []*MyStruct{
				{
					IntPtr:    Ptr(1),
					StringPtr: Ptr("base.StringPtr"),
				},
			}
			overlay := []*MyStruct{
				{
					IntPtr:    Ptr(10),
					StringPtr: Ptr("overlay.StringPtr"),
				},
			}
			expected := []*MyStruct{
				{
					IntPtr:    Ptr(1),
					StringPtr: Ptr("base.StringPtr"),
				},
				{
					IntPtr:    Ptr(10),
					StringPtr: Ptr("overlay.StringPtr"),
				},
			}

			out := MustMergeTagged(base, overlay)

			t.Run("merged results is as expected", func(t *testing.T) {
				assert.Equal(t, expected, out)
			})

			t.Run("fields in merged result points to different objects", func(t *testing.T) {
				assert.NotSame(t, base[0], out[0])
				assert.NotSame(t, base[0], out[1])
				assert.NotSame(t, overlay[0], out[0])
				assert.NotSame(t, overlay[0], out[1])
			})

			t.Run("modifying merged result does not affect base or overlay", func(t *testing.T) {
				expectedBase := []*MyStruct{
					{
						IntPtr:    Ptr(1),
						StringPtr: Ptr("base.StringPtr"),
					},
				}
				expectedOverlay := []*MyStruct{
					{
						IntPtr:    Ptr(10),
						StringPtr: Ptr("overlay.StringPtr"),
					},
				}
				expectedMerged := []*MyStruct{
					{
						IntPtr:    Ptr(9991),
						StringPtr: Ptr("modified base.StringPtr"),
					},
					{
						IntPtr:    Ptr(99910),
						StringPtr: Ptr("modified overlay.StringPtr"),
					},
				}

				// Change merged values.
				out[0].IntPtr = Ptr(9991)
				out[0].StringPtr = Ptr("modified base.StringPtr")
				out[1].IntPtr = Ptr(99910)
				out[1].StringPtr = Ptr("modified overlay.StringPtr")

				assert.Equal(t, expectedBase, base, "base should not be modified")
				assert.Equal(t, expectedOverlay, overlay, "overlay should not be modified")
				assert.Equal(t, expectedMerged, out, "modified merged result should match expected result")
			})
		})

		t.Run("BUG FIXED: slices within slices should merge correctly", func(t *testing.T) {
			base := [][]int{{1, 2, 3}}
			overlay := [][]int{{10, 20, 30}}
			expected := [][]int{{1, 2, 3}, {10, 20, 30}}

			// BUG: each element of base and overlay gets merged with each other
			bugActual := [][]int{{1, 2, 3, 1, 2, 3}, {10, 20, 30, 10, 20, 30}}

			out := MustMergeTagged(base, overlay)
			assert.NotEqual(t, bugActual, out)
			assert.Equal(t, expected, out)
		})

		t.Run("BUG: merge is incorrect with nil pointer to base slice", func(t *testing.T) {
			var base *[]int = nil
			var overlay *[]int = &[]int{10, 20, 30}

			var bugExpected *[]int = &[]int{10, 20, 30, 10, 20, 30}
			var expected *[]int = &[]int{10, 20, 30}

			out := MustMergeTagged(base, overlay)
			assert.NotEqual(t, bugExpected, out, "BUG: merge is incorrect with nil pointer to base slice")
			assert.Equal(t, expected, out, "BUG: merge is incorrect with nil pointer to base slice")
		})
	})

	t.Run("maps", func(t *testing.T) {
		t.Run("simple types", func(t *testing.T) {
			assert.Equal(t, map[string]int{}, MustMergeTagged(map[string]int{}, map[string]int{}))
			assert.Equal(t, map[string]int{"a": 1, "b": 2, "c": 3}, MustMergeTagged(map[string]int{"a": 1, "b": 2, "c": 3}, map[string]int{}))
			assert.Equal(t, map[string]int{"a": 10, "b": 20, "c": 30}, MustMergeTagged(map[string]int{}, map[string]int{"a": 10, "b": 20, "c": 30}))
			assert.Equal(t, map[string]int{"a": 10, "b": 20, "c": 30}, MustMergeTagged(map[string]int{"a": 1, "b": 2, "c": 3}, map[string]int{"a": 10, "b": 20, "c": 30}))
			assert.Equal(t, map[string]int{"a": 10, "b": 20, "x": 3, "y": 30}, MustMergeTagged(map[string]int{"a": 1, "b": 2, "x": 3}, map[string]int{"a": 10, "b": 20, "y": 30}))

		})

		t.Run("nil overlay map, merged result is just base", func(t *testing.T) {
			base := map[string]int{"a": 1, "b": 2, "c": 3}
			var overlay map[string]int
			out := MustMergeTagged(base, overlay)
			assert.Equal(t, base, out)
		})

		t.Run("nil base map, merged resutls is just overlay", func(t *testing.T) {
			var base map[string]int
			overlay := map[string]int{"a": 1, "b": 2, "c": 3}
			out := MustMergeTagged(base, overlay)
			assert.Equal(t, overlay, out)
		})

		t.Run("maps with simple pointer types", func(t *testing.T) {
			base := map[string]*int{"a": Ptr(1), "b": Ptr(2), "c": Ptr(3)}
			overlay := map[string]*int{"a": Ptr(10), "b": Ptr(20), "x": Ptr(90)}
			expected := map[string]*int{"a": Ptr(10), "b": Ptr(20), "c": Ptr(3), "x": Ptr(90)}
			out := MustMergeTagged(base, overlay)

			t.Run("merged results is as expected", func(t *testing.T) {
				assert.Equal(t, expected, out)
			})

			t.Run("fields in merged result points to different objects", func(t *testing.T) {
				for k, v := range out {
					assert.NotSame(t, base[k], v)
				}
			})

			t.Run("modifying merged result does not affect base", func(t *testing.T) {
				// Common key
				*out["a"] = 999
				assert.Equal(t, base["a"], Ptr(1), "base should not be modified")
				assert.Equal(t, overlay["a"], Ptr(10), "overlay should not be modified")
				assert.Equal(t, out["a"], Ptr(999), "modified merged result should match expected result")

				// Base-only key
				*out["c"] = 9999
				assert.Equal(t, base["c"], Ptr(3), "base should not be modified")
				assert.Nil(t, overlay["c"], nil, "overlay should not be modified")
				assert.Equal(t, out["c"], Ptr(9999), "modified merged result should match expected result")

				// Overlay-only key
				*out["x"] = 99999
				assert.Nil(t, base["x"], nil, "base should not be modified")
				assert.Equal(t, overlay["x"], Ptr(90), "overlay should not be modified")
				assert.Equal(t, out["x"], Ptr(99999), "modified merged result should match expected result")
			})
		})

		t.Run("maps with pointer structs in values", func(t *testing.T) {
			type MyStruct struct {
				IntPtr    *int
				StringPtr *string
			}

			base := map[string]*MyStruct{
				"a": {
					IntPtr:    Ptr(1),
					StringPtr: Ptr("base.StringPtr"),
				},
			}
			overlay := map[string]*MyStruct{
				"b": {
					IntPtr:    Ptr(10),
					StringPtr: Ptr("overlay.StringPtr"),
				},
			}
			expected := map[string]*MyStruct{
				"a": {
					IntPtr:    Ptr(1),
					StringPtr: Ptr("base.StringPtr"),
				},
				"b": {
					IntPtr:    Ptr(10),
					StringPtr: Ptr("overlay.StringPtr"),
				},
			}

			out := MustMergeTagged(base, overlay)

			t.Run("merged results is as expected", func(t *testing.T) {
				assert.Equal(t, expected, out)
			})

			t.Run("fields in merged result points to different objects", func(t *testing.T) {
				for k, v := range out {
					assert.NotSame(t, base[k], v, "values in merged result should not point to same objects as base")
				}

				for k, v := range out {
					assert.NotSame(t, overlay[k], v, "values in merged result should not point to same objects as overlay")
				}
			})
		})
	})

	t.Run("arrays", func(t *testing.T) {
		t.Run("simple types", func(t *testing.T) {
			assert.Equal(t, [0]int{}, MustMergeTagged([0]int{}, [0]int{}))
			assert.Equal(t, [3]int{10, 20, 30}, MustMergeTagged([3]int{1, 2, 3}, [3]int{10, 20, 30}))
		})

		t.Run("pointer types", func(t *testing.T) {
			// Values within array follow the same rules as for slices.
			base := [3]*int{Ptr(1), Ptr(2), Ptr(3)}
			overlay := [3]*int{Ptr(10), Ptr(20), Ptr(30)}
			expected := [3]*int{Ptr(10), Ptr(20), Ptr(30)}
			assert.Equal(t, expected, MustMergeTagged(base, overlay))
		})

		t.Run("pointer types 2", func(t *testing.T) {
			// Values within array follow the same rules as for slices.
			base := [3]*int{Ptr(1), Ptr(2), Ptr(3)}
			overlay := [3]*int{Ptr(10), nil, nil}
			out := MustMergeTagged(base, overlay)
			assert.Equal(t, 10, *out[0])
			assert.Equal(t, 2, *out[1])
			assert.Equal(t, 3, *out[2])
		})
	})

	t.Run("structs", func(t *testing.T) {
		t.Run("value types", func(t *testing.T) {
			type MyStruct struct {
				A int
				B string
			}

			assert.Equal(t, MyStruct{}, MustMergeTagged(MyStruct{}, MyStruct{}))
			assert.Equal(t, MyStruct{A: 10, B: "overlay"}, MustMergeTagged(MyStruct{A: 1, B: "base"}, MyStruct{A: 10, B: "overlay"}))
			assert.Equal(t, MyStruct{A: 10, B: ""}, MustMergeTagged(MyStruct{A: 1, B: "b"}, MyStruct{A: 10}))
		})

		t.Run("pointer types", func(t *testing.T) {
			type MyStruct struct {
				A *int
				B *string
			}

			assert.Equal(t, MyStruct{}, MustMergeTagged(MyStruct{}, MyStruct{}))
			assert.Equal(t, MyStruct{A: Ptr(10), B: Ptr("overlay")}, MustMergeTagged(MyStruct{A: Ptr(1), B: Ptr("base")}, MyStruct{A: Ptr(10), B: Ptr("overlay")}))
			assert.Equal(t, MyStruct{A: Ptr(10), B: Ptr("base")}, MustMergeTagged(MyStruct{A: Ptr(1), B: Ptr("base")}, MyStruct{A: Ptr(10), B: nil}))
		})

		t.Run("pointers to struct with pointer types", func(t *testing.T) {
			type MyStruct struct {
				A *int
				B *string
			}

			assert.Equal(t, MyStruct{}, *MustMergeTagged(&MyStruct{}, &MyStruct{}))
			assert.Equal(t, MyStruct{A: Ptr(10), B: Ptr("overlay")}, *MustMergeTagged(&MyStruct{A: Ptr(1), B: Ptr("base")}, &MyStruct{A: Ptr(10), B: Ptr("overlay")}))
			assert.Equal(t, MyStruct{A: Ptr(10), B: Ptr("base")}, *MustMergeTagged(&MyStruct{A: Ptr(1), B: Ptr("base")}, &MyStruct{A: Ptr(10), B: nil}))
		})
	})

	t.Run("interface", func(t *testing.T) {
		type MyStruct struct {
			A *int
			B *string
		}

		t.Run("value interface", func(t *testing.T) {
			var base interface{} = MyStruct{A: Ptr(1), B: Ptr("base")}
			var overlay interface{} = MyStruct{A: Ptr(10), B: nil}
			out := MustMergeTagged(base, overlay)
			assert.Equal(t, MyStruct{A: Ptr(10), B: Ptr("base")}, out.(MyStruct))
		})

		t.Run("pointer interface", func(t *testing.T) {
			var base interface{} = &MyStruct{A: Ptr(1), B: Ptr("base")}
			var overlay interface{} = &MyStruct{A: Ptr(10), B: nil}
			out := MustMergeTagged(base, overlay)
			assert.Equal(t, MyStruct{A: Ptr(10), B: Ptr("base")}, *(out.(*MyStruct)))
		})

	})

	t.Run("advanced - ensure deep copy of values in pointer types", func(t *testing.T) {
		// Things pointed to by pointers are copied.
		// This means we can modify merged restult without affecting the base and overlay.
		t.Run("pointer simple type", func(t *testing.T) {
			base := Ptr("base")
			overlay := Ptr("overlay")
			out := MustMergeTagged(base, overlay)
			assert.Equal(t, "overlay", *out)

			// Make sure we cannot overwrite the base pointer by writing
			// to the out pointer.
			*out = "modified"
			assert.Equal(t, "base", *base)
			assert.Equal(t, "overlay", *overlay)
			assert.Equal(t, "modified", *out)
		})

		t.Run("pointer structs in structs", func(t *testing.T) {
			type Inner struct {
				StringPtr   *string
				StringSlice []*string
				StringMap   map[string]*string
			}

			type Outer struct {
				Inner *Inner
			}

			base := &Outer{
				Inner: &Inner{
					StringPtr: Ptr("base.StringPtr"),
					StringSlice: []*string{
						Ptr("base.StringSlice[0]"),
					},
					StringMap: map[string]*string{
						"base": Ptr("base.StringMap.base"),
					},
				},
			}

			overlay := &Outer{
				Inner: &Inner{
					StringPtr: Ptr("overlay.StringPtr"),
					StringSlice: []*string{
						Ptr("overlay.StringSlice[0]"),
					},
					StringMap: map[string]*string{
						"overlay": Ptr("overlay.StringMap.overlay"),
					},
				},
			}

			expected := &Outer{
				Inner: &Inner{
					StringPtr: Ptr("overlay.StringPtr"),
					StringSlice: []*string{
						Ptr("base.StringSlice[0]"),
						Ptr("overlay.StringSlice[0]"),
					},
					StringMap: map[string]*string{
						"base":    Ptr("base.StringMap.base"),
						"overlay": Ptr("overlay.StringMap.overlay"),
					},
				},
			}

			out := MustMergeTagged(base, overlay)

			t.Run("merged results is as expected", func(t *testing.T) {
				assert.Equal(t, *expected.Inner.StringPtr, *out.Inner.StringPtr, "overlay.StringPtr should be merged into base.StringPtr")
				assert.Equal(t, *expected.Inner.StringSlice[0], *out.Inner.StringSlice[0], "overlay.StringPtr should be merged into base.StringPtr")
				assert.Equal(t, *expected.Inner.StringMap["base"], *out.Inner.StringMap["base"], "overlay.StringMap should be merged into base.StringMap")
				assert.Equal(t, *expected.Inner.StringMap["overlay"], *out.Inner.StringMap["overlay"], "overlay.StringMap should be merged into base.StringMap")
			})

			t.Run("base and should not be modified after merge", func(t *testing.T) {
				expectedBase := &Outer{
					Inner: &Inner{
						StringPtr: Ptr("base.StringPtr"),
						StringSlice: []*string{
							Ptr("base.StringSlice[0]"),
						},
						StringMap: map[string]*string{
							"base": Ptr("base.StringMap.base"),
						},
					},
				}

				expectedOverlay := &Outer{
					Inner: &Inner{
						StringPtr: Ptr("overlay.StringPtr"),
						StringSlice: []*string{
							Ptr("overlay.StringSlice[0]"),
						},
						StringMap: map[string]*string{
							"overlay": Ptr("overlay.StringMap.overlay"),
						},
					},
				}

				out.Inner.StringPtr = Ptr("modified.Outer.StringPtr")
				out.Inner.StringSlice[0] = Ptr("modified.Outer.StringSlice[0]")
				out.Inner.StringSlice[1] = Ptr("modified.Outer.StringSlice[0]")
				out.Inner.StringMap["base"] = Ptr("modified.Outer.StringMap.base")
				out.Inner.StringMap["overlay"] = Ptr("modified.Outer.StringMap.overlay")

				assert.Equal(t, expectedBase, base, "base should not be modified")
				assert.Equal(t, expectedOverlay, overlay, "overlay should not be modified")
			})

		})
	})

	t.Run("tag: atomic_object", func(t *testing.T) {
		t.Run("core case", func(t *testing.T) {

			// Need at least two fields in this to test
			type Inner struct {
				StringPtr  *string
				StringPtr2 *string
				StringPtr3 *string
			}

			// When atomic tag is present, the overlay always wins when not nil
			type Outer struct {
				// Simple types and pointers to them are always atomic
				Inner       *Inner
				InnerAtomic *Inner `merge:"atomic_object"`

				// Regular and atomic slices
				// TODO: nil slices are empty slices, how do we treat them?
				Slice       []*string
				SliceAtomic []*string `merge:"atomic_object"`

				//
				Map       map[string]*string
				MapAtomic map[string]*string `merge:"atomic_object"`
			}

			base := &Outer{
				Inner: &Inner{
					// All fields are set in the base
					StringPtr:  Ptr("base.Outer.StringPtr"),
					StringPtr2: Ptr("base.Outer.StringPtr2"),
					StringPtr3: Ptr("base.Outer.StringPtr3"),
				},
				InnerAtomic: &Inner{
					// All fields are set in the base here too
					StringPtr:  Ptr("base.InnerAtomic.StringPtr"),
					StringPtr2: Ptr("base.InnerAtomic.StringPtr2"),
					StringPtr3: Ptr("base.InnerAtomic.StringPtr3"),
				},
				Slice: []*string{
					Ptr("base.Slice[0]"),
				},
				SliceAtomic: []*string{
					Ptr("base.SliceAtomic[0]"),
				},
				Map: map[string]*string{
					"base": Ptr("base.Map.base"),
				},
				MapAtomic: map[string]*string{
					"base": Ptr("base.MapAtomic.base"),
				},
			}

			overlay := &Outer{
				Inner: &Inner{
					StringPtr: Ptr("overlay.Outer.StringPtr"),
				},
				InnerAtomic: &Inner{
					// Only one field set here
					StringPtr: Ptr("overlay.InnerAtomic.StringPtr"),
				},
				Slice: []*string{
					Ptr("overlay.Slice[0]"),
				},
				SliceAtomic: []*string{
					Ptr("overlay.SliceAtomic[0]"),
				},
				Map: map[string]*string{
					"overlay": Ptr("overlay.Map.overlay"),
				},
				MapAtomic: map[string]*string{
					"overlay": Ptr("inter.MapAtomic.overlay"),
				},
			}

			expected := &Outer{
				Inner: &Inner{
					// Default overlay here, base+overlay
					StringPtr:  Ptr("overlay.Outer.StringPtr"),
					StringPtr2: Ptr("base.Outer.StringPtr2"),
					StringPtr3: Ptr("base.Outer.StringPtr3"),
				},
				InnerAtomic: &Inner{
					// Overlay completely won here
					StringPtr: Ptr("overlay.InnerAtomic.StringPtr"),
				},
				Slice: []*string{
					// Standard merge of slices here
					Ptr("base.Slice[0]"),
					Ptr("overlay.Slice[0]"),
				},
				SliceAtomic: []*string{
					// Atomic slice, overlay wins
					Ptr("overlay.SliceAtomic[0]"),
				},
				Map: map[string]*string{
					// Standard map merge here
					"base":    Ptr("base.Map.base"),
					"overlay": Ptr("overlay.Map.overlay"),
				},
				MapAtomic: map[string]*string{
					// Atomic map, overlay wins
					"overlay": Ptr("inter.MapAtomic.overlay"),
				},
			}

			out := MustMergeTagged(base, overlay)

			assert.Equal(t, *expected.Inner, *out.Inner, "Outer should be merged")
			assert.Equal(t, *expected.InnerAtomic, *out.InnerAtomic, "InnerAtomic should be merged")
			assert.Equal(t, expected.Slice, out.Slice, "Slice should be merged")
			assert.Equal(t, expected.SliceAtomic, out.SliceAtomic, "SliceAtomic should be merged")
			assert.Equal(t, expected.Map, out.Map, "Map should be merged")
			assert.Equal(t, expected.MapAtomic, out.MapAtomic, "MapAtomic should be merged")
		})

		t.Run("nil and empty slices, maps and structs in overlay", func(t *testing.T) {
			// There is a difference between nil and empty slices. The []*string{} is empty slice, not nil
			// Same goes for maps.
			type Inner struct {
				StringPtr  *string
				StringPtr2 *string
				StringPtr3 *string
			}

			type Outer struct {
				SliceAtomic     []*string          `merge:"atomic_object"`
				MapAtomic       map[string]*string `merge:"atomic_object"`
				StructAtomicPtr *Inner             `merge:"atomic_object"`

				//// TODO: what do we do with structs which are vals?
				//StructAtomicVal Inner `merge:"atomic_object"`
			}

			base := &Outer{
				SliceAtomic: []*string{
					Ptr("base.SliceAtomic[0]"),
				},
				MapAtomic: map[string]*string{
					"base": Ptr("base.MapAtomic.base"),
				},
				StructAtomicPtr: &Inner{
					StringPtr:  Ptr("base.StructAtomicPtr.StringPtr"),
					StringPtr2: Ptr("base.StructAtomicPtr.StringPtr2"),
					StringPtr3: Ptr("base.StructAtomicPtr.StringPtr3"),
				},
				//StructAtomicVal: Inner{
				//	StringPtr: Ptr("base.StructAtomicVal.StringPtr"),
				//},
			}

			t.Run("NIL: nil in overlay, completely not set: base wins", func(t *testing.T) {
				overlay := &Outer{}
				expected := base
				out := MustMergeTagged(base, overlay)
				assert.Equal(t, expected, out)
			})

			t.Run("NIL: nil overlay slice set explicitly to nil: base wins", func(t *testing.T) {
				overlay := &Outer{
					SliceAtomic: nil,
				}
				expected := base
				out := MustMergeTagged(base, overlay)
				assert.Equal(t, expected, out)
			})

			t.Run("NIL: nil overlay slice init to nil slice var: base wins", func(t *testing.T) {
				var nilSlice []*string
				overlay := &Outer{
					SliceAtomic: nilSlice,
				}
				expected := base
				out := MustMergeTagged(base, overlay)
				assert.Equal(t, expected, out)
			})

			t.Run("EMPTY: overlay slice is EMPTY slice, overlay wins", func(t *testing.T) {
				overlay := &Outer{
					SliceAtomic:     []*string{},
					MapAtomic:       map[string]*string{},
					StructAtomicPtr: &Inner{},
				}
				expected := &Outer{
					// EMPTY slice because overlay is empty, but not nil
					SliceAtomic:     []*string{},
					MapAtomic:       map[string]*string{},
					StructAtomicPtr: &Inner{},
				}
				out := MustMergeTagged(base, overlay)
				assert.Equal(t, expected, out)
			})
		})

		t.Run("NO SUPPORT support for value types", func(t *testing.T) {
			type Inner struct {
				StringPtr  *string
				StringPtr2 *string
				StringPtr3 *string
			}

			type Outer struct {
				InnerAtomicVal Inner `merge:"atomic_object"`
			}

			base := &Outer{
				InnerAtomicVal: Inner{
					StringPtr:  Ptr("base.InnerAtomicVal.StringPtr"),
					StringPtr2: Ptr("base.InnerAtomicVal.StringPtr2"),
					StringPtr3: Ptr("base.InnerAtomicVal.StringPtr3"),
				},
			}

			overlay := &Outer{
				InnerAtomicVal: Inner{
					StringPtr: Ptr("overlay.InnerAtomicVal.StringPtr"),
				},
			}
			require.Panics(t, func() { MustMergeTagged(base, overlay) })
		})
	})
}
