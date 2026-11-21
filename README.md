# gofucs

Go Functional Collections: functional slice helpers enabled by Go 1.27
generic methods.

```go
items := fucs.FromSlice([]int{1, 2, 3, 4})

evens := items.Filter(func(x int) bool { return x%2 == 0 })

sum := fucs.FromSlice(evens).
	Map(func(x int) int { return x * 2 }).
	Reduce(func(acc, x int) int { return acc + x })
```

## Performance and aliasing contract

- `FromSlice` stores the given slice without copying its elements. The
  collection is a 24-byte header; it stays on the stack when calls are chained
  and inlined, and costs at most 24 bytes if it escapes — never a copy of the
  data.
- `Collect` returns the backing slice as-is. It aliases the input and does not
  copy.
- `Map` and `Filter` allocate the result slice; `Filter` may retain spare
  capacity beyond the returned length, up to the input length.
- The `...Into` variants (`MapInto`, `FilterInto`, `MapConcurrentInto`) write
  into a caller-supplied destination when it has sufficient capacity,
  performing zero allocations; otherwise they allocate an exact-size slice.
- `FilterInto` supports in-place compaction, so
  `c.FilterInto(c.Collect()[:0], pred)` filters the collection's own backing
  slice without copying.
- `MapInto`'s destination must not alias the source unless the mapper is pure
  with respect to the source elements.

## MapConcurrent

`MapConcurrent` partitions the input into contiguous chunks and maps them in
parallel. Results are identical to `Map` and remain ordered. Use it when the
mapper is CPU-heavy enough to outweigh coordination overhead; for trivial
mappers sequential `Map` is usually faster.

## Benchmarks

```sh
go test ./fucs -run '^$' -bench . -benchmem
```
