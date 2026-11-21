// Package fucs provides functional-style operations over slices.
//
// Collections never modify their input slice. FromSlice and Collect do not copy
// elements: a collection stores only the 24-byte slice header, which the
// compiler usually keeps on the stack when calls are chained. Map and Filter
// allocate their result; the ...Into variants write into a caller-provided
// buffer and reuse its capacity, so steady-state pipelines can run with zero
// allocations. Filter and FilterInto may return a result whose capacity exceeds
// its length, bounded by the input length.
package fucs

import (
	"runtime"
	"sync"
)

type sliceCollection[T any] struct {
	items []T
}

// Mapper transforms a single element from type T to type O.
type Mapper[T, O any] func(item T) O

// Reducer accumulates a running result of type O over a sequence of T values.
type Reducer[T, O any] func(acc O, curr T) O

// Predicate tests whether an element of type T should be included.
type Predicate[T any] func(curr T) bool

// FromSlice wraps items without copying them.
func FromSlice[T any](items []T) *sliceCollection[T] {
	return &sliceCollection[T]{items}
}

// Map transforms every element with f, returning a new collection.
//
// The loop is written out instead of delegating to MapInto: generic methods are
// shape-stenciled, so a non-inlined Map running with dictionary types spills f,
// src and dst on every element (~2x slower). A small direct body also lets the
// compiler inline Mapper calls for large element types.
func (c *sliceCollection[T]) Map[O any](f Mapper[T, O]) *sliceCollection[O] {
	newItems := make([]O, len(c.items))

	// Index-addressed loop avoids copying each element into a loop variable
	// before passing it to f, which is faster for large element types.
	for i := 0; i < len(c.items); i++ {
		newItems[i] = f(c.items[i])
	}

	return FromSlice(newItems)
}

// MapInto writes the mapped elements into dst and returns the result.
//
// When cap(dst) is at least the number of elements, dst is resliced and written
// in place, so no allocation happens. Otherwise a new slice is allocated.
// Unless f is pure with respect to the indices it has already written, dst must
// not alias the collection's backing array.
func (c *sliceCollection[T]) MapInto[O any](dst []O, f Mapper[T, O]) []O {
	src := c.items
	n := len(src)
	if cap(dst) >= n {
		dst = dst[:n]
	} else {
		dst = make([]O, n)
	}

	for i := range src {
		dst[i] = f(src[i])
	}

	return dst
}

// MapConcurrent applies f to every element concurrently, preserving order.
func (c *sliceCollection[T]) MapConcurrent[O any](f Mapper[T, O]) *sliceCollection[O] {
	return FromSlice(c.MapConcurrentInto(make([]O, len(c.items)), f))
}

// MapConcurrentInto is MapInto with the per-element work spread across workers.
// Output order matches Map. It allocates only when dst lacks capacity.
func (c *sliceCollection[T]) MapConcurrentInto[O any](dst []O, f Mapper[T, O]) []O {
	src := c.items
	n := len(src)
	if cap(dst) >= n {
		dst = dst[:n]
	} else {
		dst = make([]O, n)
	}

	if n == 0 {
		return dst
	}

	// One goroutine per core at most, each owning a disjoint contiguous output
	// range: results stay ordered and no synchronisation is needed per element.
	// Splitting by index (not by fixed chunk size) gives every worker the same
	// amount of work to within one element.
	workers := runtime.GOMAXPROCS(0)
	if workers > n {
		workers = n
	}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		lo := w * n / workers
		hi := (w + 1) * n / workers
		s, d := src[lo:hi], dst[lo:hi]
		wg.Go(func() {
			// Resliced src/dst let the compiler prove the loop is in bounds.
			for i := range s {
				d[i] = f(s[i])
			}
		})
	}
	wg.Wait()

	return dst
}

// Reduce reduces the collection using the provided reduction function
// since this function doesn't use a start value, a temporary element
// with the default value for the type O will be used for a start value!
func (c *sliceCollection[T]) Reduce[O any](f Reducer[T, O]) O {
	var reduced O
	return c.ReduceWith(reduced, f)
}

// ReduceWith reduces with a startValue.
func (c *sliceCollection[T]) ReduceWith[O any](startValue O, f Reducer[T, O]) O {
	reduced := startValue

	// Direct indexing instead of slices.Values avoids a function call per element.
	for i := range c.items {
		reduced = f(reduced, c.items[i])
	}

	return reduced
}

// Filter returns the elements for which f returns true.
//
// Direct loop for the same inlining reason as Map. Pre-sizing to the input
// length trades at most one input-sized buffer for repeated slice growth; the
// cost is a buffer up to the input size even when nothing matches, bounded by
// the input you already hold.
func (c *sliceCollection[T]) Filter(f Predicate[T]) []T {
	filtered := make([]T, 0, len(c.items))

	for i := range c.items {
		if f(c.items[i]) {
			filtered = append(filtered, c.items[i])
		}
	}

	return filtered
}

// FilterInto writes the kept elements into dst and returns the result. When
// cap(dst) is sufficient it compacts in place, so FilterInto(c.Collect()[:0], f)
// filters without copying; a separate dst leaves the input untouched. dst must
// not be an offset subslice of the source: only dst[:0] (same start) keeps the
// unread elements ahead of the write cursor.
func (c *sliceCollection[T]) FilterInto(dst []T, f Predicate[T]) []T {
	src := c.items
	n := len(src)
	if cap(dst) >= n {
		dst = dst[:n]
	} else {
		dst = make([]T, n)
	}

	kept := 0
	// kept <= i keeps in-place compaction safe: a write at dst[kept] never
	// clobbers a src[i] that has not been read yet.
	for i := range src {
		if f(src[i]) {
			dst[kept] = src[i]
			kept++
		}
	}

	return dst[:kept]
}

// Collect returns the collection's backing slice without copying it.
func (c *sliceCollection[T]) Collect() []T {
	return c.items
}
