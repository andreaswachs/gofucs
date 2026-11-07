package fucs_test

import (
	"crypto/sha256"
	"fmt"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andreaswachs/gofucs/fucs"
)

// This file answers one question: does MapConcurrent turn the data parallelism
// it advertises into real parallel execution, or does it merely compile?
//
// The benchmarks report speedup relative to a sequential Map pass with exactly
// the same work, for 1, 2, 4, ... GOMAXPROCS processors, so the scaling curve is
// visible in one run. TestMapConcurrentUsesAllCores asserts the two observable
// properties of a genuinely data-parallel run: all workers executing at the same
// time, and wall time dropping to what the hardware allows.

// busyRounds is the amount of CPU work per element. It is large enough that
// per-element scheduling and chunk bookkeeping are noise, so the measured
// numbers reflect the mapper work and not the framework.
const busyRounds = 1_000

// resultSink keeps mapped results reachable so the compiler cannot drop the work
// being measured.
var resultSink any

// busyWork is deterministic, allocation-free CPU work of the sort a real mapper
// would do: a dependent integer pipeline with no branches.
func busyWork(x, rounds int) int {
	acc := x | 1
	for range rounds {
		acc = (acc*1103515245 + 12345) & 0x7fffffff
	}
	return acc
}

// concurrencyProbe tracks how many mapper invocations are running right now and
// the highest count ever seen. Peak concurrency near GOMAXPROCS is the direct
// signal that the cores are lit up.
type concurrencyProbe struct {
	inFlight atomic.Int64
	peak     atomic.Int64
}

func (p *concurrencyProbe) enter() {
	n := p.inFlight.Add(1)
	for {
		peak := p.peak.Load()
		if n <= peak || p.peak.CompareAndSwap(peak, n) {
			return
		}
	}
}

func (p *concurrencyProbe) leave() { p.inFlight.Add(-1) }

func (p *concurrencyProbe) reset() {
	p.inFlight.Store(0)
	p.peak.Store(0)
}

// coreCounts returns 1, 2, 4, ... up to this machine's GOMAXPROCS, so a scaling
// run shows the full curve instead of a single point.
func coreCounts() []int {
	limit := runtime.GOMAXPROCS(0)
	counts := []int{1}
	for c := 2; c < limit; c *= 2 {
		counts = append(counts, c)
	}
	if limit > 1 && counts[len(counts)-1] != limit {
		counts = append(counts, limit)
	}
	return counts
}

// bestOf returns the fastest of runs executions of fn. Sequential baselines are
// measured this way because scheduler and GC noise only ever makes a run
// slower; picking the fastest run gives the parallel variants a fair comparison.
func bestOf(runs int, fn func()) time.Duration {
	best := time.Duration(1<<63 - 1)
	for range runs {
		start := time.Now()
		fn()
		best = min(best, time.Since(start))
	}
	return best
}

// benchScaling runs MapConcurrent capped at cores processors and reports
// throughput plus speedup over the supplied sequential baseline.
func benchScaling[T, O any](b *testing.B, cores int, items []T, f fucs.Mapper[T, O], seq time.Duration) {
	prev := runtime.GOMAXPROCS(cores)
	defer runtime.GOMAXPROCS(prev)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		resultSink = fucs.FromSlice(items).MapConcurrent(f).Collect()
	}
	b.StopTimer()

	elapsed := b.Elapsed()
	perOp := max(elapsed/time.Duration(b.N), 1)

	b.ReportMetric(float64(seq)/float64(perOp), "x_speedup")
	b.ReportMetric(float64(b.N*len(items))/elapsed.Seconds(), "items/s")
}

// BenchmarkMapConcurrentScaling measures how far MapConcurrent scales pure CPU
// work across cores. x_speedup compares against a single sequential Map pass
// over the same input; it rises with the core count until the hardware limit
// (physical cores, not SMT siblings) and then flattens.
func BenchmarkMapConcurrentScaling(b *testing.B) {
	items := makeInts(sizeLarge)
	work := func(x int) int { return busyWork(x, busyRounds) }
	seq := bestOf(3, func() { resultSink = fucs.FromSlice(items).Map(work).Collect() })

	for _, cores := range coreCounts() {
		b.Run(fmt.Sprintf("cores=%d", cores), func(b *testing.B) {
			benchScaling(b, cores, items, work, seq)
		})
	}
}

// BenchmarkMapConcurrentScalingZeroAlloc is the same measurement with a mapper
// that allocates nothing, so the speedup cannot be attributed to concurrent GC
// work.
func BenchmarkMapConcurrentScalingZeroAlloc(b *testing.B) {
	items := makeInts(sizeLarge)
	work := func(x int) int { return busyWork(x, 200) }
	seq := bestOf(3, func() { resultSink = fucs.FromSlice(items).Map(work).Collect() })

	for _, cores := range coreCounts() {
		b.Run(fmt.Sprintf("cores=%d", cores), func(b *testing.B) {
			benchScaling(b, cores, items, work, seq)
		})
	}
}

// makePayloads returns n request-sized byte slices, the sort of input a hashing
// pipeline maps over.
func makePayloads(n, size int) [][]byte {
	out := make([][]byte, n)
	for i := range out {
		payload := make([]byte, size)
		for j := range payload {
			payload[j] = byte(i + j)
		}
		out[i] = payload
	}
	return out
}

