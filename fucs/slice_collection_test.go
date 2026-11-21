package fucs_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/andreaswachs/gofucs/fucs"
	"github.com/stretchr/testify/assert"
)

func TestSliceCollection(t *testing.T) {
	t.Run("can_map_empty_collection", func(t *testing.T) {
		values := []int{}

		actual := fucs.FromSlice(values).
			Map(func(item int) string {
				return "mapped"
			}).
			Collect()

		expected := []string{}

		assert.Equal(t, expected, actual)
	})

	t.Run("can_map_collection", func(t *testing.T) {
		values := []int{1, 2, 3}

		actual := fucs.FromSlice(values).
			Map(func(item int) string {
				return "mapped"
			}).
			Collect()

		expected := []string{"mapped", "mapped", "mapped"}

		assert.Equal(t, expected, actual)
	})

	t.Run("can_reduce_empty_collection_with_start_value", func(t *testing.T) {
		values := []int{}

		actual := fucs.FromSlice(values).ReduceWith(1000, func(acc int, curr int) int {
			return 67
		})

		assert.Equal(t, 1000, actual)
	})

	t.Run("can_reduce_singleton_collection_with_start_value", func(t *testing.T) {
		expected := 67
		values := []int{expected}

		actual := fucs.FromSlice(values).
			ReduceWith(0, func(acc int, i int) int {
				return acc + i
			})

		assert.Equal(t, expected, actual)
	})

	t.Run("can_reduce_collection_with_start_value", func(t *testing.T) {
		values := []int{1, 2, 3}

		actual := fucs.FromSlice(values).
			ReduceWith("", func(acc string, i int) string {
				return acc + strings.Repeat("a", i)
			})

		assert.Equal(t, "aaaaaa", actual)
	})

	t.Run("can_filter_empty_collection", func(t *testing.T) {
		values := []int{}

		actual := fucs.FromSlice(values).Filter(func(curr int) bool {
			return true
		})

		assert.Empty(t, actual)
	})

	t.Run("can_filter_singleton_collection", func(t *testing.T) {
		values := []int{1}

		actual := fucs.FromSlice(values).Filter(func(curr int) bool {
			return true
		})

		assert.Equal(t, []int{1}, actual)
	})

	t.Run("can_filter_collection", func(t *testing.T) {
		values := []int{1, 2, 3}

		actual := fucs.FromSlice(values).Filter(func(curr int) bool {
			return curr >= 2
		})

		assert.Equal(t, []int{2, 3}, actual)
	})
}

