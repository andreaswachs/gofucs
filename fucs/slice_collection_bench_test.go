package fucs_test

import (
	"testing"

	"github.com/andreaswachs/gofucs/fucs"
)

// sink prevents the compiler from eliminating the work being benchmarked.
var (
	sinkSlice  any
	sinkInt    int
	sinkString string
)

const (
	sizeSmall  = 10
	sizeMedium = 1_000
	sizeLarge  = 100_000
)

func makeInts(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

func makeStrings(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "x"
	}
	return out
}

// intMapFn is a trivial work function so the benchmark measures framework
// overhead, not the work itself.
func intMapFn(x int) int { return x + 1 }

// --- Map ---

func benchMap(b *testing.B, n int) {
	items := makeInts(n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = fucs.FromSlice(items).
			Map(intMapFn).
			Collect()
	}
}

// largeItem is a 64-byte struct used to measure Map's per-element copy cost,
// where index-only ranging (vs. copying into a loop variable) should show a
// clearer benefit than for 8-byte ints.
type largeItem struct {
	a, b, c, d, e, f, g, h int
}

func makeLargeItems(n int) []largeItem {
	out := make([]largeItem, n)
	for i := range out {
		out[i] = largeItem{a: i}
	}
	return out
}

func largeMapFn(x largeItem) largeItem { x.a++; return x }

func benchMapLarge(b *testing.B, n int) {
	items := makeLargeItems(n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 64))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = fucs.FromSlice(items).Map(largeMapFn).Collect()
	}
}

func BenchmarkMapLargeItemMedium(b *testing.B) { benchMapLarge(b, sizeMedium) }
func BenchmarkMapLargeItemLarge(b *testing.B)  { benchMapLarge(b, sizeLarge) }

func BenchmarkMapSmall(b *testing.B)  { benchMap(b, sizeSmall) }
func BenchmarkMapMedium(b *testing.B) { benchMap(b, sizeMedium) }
func BenchmarkMapLarge(b *testing.B)  { benchMap(b, sizeLarge) }

// --- MapConcurrent ---

func benchMapConcurrent(b *testing.B, n int) {
	items := makeInts(n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = fucs.FromSlice(items).
			MapConcurrent(intMapFn).
			Collect()
	}
}

func BenchmarkMapConcurrentSmall(b *testing.B)  { benchMapConcurrent(b, sizeSmall) }
func BenchmarkMapConcurrentMedium(b *testing.B) { benchMapConcurrent(b, sizeMedium) }
func BenchmarkMapConcurrentLarge(b *testing.B)  { benchMapConcurrent(b, sizeLarge) }

// --- ReduceWith ---

func benchReduceWith(b *testing.B, n int) {
	items := makeInts(n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkInt = fucs.FromSlice(items).
			ReduceWith(0, func(acc, curr int) int {
				return acc + curr
			})
	}
}

func BenchmarkReduceWithSmall(b *testing.B)  { benchReduceWith(b, sizeSmall) }
func BenchmarkReduceWithMedium(b *testing.B) { benchReduceWith(b, sizeMedium) }
func BenchmarkReduceWithLarge(b *testing.B)  { benchReduceWith(b, sizeLarge) }

// --- Reduce (no start value) ---

func benchReduce(b *testing.B, n int) {
	items := makeInts(n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkInt = fucs.FromSlice(items).
			Reduce(func(acc, curr int) int {
				return acc + curr
			})
	}
}

func BenchmarkReduceSmall(b *testing.B)  { benchReduce(b, sizeSmall) }
func BenchmarkReduceMedium(b *testing.B) { benchReduce(b, sizeMedium) }
func BenchmarkReduceLarge(b *testing.B)  { benchReduce(b, sizeLarge) }

// --- Filter (about half pass) ---

func benchFilter(b *testing.B, n int) {
	items := makeInts(n)
	pred := func(curr int) bool { return curr%2 == 0 }
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = fucs.FromSlice(items).Filter(pred)
	}
}

func BenchmarkFilterSmall(b *testing.B)  { benchFilter(b, sizeSmall) }
func BenchmarkFilterMedium(b *testing.B) { benchFilter(b, sizeMedium) }
func BenchmarkFilterLarge(b *testing.B)  { benchFilter(b, sizeLarge) }

// --- Filter all pass (worst case for append growth in naive impl) ---

func benchFilterAll(b *testing.B, n int) {
	items := makeInts(n)
	pred := func(curr int) bool { return true }
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = fucs.FromSlice(items).Filter(pred)
	}
}

func BenchmarkFilterAllSmall(b *testing.B)  { benchFilterAll(b, sizeSmall) }
func BenchmarkFilterAllMedium(b *testing.B) { benchFilterAll(b, sizeMedium) }
func BenchmarkFilterAllLarge(b *testing.B)  { benchFilterAll(b, sizeLarge) }

// --- Filter none pass (no appends) ---