// BenchmarkMapConcurrentHashScaling is the realistic counterpart to the
// synthetic scaling benchmarks: a SHA-256 per payload, the classic
// embarrassingly parallel mapper.
func BenchmarkMapConcurrentHashScaling(b *testing.B) {
	const (
		payloadCount = 20_000
		payloadLen   = 512
	)
	items := makePayloads(payloadCount, payloadLen)
	work := func(p []byte) [sha256.Size]byte { return sha256.Sum256(p) }
	seq := bestOf(3, func() { resultSink = fucs.FromSlice(items).Map(work).Collect() })

	for _, cores := range coreCounts() {
		b.Run(fmt.Sprintf("cores=%d", cores), func(b *testing.B) {
			b.SetBytes(payloadCount * payloadLen)
			benchScaling(b, cores, items, work, seq)
		})
	}
}

// TestMapConcurrentUsesAllCores proves MapConcurrent delivers data parallelism
// rather than fake concurrency: identical results to Map, every worker executing
// at the same time, and a wall-time speedup that only parallelism can explain.
//
// Absolute speedup targets are deliberately avoided. GOMAXPROCS counts logical
// processors, so an SMT machine tops out near its physical core count, and CPU
// quotas or background load move the ceiling around. Instead the test measures
// what this machine can do with a hand-rolled parallel map over the same work
// and requires MapConcurrent to stay close to it.
func TestMapConcurrentUsesAllCores(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping core utilisation test in -short mode")
	}

	cores := runtime.GOMAXPROCS(0)
	if cores < 2 {
		t.Skipf("needs at least 2 usable cores, GOMAXPROCS=%d", cores)
	}
	t.Cleanup(func() { runtime.GOMAXPROCS(cores) })

	items := makeInts(sizeLarge)
	work := func(x int) int { return busyWork(x, busyRounds) }

	// Sequential baseline: same input, same per-element work.
	seq := bestOf(3, func() { resultSink = fucs.FromSlice(items).Map(work).Collect() })
	want := fucs.FromSlice(items).Map(work).Collect()

	// Warm up goroutine creation and heap growth so the measured runs are not
	// paying for first-use costs.
	fucs.FromSlice(items).MapConcurrent(work)

	type row struct {
		cores   int
		workers int
		elapsed time.Duration
	}

	var rows []row
	for _, n := range coreCounts() {
		runtime.GOMAXPROCS(n)

		// Warm up first: the runtime creates Ps and Ms lazily when GOMAXPROCS
		// grows, so the first run at a new processor count pays for that.
		parallel := fucs.FromSlice(items).MapConcurrent(work).Collect()

		// Parallelism must not change the answer.
		if !slices.Equal(parallel, want) {
			t.Fatalf("MapConcurrent with GOMAXPROCS=%d differs from Map", n)
		}

		elapsed := bestOf(3, func() { resultSink = fucs.FromSlice(items).MapConcurrent(work).Collect() })
		rows = append(rows, row{cores: n, workers: min(len(items), n), elapsed: elapsed})
	}
	runtime.GOMAXPROCS(cores)

	// Peak concurrency is measured separately: the probe's shared atomic counter
	// contends across threads and would distort the timings above.
	probe := &concurrencyProbe{}
	probedWork := func(x int) int {
		probe.enter()
		defer probe.leave()
		return work(x)
	}
	resultSink = fucs.FromSlice(items).MapConcurrent(probedWork).Collect()
	peak, workers := int(probe.peak.Load()), min(len(items), cores)

	// Hand-rolled reference over the same work: the hardware ceiling for this
	// machine and this workload.
	resultSink = parallelReference(items, work)
	ceiling := bestOf(3, func() { resultSink = parallelReference(items, work) })
	refSpeedup := float64(seq) / float64(ceiling)

	t.Logf("%d items x %d rounds of CPU work, sequential %v", len(items), busyRounds, seq)
	for _, r := range rows {
		t.Logf("  GOMAXPROCS=%-2d %8v  %5.2fx", r.cores, r.elapsed, float64(seq)/float64(r.elapsed))
	}
	t.Logf("  hand-rolled reference (GOMAXPROCS=%d) %8v  %5.2fx", cores, ceiling, refSpeedup)
	t.Logf("  peak concurrency %d of %d workers at GOMAXPROCS=%d", peak, workers, cores)

	full := rows[len(rows)-1]

	if peak < workers {
		t.Errorf("peak concurrency %d while %d workers were launched: the work never ran on all cores simultaneously",
			peak, workers)
	}

	speedup := float64(seq) / float64(full.elapsed)
	// A serialised implementation lands at 1x; anything above needs real overlap.
	if speedup < 2 {
		t.Errorf("speedup %.2fx at GOMAXPROCS=%d, want at least 2.00x", speedup, full.cores)
	}
	// Within 25% of the ceiling measured on the same machine over the same work.
	if speedup < 0.75*refSpeedup {
		t.Errorf("speedup %.2fx at GOMAXPROCS=%d while a hand-rolled parallel loop reaches %.2fx: MapConcurrent is leaving parallelism on the table",
			speedup, full.cores, refSpeedup)
	}
}

// parallelReference is a hand-rolled data-parallel map: one goroutine per
// processor over strided indices, no framework in the way. It exists so the test
// compares against this machine's real capability instead of a hardcoded number,
// and so a regression that serialises MapConcurrent (a mutex, one goroutine per
// element, needless copying) shows up as a gap to the reference.
func parallelReference[T, O any](items []T, f fucs.Mapper[T, O]) []O {
	out := make([]O, len(items))
	workers := min(runtime.GOMAXPROCS(0), len(items))
	if workers == 0 {
		return out
	}

	var wg sync.WaitGroup
	for w := range workers {
		wg.Go(func() {
			for i := w; i < len(items); i += workers {
				out[i] = f(items[i])
			}
		})
	}
	wg.Wait()

	return out
}
