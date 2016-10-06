package radix

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"testing"
)

type pairs []Pair

type item struct {
	p pairs
	v int
}

type item_p struct {
	p Path
	v int
}

type del struct {
	item
	ok bool
}

func TestTrieInsert(t *testing.T) {
	for i, test := range []struct {
		insert pairs
		values []int
	}{
		{
			insert: pairs{{1, "a"}, {2, "b"}},
			values: []int{1, 2, 3, 4},
		},
	} {
		trie := New()
		for _, v := range test.values {
			trie.Insert(PathFromSlice(test.insert...), v)
		}
		trie.heap.Ascend(func(x *node) bool {
			if x.key != 2 {
				return true
			}
			if !listEq(x.leaf("b").data, test.values) {
				t.Errorf("[%d] leaf values is %v; want %v", i, x.leaf("b").data, test.values)
			}
			return false
		})
	}
}

func TestTrieInsertLookup(t *testing.T) {
	for i, test := range []struct {
		insert []item
		lookup []pairs
		expect []int
	}{
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
			},
			lookup: []pairs{
				pairs{{1, "a"}, {2, "b"}},
				pairs{{2, "b"}, {1, "a"}},
			},
			expect: []int{1},
		},
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
				{pairs{{1, "a"}, {2, "b"}}, 2},
				{pairs{{1, "a"}}, 3},
				{pairs{{2, "b"}}, 4},
				{pairs{}, 5},
			},
			lookup: []pairs{
				pairs{{1, "a"}, {2, "b"}},
				pairs{{2, "b"}, {1, "a"}},
			},
			expect: []int{1, 2, 3, 4, 5},
		},
	} {
		trie := New()
		for _, op := range test.insert {
			trie.Insert(PathFromSlice(op.p...), op.v)
		}

		for _, p := range test.lookup {
			var result []int
			trie.Lookup(PathFromSlice(p...), func(v int) bool {
				result = append(result, v)
				return true
			})
			if !listEq(result, test.expect) {
				t.Errorf("[%d] Lookup(%v) = %v; want %v", i, p, result, test.expect)
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
				{pairs{{1, "a"}, {2, "b"}}, 1},
			},
			delete: []del{
				{item{pairs{{1, "a"}, {2, "b"}}, 1}, true},
			},
			expect: []int{},
		},
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
				{pairs{{1, "a"}, {2, "b"}}, 2},
				{pairs{{1, "a"}}, 3},
				{pairs{{2, "b"}}, 4},
				{pairs{}, 5},
			},
			delete: []del{
				{item{pairs{{1, "a"}, {2, "b"}}, 1}, true},
				{item{pairs{{1, "a"}}, 3}, true},
				{item{pairs{{1, "a"}}, 4}, false},
			},
			expect: []int{2, 4, 5},
		},
	} {
		trie := New()
		for _, op := range test.insert {
			trie.Insert(PathFromSlice(op.p...), op.v)
		}
		for _, del := range test.delete {
			if del.ok != trie.Delete(PathFromSlice(del.p...), del.v) {
				t.Errorf("[%d] Delete(%v, %v) = %v; want %v", i, del.p, del.v, !del.ok, del.ok)
			}
		}
		var result []int
		trie.Lookup(PathFromSlice(), func(v int) bool {
			result = append(result, v)
			return true
		})
		if !listEq(result, test.expect) {
			t.Errorf(
				"[%d] after Delete; Lookup(%v) = %v; want %v",
				i, pairs{}, result, test.expect,
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

func benchmarkInsert(b *testing.B, exists int) {
	t := New()
	values := randStr(exists+1, 16)
	for i := 0; i < exists; i++ {
		t.Insert(PathFromSlice(Pair{1, values[i]}), i)
	}
	insert := values[len(values)-1]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(insert); j++ {
			t.Insert(PathFromSlice(Pair{1, insert}), exists)
		}
	}
}

func BenchmarkTrieInsert_0(b *testing.B)      { benchmarkInsert(b, 0) }
func BenchmarkTrieInsert_1000(b *testing.B)   { benchmarkInsert(b, 1000) }
func BenchmarkTrieInsert_100000(b *testing.B) { benchmarkInsert(b, 100000) }

func fill(t *Trie, path []Pair, d, n, v int, values []string, k, m int) (ret []item_p) {
	if d == 0 {
		return
	}
	for i := 0; i < n; i++ {
		for j := 0; j < v; j++ {
			np := append(path, Pair{uint(m + i), values[k]})
			ps := PathFromSlice(np...)
			val := 0
			if d == 1 {
				ret = append(ret, item_p{ps, val})
			}
			fmt.Println("INSERT:", np, m+i)
			t.Insert(ps, val)
			k++
			ret = append(ret, fill(t, np, d-1, n, v, values, k, m+n)...)
		}
	}
	return
}

func trieSize(d, n, v int) (int, int) {
	nodes := float64(n)
	var i float64
	for i = 1; i < float64(d); i++ {
		nodes += i * float64(v) * math.Pow(float64(n), i)
	}
	return int(nodes), int(nodes * float64(v))
}

func benchmarkLookup(b *testing.B, d, n, v int) {
	_, leafs := trieSize(d, n, v)
	values := randStr(leafs, 8)
	t := New()
	fill(t, nil, d, n, v, values, 0, 0)
	b.ResetTimer()
	// initial tree
	Graphviz(os.Stdout, fmt.Sprintf("bench_%d_%d_%d", d, n, v), t)
	os.Stdout.Write([]byte{'\n', '\n'})
}

func BenchmarkTrieLookup_1_1_2(b *testing.B) { benchmarkLookup(b, 1, 1, 2) }
func BenchmarkTrieLookup_2_1_2(b *testing.B) { benchmarkLookup(b, 2, 1, 1) }