func benchFilterNone(b *testing.B, n int) {
	items := makeInts(n)
	pred := func(curr int) bool { return false }
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = fucs.FromSlice(items).Filter(pred)
	}
}

func BenchmarkFilterNoneSmall(b *testing.B)  { benchFilterNone(b, sizeSmall) }
func BenchmarkFilterNoneMedium(b *testing.B) { benchFilterNone(b, sizeMedium) }
func BenchmarkFilterNoneLarge(b *testing.B)  { benchFilterNone(b, sizeLarge) }

// --- Map vs MapConcurrent with heavy per-element work ---
//
// The trivial-work benchmarks above measure framework overhead. These
// benchmarks use a CPU-bound work function so concurrency can actually pay
// off, showing the crossover where MapConcurrent beats sequential Map.

// heavyWork is deterministic CPU work (~incompressible, no allocation).
func heavyWork(x int) int {
	acc := x
	for i := 0; i < 1000; i++ {
		acc = (acc*1103515245 + 12345) & 0x7fffffff
	}
	return acc
}

func benchMapHeavy(b *testing.B, n int) {
	items := makeInts(n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = fucs.FromSlice(items).Map(heavyWork).Collect()
	}
}

func benchMapConcurrentHeavy(b *testing.B, n int) {
	items := makeInts(n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sinkSlice = fucs.FromSlice(items).MapConcurrent(heavyWork).Collect()
	}
}

func BenchmarkMapHeavyMedium(b *testing.B)           { benchMapHeavy(b, sizeMedium) }
func BenchmarkMapHeavyLarge(b *testing.B)            { benchMapHeavy(b, sizeLarge) }
func BenchmarkMapConcurrentHeavyMedium(b *testing.B) { benchMapConcurrentHeavy(b, sizeMedium) }
func BenchmarkMapConcurrentHeavyLarge(b *testing.B)  { benchMapConcurrentHeavy(b, sizeLarge) }

// --- Zero-allocation paths ---
//
// The Into variants write into a caller-supplied destination when it has
// enough capacity, so these benchmarks reuse one destination and should
// report 0 allocs/op (sequential paths).

// intoSinkSlice is a distinct typed sink so the Into results cannot be
// optimized away, without paying for interface boxing in the measured loop.
var intoSinkSlice []int

func benchFromSliceCollect(b *testing.B, n int) {
	items := makeInts(n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for range b.N {
		intoSinkSlice = fucs.FromSlice(items).Collect()
	}
}

func BenchmarkFromSliceCollect(b *testing.B) { benchFromSliceCollect(b, sizeMedium) }

func benchMapIntoReuse(b *testing.B, n int) {
	items := makeInts(n)
	dst := make([]int, n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for range b.N {
		intoSinkSlice = fucs.FromSlice(items).MapInto(dst, intMapFn)
	}
}

func BenchmarkMapIntoReuseMedium(b *testing.B) { benchMapIntoReuse(b, sizeMedium) }
func BenchmarkMapIntoReuseLarge(b *testing.B)  { benchMapIntoReuse(b, sizeLarge) }

func benchFilterIntoReuse(b *testing.B, n int) {
	items := makeInts(n)
	pred := func(curr int) bool { return curr%2 == 0 }
	dst := make([]int, n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for range b.N {
		intoSinkSlice = fucs.FromSlice(items).FilterInto(dst, pred)
	}
}

func BenchmarkFilterIntoReuseMedium(b *testing.B) { benchFilterIntoReuse(b, sizeMedium) }
func BenchmarkFilterIntoReuseLarge(b *testing.B)  { benchFilterIntoReuse(b, sizeLarge) }

// benchFilterInPlace measures the zero-copy in-place filter pattern. The
// separate buffer is reset with buf[:0] each iteration so compaction reuses
// the same backing array; the input items are never mutated.
func benchFilterInPlace(b *testing.B, n int) {
	items := makeInts(n)
	pred := func(curr int) bool { return curr%2 == 0 }
	buf := make([]int, n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for range b.N {
		intoSinkSlice = fucs.FromSlice(items).FilterInto(buf[:0], pred)
	}
}

func BenchmarkFilterInPlaceMedium(b *testing.B) { benchFilterInPlace(b, sizeMedium) }
func BenchmarkFilterInPlaceLarge(b *testing.B)  { benchFilterInPlace(b, sizeLarge) }

// benchMapConcurrentIntoReuse reuses one destination. Per-worker closure
// allocation may remain, so compare its allocs/op to BenchmarkMapConcurrentLarge
// rather than expecting exactly zero.
func benchMapConcurrentIntoReuse(b *testing.B, n int) {
	items := makeInts(n)
	dst := make([]int, n)
	b.ReportAllocs()
	b.SetBytes(int64(n * 8))
	b.ResetTimer()
	for range b.N {
		intoSinkSlice = fucs.FromSlice(items).MapConcurrentInto(dst, intMapFn)
	}
}

func BenchmarkMapConcurrentIntoReuse(b *testing.B) { benchMapConcurrentIntoReuse(b, sizeLarge) }
