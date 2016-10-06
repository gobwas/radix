package radix

import (
	"math/rand"
	"testing"
)

func benchmarkPathFromMap(b *testing.B, size int) {
	m := make(map[uint]string, size)
	s := randStr(size, 16)
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
	s := randStr(size, 16)
	for i, key := range rand.Perm(size) {
		data[i] = Pair{uint(key), s[i]}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PathFromSlice(data...)
	}
}

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