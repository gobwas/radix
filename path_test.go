package radix_test

import (
	"fmt"
	"math/rand"
	"testing"

	. "github.com/gobwas/radix"
)

func TestPathRemove(t *testing.T) {
	for _, test := range []struct {
		data            map[uint]string
		exclude, remove map[uint]bool
	}{
		{
			data: map[uint]string{
				0: "a",
				1: "b",
				2: "c",
			},
			exclude: map[uint]bool{
				1: true,
			},
			remove: map[uint]bool{
				0: true,
			},
		},
	} {
		t.Run("", func(t *testing.T) {
			p := PathFromMapStr(test.data)
			for k := range test.exclude {
				p = p.Without(k)
			}
			for k := range test.remove {
				p.Remove(k)
			}
			for k, exp := range test.data {
				switch {
				case test.remove[k]:
					if p.Has(k) {
						t.Errorf("path has removed key %#x", k)
					}

				case test.exclude[k]:
					if p.Has(k) {
						t.Errorf("path has excluded key %#x", k)
					}

				case !p.Has(k):
					t.Errorf("path does not have included key %#x", k)

				default:
					if act, _ := p.Get(k); string(act) != exp {
						t.Errorf("unexpected %#x key value: %q; want %q", k, act, exp)
					}
				}
			}
		})
	}
}

func TestPathFirst(t *testing.T) {
	for i, test := range []struct {
		data  []PairStr
		first Pair
		last  Pair
	}{
		{
			data: []PairStr{
				{3, "d"},
				{1, "b"},
				{2, "c"},
				{0, "a"},
			},
			first: Pair{0, []byte("a")},
			last:  Pair{3, []byte("d")},
		},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			p := PathFromSliceStr(test.data)

			_, f, ok := p.First()
			if !ok {
				t.Errorf("expected First() to return true, but got false")
			}
			if !f.Equal(test.first) {
				t.Errorf("First() = %v; want %v", f, test.first)
			}

			_, l, ok := p.Last()
			if !ok {
				t.Errorf("expected Last() to return true, but got false")
			}
			if !l.Equal(test.last) {
				t.Errorf("Last() = %v; want %v", l, test.first)
			}
		})
	}
}

var sizes = []int{2, 4, 6, 8, 10, 16, 32}

func BenchmarkPathBuilder(b *testing.B) {
	for _, size := range sizes {
		m := make(map[uint][]byte, size)
		s := randBts(size)
		for i := 0; i < size; i++ {
			m[uint(i)] = s[i]
		}

		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				builder := NewPathBuilder(size)
				for key, value := range m {
					builder.Add(key, value)
				}
				_ = builder.Build()
			}
		})
	}
}

func BenchmarkPathBuilderReuse(b *testing.B) {
	for _, size := range sizes {
		m := make(map[uint][]byte, size)
		s := randBts(size)
		for i := 0; i < size; i++ {
			m[uint(i)] = s[i]
		}

		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				builder := GetPathBuilder(size)
				for key, value := range m {
					builder.Add(key, value)
				}
				_ = builder.Build()
				PutPathBuilder(builder)
			}
		})
	}
}

func BenchmarkPathFromMap(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			m := make(map[uint][]byte, size)
			s := randBts(size)
			for i := 0; i < size; i++ {
				m[uint(i)] = s[i]
			}
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = PathFromMap(m)
			}
		})
	}
}

func BenchmarkPathFromSlice(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			data := make([]Pair, size)
			s := randBts(size)
			for i, key := range rand.Perm(size) {
				data[i] = Pair{uint(key), s[i]}
			}
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = PathFromSlice(data)
			}
		})
	}
}

func BenchmarkPathWithout(b *testing.B) {
	for _, size := range sizes {
		b.Run(fmt.Sprintf("%d", size), func(b *testing.B) {
			path := makePath(size)
			rid := rand.Perm(size)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = path.Without(uint(rid[i%size]))
			}
		})
	}
}

func doSomeFunc(it func() (uint, bool)) {
	for ok := true; ok; _, ok = it() {
	}
}

func BenchmarkPathAscendKeyIterator(b *testing.B) {
	for _, size := range sizes {
		path := makePath(size)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			p := path.Begin()
			doSomeFunc(func() (key uint, ok bool) {
				p, key, ok = path.NextKey(p)
				return
			})
		}
	}
}

func makePath(size int) Path {
	data := make([]Pair, size)
	s := randBts(size)
	rid := rand.Perm(size)
	for i, key := range rid {
		data[i] = Pair{uint(key), s[i]}
	}
	return PathFromSlice(data)
}