func TestSliceCollectionEdgeCases(t *testing.T) {
	t.Run("map_singleton_collection", func(t *testing.T) {
		actual := fucs.FromSlice([]int{42}).
			Map(func(item int) int { return item * 2 }).
			Collect()

		assert.Equal(t, []int{84}, actual)
	})

	t.Run("map_preserves_order", func(t *testing.T) {
		actual := fucs.FromSlice([]int{3, 1, 2}).
			Map(func(item int) int { return item }).
			Collect()

		assert.Equal(t, []int{3, 1, 2}, actual)
	})

	t.Run("map_chained_type_change", func(t *testing.T) {
		actual := fucs.FromSlice([]int{1, 2, 3}).
			Map(func(item int) int { return item + 1 }).
			Map(func(item int) string { return strconv.Itoa(item * 10) }).
			Collect()

		assert.Equal(t, []string{"20", "30", "40"}, actual)
	})

	t.Run("map_concurrent_empty_collection", func(t *testing.T) {
		actual := fucs.FromSlice([]int{}).
			MapConcurrent(func(item int) int { return item * 2 }).
			Collect()

		assert.Equal(t, []int{}, actual)
	})

	t.Run("map_concurrent_singleton_collection", func(t *testing.T) {
		actual := fucs.FromSlice([]int{42}).
			MapConcurrent(func(item int) int { return item * 2 }).
			Collect()

		assert.Equal(t, []int{84}, actual)
	})

	t.Run("map_concurrent_type_change", func(t *testing.T) {
		actual := fucs.FromSlice([]int{1, 2, 3}).
			MapConcurrent(func(item int) string { return strconv.Itoa(item * 2) }).
			Collect()

		assert.Equal(t, []string{"2", "4", "6"}, actual)
	})

	t.Run("map_concurrent_preserves_order", func(t *testing.T) {
		values := make([]int, 1000)
		for i := range values {
			values[i] = i
		}

		actual := fucs.FromSlice(values).
			MapConcurrent(func(item int) int { return item * 2 }).
			Collect()

		assert.Equal(t, len(values), len(actual))
		for i := range values {
			assert.Equal(t, values[i]*2, actual[i])
		}
	})

	t.Run("reduce_empty_collection", func(t *testing.T) {
		actual := fucs.FromSlice([]int{}).
			Reduce(func(acc, curr int) int { return acc + curr })

		assert.Equal(t, 0, actual)
	})

	t.Run("reduce_singleton_collection", func(t *testing.T) {
		actual := fucs.FromSlice([]int{42}).
			Reduce(func(acc, curr int) int { return acc + curr })

		assert.Equal(t, 42, actual)
	})

	t.Run("reduce_collection", func(t *testing.T) {
		actual := fucs.FromSlice([]int{1, 2, 3, 4, 5}).
			Reduce(func(acc, curr int) int { return acc + curr })

		assert.Equal(t, 15, actual)
	})

	t.Run("filter_rejects_all", func(t *testing.T) {
		actual := fucs.FromSlice([]int{1, 2, 3}).
			Filter(func(curr int) bool { return false })

		assert.Empty(t, actual)
	})

	t.Run("filter_keeps_all", func(t *testing.T) {
		actual := fucs.FromSlice([]int{1, 2, 3}).
			Filter(func(curr int) bool { return true })

		assert.Equal(t, []int{1, 2, 3}, actual)
	})

	t.Run("filter_singleton_rejected", func(t *testing.T) {
		actual := fucs.FromSlice([]int{1}).
			Filter(func(curr int) bool { return false })

		assert.Empty(t, actual)
	})

	t.Run("filter_preserves_order", func(t *testing.T) {
		actual := fucs.FromSlice([]int{5, 3, 8, 1, 6}).
			Filter(func(curr int) bool { return curr > 4 })

		assert.Equal(t, []int{5, 8, 6}, actual)
	})

	t.Run("collect_returns_underlying_slice", func(t *testing.T) {
		values := []int{1, 2, 3}

		actual := fucs.FromSlice(values).Collect()

		assert.Equal(t, values, actual)
	})

	t.Run("handles_nil_slice", func(t *testing.T) {
		var values []int

		mapped := fucs.FromSlice(values).
			Map(func(item int) int { return item }).
			Collect()
		assert.Empty(t, mapped)

		filtered := fucs.FromSlice(values).
			Filter(func(curr int) bool { return true })
		assert.Empty(t, filtered)

		reduced := fucs.FromSlice(values).
			Reduce(func(acc, curr int) int { return acc + curr })
		assert.Equal(t, 0, reduced)
	})
}

// intoSink keeps Into results reachable so the compiler cannot drop them.
var intoSink []int

func intoAddOne(x int) int  { return x + 1 }
func intoIsEven(x int) bool { return x%2 == 0 }

