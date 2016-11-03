package radix_test

import (
	"fmt"
	"math/rand"
	"testing"

	. "github.com/gobwas/radix"
)

func TestPathFirst(t *testing.T) {
	for i, test := range []struct {
		data  []Pair
		first Pair
		last  Pair
	}{
		{
			data: []Pair{
				{3, "d"},
				{1, "b"},
				{2, "c"},
				{0, "a"},
			},
			first: Pair{0, "a"},
			last:  Pair{3, "d"},
		},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			p := PathFromSliceBorrow(test.data)

			f, _, ok := p.First()
			if !ok {
				t.Errorf("expected First() to return true, but got false")
			}
			if f != test.first {
				t.Errorf("First() = %v; want %v", f, test.first)
			}

			l, _, ok := p.Last()
			if !ok {
				t.Errorf("expected Last() to return true, but got false")
			}
			if l != test.last {
				t.Errorf("Last() = %v; want %v", l, test.first)
			}
		})
	}
}

func benchmarkPathFromMap(b *testing.B, size int) {
	m := make(map[uint]string, size)
	s := randStr(size)
	for i := 0; i < size; i++ {
		m[uint(i)] = s[i]
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PathFromMap(m)
	}
}

func benchmarkPathFromSlice(b *testing.B, size int) {
	data := make([]Pair, size)
	s := randStr(size)
	for i, key := range rand.Perm(size) {
		data[i] = Pair{uint(key), s[i]}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PathFromSlice(data)
	}
}

func benchmarkPathWithout(b *testing.B, size int) {
	data := make([]Pair, size)
	s := randStr(size)
	rid := rand.Perm(size)
	for i, key := range rid {
		data[i] = Pair{uint(key), s[i]}
	}
	path := PathFromSlice(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = path.Without(uint(rid[i%size]))
	}
}

func BenchmarkPathWithout_2(b *testing.B)  { benchmarkPathWithout(b, 2) }
func BenchmarkPathWithout_4(b *testing.B)  { benchmarkPathWithout(b, 4) }
func BenchmarkPathWithout_8(b *testing.B)  { benchmarkPathWithout(b, 8) }
func BenchmarkPathWithout_16(b *testing.B) { benchmarkPathWithout(b, 16) }
func BenchmarkPathWithout_32(b *testing.B) { benchmarkPathWithout(b, 32) }

func BenchmarkPathFromMap_2(b *testing.B)  { benchmarkPathFromMap(b, 2) }
func BenchmarkPathFromMap_4(b *testing.B)  { benchmarkPathFromMap(b, 4) }
func BenchmarkPathFromMap_8(b *testing.B)  { benchmarkPathFromMap(b, 8) }
func BenchmarkPathFromMap_16(b *testing.B) { benchmarkPathFromMap(b, 16) }
func BenchmarkPathFromMap_32(b *testing.B) { benchmarkPathFromMap(b, 32) }

func BenchmarkPathFromSlice_2(b *testing.B)  { benchmarkPathFromSlice(b, 2) }
func BenchmarkPathFromSlice_4(b *testing.B)  { benchmarkPathFromSlice(b, 4) }
func BenchmarkPathFromSlice_8(b *testing.B)  { benchmarkPathFromSlice(b, 8) }
func BenchmarkPathFromSlice_16(b *testing.B) { benchmarkPathFromSlice(b, 16) }
func BenchmarkPathFromSlice_32(b *testing.B) { benchmarkPathFromSlice(b, 32) }
