package radix

import (
	"math/rand"
	"sort"
	"testing"
)

type item struct {
	p Pairs
	v int
}

type del struct {
	item
	ok bool
}

func TestTrieInsertLookup(t *testing.T) {
	for i, test := range []struct {
		insert []item
		lookup []Pairs
		expect []int
	}{
		{
			insert: []item{
				{Pairs{{1, "a"}, {2, "b"}}, 1},
			},
			lookup: []Pairs{
				Pairs{{1, "a"}, {2, "b"}},
				Pairs{{2, "b"}, {1, "a"}},
			},
			expect: []int{1},
		},
		{
			insert: []item{
				{Pairs{{1, "a"}, {2, "b"}}, 1},
				{Pairs{{1, "a"}, {2, "b"}}, 2},
				{Pairs{{1, "a"}}, 3},
				{Pairs{{2, "b"}}, 4},
				{Pairs{}, 5},
			},
			lookup: []Pairs{
				Pairs{{1, "a"}, {2, "b"}},
				Pairs{{2, "b"}, {1, "a"}},
			},
			expect: []int{1, 2, 3, 4, 5},
		},
	} {
		trie := New()
		for _, op := range test.insert {
			trie.Insert(op.p, op.v)
		}

		for _, path := range test.lookup {
			var result []int
			trie.Lookup(path, func(v int) bool {
				result = append(result, v)
				return true
			})
			if !listEq(result, test.expect) {
				t.Errorf("[%d] Lookup(%v) = %v; want %v", i, path, result, test.expect)
			}
		}
	}
}

func TestTrieInsertDelete(t *testing.T) {
	for i, test := range []struct {
		insert []item
		delete []del
		expect []int
	}{
		{
			insert: []item{
				{Pairs{{1, "a"}, {2, "b"}}, 1},
			},
			delete: []del{
				{item{Pairs{{1, "a"}, {2, "b"}}, 1}, true},
			},
			expect: []int{},
		},
		{
			insert: []item{
				{Pairs{{1, "a"}, {2, "b"}}, 1},
				{Pairs{{1, "a"}, {2, "b"}}, 2},
				{Pairs{{1, "a"}}, 3},
				{Pairs{{2, "b"}}, 4},
				{Pairs{}, 5},
			},
			delete: []del{
				{item{Pairs{{1, "a"}, {2, "b"}}, 1}, true},
				{item{Pairs{{1, "a"}}, 3}, true},
				{item{Pairs{{1, "a"}}, 4}, false},
			},
			expect: []int{2, 4, 5},
		},
	} {
		trie := New()
		for _, op := range test.insert {
			trie.Insert(op.p, op.v)
		}
		for _, del := range test.delete {
			if del.ok != trie.Delete(del.p, del.v) {
				t.Errorf("[%d] Delete(%v, %v) = %v; want %v", i, del.p, del.v, !del.ok, del.ok)
			}
		}
		var result []int
		trie.Lookup(nil, func(v int) bool {
			result = append(result, v)
			return true
		})
		if !listEq(result, test.expect) {
			t.Errorf(
				"[%d] after Delete; Lookup(%v) = %v; want %v",
				i, Pairs{}, result, test.expect,
			)
		}
	}
}

func listEq(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	sort.Ints(a)
	sort.Ints(b)
	for i, av := range a {
		if b[i] != av {
			return false
		}
	}
	return true
}

func randStr(n, size int) (ret []string) {
	dup := make(map[string]bool)
	var b []byte
	for i := 0; i < n; i++ {
		for {
			b = b[:0]
			for j := 0; j < size; j++ {
				b = append(b, byte(rand.Intn('z'-'a'+1)+'a'))
			}
			if !dup[string(b)] {
				ret = append(ret, string(b))
				dup[string(b)] = true
				break
			}
		}
	}
	return
}

func benchmarkInsert(b *testing.B, exists, n int) {
	var v int
	t := New()
	values := randStr(exists+n, 16)
	for i := 0; i < exists; i++ {
		t.Insert(Pairs{{1, values[i]}}, v)
		v++
	}
	insert := values[exists:]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(insert); j++ {
			t.Insert(Pairs{{1, insert[j]}}, v)
			v++
		}
		b.StopTimer()
		for j := len(insert) - 1; j >= 0; j-- {
			v--
			if !t.Delete(Pairs{{1, insert[j]}}, v) {
				b.Fatalf("could not delete previously inserted element")
			}
		}
		b.StartTimer()
	}
}

func BenchmarkTrieInsert_0_10(b *testing.B)     { benchmarkInsert(b, 0, 10) }
func BenchmarkTrieInsert_0_1000(b *testing.B)   { benchmarkInsert(b, 0, 1000) }
func BenchmarkTrieInsert_0_100000(b *testing.B) { benchmarkInsert(b, 0, 100000) }