func TestIntoAndAllocation(t *testing.T) {
	t.Run("map_into_reuses_backing_array", func(t *testing.T) {
		src := []int{1, 2, 3}
		dst := make([]int, 3, 8)

		got := fucs.FromSlice(src).MapInto(dst, intoAddOne)

		assert.Equal(t, 3, len(got))
		assert.Equal(t, []int{2, 3, 4}, got)
		assert.True(t, &dst[0] == &got[0], "MapInto should reuse dst's backing array")
	})

	t.Run("map_into_allocates_when_capacity_is_low", func(t *testing.T) {
		src := []int{1, 2, 3}
		dst := make([]int, 2)

		got := fucs.FromSlice(src).MapInto(dst, intoAddOne)

		assert.Equal(t, []int{2, 3, 4}, got)
		assert.False(t, &dst[0] == &got[0], "MapInto should grow when cap(dst) < len(src)")
	})

	t.Run("filter_into_reuses_backing_array", func(t *testing.T) {
		src := []int{1, 2, 3, 4}
		dst := make([]int, 4, 8)

		got := fucs.FromSlice(src).FilterInto(dst, intoIsEven)

		assert.Equal(t, 2, len(got))
		assert.Equal(t, []int{2, 4}, got)
		assert.True(t, &dst[0] == &got[0], "FilterInto should reuse dst's backing array")
	})

	t.Run("filter_into_compacts_in_place", func(t *testing.T) {
		src := []int{1, 2, 3, 4, 5}

		got := fucs.FromSlice(src).FilterInto(src[:0], intoIsEven)

		assert.Equal(t, 2, len(got))
		assert.Equal(t, []int{2, 4}, got)
		assert.True(t, &src[0] == &got[0], "in-place FilterInto should write into src")
	})

	t.Run("filter_into_leaves_input_intact_with_separate_buffer", func(t *testing.T) {
		src := []int{1, 2, 3, 4}
		dst := make([]int, 4)

		got := fucs.FromSlice(src).FilterInto(dst, intoIsEven)

		assert.Equal(t, []int{2, 4}, got)
		assert.Equal(t, []int{1, 2, 3, 4}, src)
	})

	t.Run("filter_into_allocates_when_capacity_is_low", func(t *testing.T) {
		src := []int{1, 2, 3, 4}
		dst := make([]int, 1)

		got := fucs.FromSlice(src).FilterInto(dst, intoIsEven)

		assert.Equal(t, []int{2, 4}, got)
		assert.False(t, &dst[0] == &got[0], "FilterInto should grow when cap(dst) < len(src)")
	})

	t.Run("map_concurrent_into_matches_map_and_reuses_buffer", func(t *testing.T) {
		src := make([]int, 1000)
		for i := range src {
			src[i] = i
		}
		want := fucs.FromSlice(src).Map(intoAddOne).Collect()
		dst := make([]int, len(src))

		got := fucs.FromSlice(src).MapConcurrentInto(dst, intoAddOne)

		assert.Equal(t, want, got)
		assert.True(t, &dst[0] == &got[0], "MapConcurrentInto should reuse dst's backing array")
	})

	t.Run("map_concurrent_into_empty_collection", func(t *testing.T) {
		dst := make([]int, 0)

		got := fucs.FromSlice([]int{}).MapConcurrentInto(dst, intoAddOne)

		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("map_concurrent_into_singleton_collection", func(t *testing.T) {
		dst := make([]int, 1)

		got := fucs.FromSlice([]int{41}).MapConcurrentInto(dst, intoAddOne)

		assert.Equal(t, []int{42}, got)
		assert.True(t, &dst[0] == &got[0], "MapConcurrentInto should reuse dst's backing array")
	})

	t.Run("from_slice_and_collect_do_not_allocate", func(t *testing.T) {
		src := make([]int, 100)
		for i := range src {
			src[i] = i
		}

		allocs := testing.AllocsPerRun(100, func() {
			intoSink = fucs.FromSlice(src).Collect()
		})

		assert.Equal(t, float64(0), allocs)
	})

	t.Run("map_into_with_sufficient_buffer_does_not_allocate", func(t *testing.T) {
		src := make([]int, 100)
		for i := range src {
			src[i] = i
		}
		dst := make([]int, len(src))

		allocs := testing.AllocsPerRun(100, func() {
			intoSink = fucs.FromSlice(src).MapInto(dst, intoAddOne)
		})

		assert.Equal(t, float64(0), allocs)
	})

	t.Run("filter_into_with_sufficient_buffer_does_not_allocate", func(t *testing.T) {
		src := make([]int, 100)
		for i := range src {
			src[i] = i
		}
		dst := make([]int, len(src))

		allocs := testing.AllocsPerRun(100, func() {
			intoSink = fucs.FromSlice(src).FilterInto(dst, intoIsEven)
		})

		assert.Equal(t, float64(0), allocs)
	})
}
