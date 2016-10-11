package radix_test

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"testing"

	. "github.com/gobwas/radix"
	"github.com/gobwas/radix/graphviz"
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
		var data []int
		trie.ForEach(func(p Path, v int) bool {
			data = append(data, v)
			return true
		})
		if !listEq(data, test.values) {
			t.Errorf("[%d] leaf values is %v; want %v", i, data, test.values)
		}
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
				buf := &bytes.Buffer{}
				graphviz.Render(trie, fmt.Sprintf("test-%d", i), buf)
				t.Errorf("[%d] Lookup(%v) = %v; want %v\nTrie graphviz:\n%s\n", i, p, result, test.expect, buf.String())
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
		trie.ForEach(func(p Path, v int) bool {
			result = append(result, v)
			return true
		})
		if !listEq(result, test.expect) {
			buf := &bytes.Buffer{}
			graphviz.Render(trie, fmt.Sprintf("test-%d", i), buf)
			t.Errorf(
				"[%d] after Delete; Lookup(%v) = %v; want %v\nTrie graphviz:\n%s\n",
				i, pairs{}, result, test.expect, buf.String(),
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

func randStrn2(n, m int) []string {
	dup := make(map[string]bool, n)
	ret := make([]string, n)
	b := make([]byte, m)
	for i := 0; i < n; i++ {
		for {
			_, err := rand.Read(b)
			if err != nil {
				panic(err)
			}
			for j, v := range b {
				b[j] = (v & 0x0e) + 'a'
			}
			if !dup[string(b)] {
				dup[string(b)] = true
				ret[i] = string(b)
				break
			}
		}
	}
	return ret
}

func randStrn(n, m int) (ret []string) {
	dup := make(map[string]bool, n)
	ret = make([]string, 0, n)
	b := make([]byte, 0, m)
	for i := 0; i < n; i++ {
		for {
			b = b[:0]
			for j := 0; j < m; j++ {
				b = append(b, byte(rand.Intn('z'-'a'+1)+'a'))
			}
			if !dup[string(b)] {
				dup[string(b)] = true
				break
			}
		}
		ret = append(ret, string(b))
	}
	return
}

func randStr(n int) (ret []string) {
	return randStrn2(n, 8)
}

func benchmarkInsert(b *testing.B, exists int) {
	t := New()
	values := randStr(exists + 1)
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

func BenchmarkTrieInsert_0(b *testing.B)       { benchmarkInsert(b, 0) }
func BenchmarkTrieInsert_1000(b *testing.B)    { benchmarkInsert(b, 1000) }
func BenchmarkTrieInsert_100000(b *testing.B)  { benchmarkInsert(b, 100000) }
func BenchmarkTrieInsert_1000000(b *testing.B) { benchmarkInsert(b, 1000000) }

func fill(t *Trie, path []Pair, d, n, v int, values []string, ret *[]item_p, k *int, m int, val *int) {
	if d == 0 {
		return
	}
	for i := 0; i < n; i++ {
		for j := 0; j < v; j++ {
			np := append(path, Pair{uint(m + i), values[*k]})
			ps := PathFromSlice(np...)
			t.Insert(ps, *val)
			if d == 1 {
				*ret = append(*ret, item_p{ps, *val})
			}
			*k++
			*val++
			fill(t, np, d-1, n, v, values, ret, k, (m+i+j)*int(math.Pow(10, float64(i+1))), val)
		}
	}
}

func trieSize(d, n, v int) int {
	s := n * v
	for i := 1; i < d; i++ {
		s *= n * v
	}
	return s
}

func genTrie(d, n, v int) (*Trie, []item_p) {
	leafs := trieSize(d, n, v)
	values := randStr(leafs)
	t := New()
	r := make([]item_p, 0)
	fill(t, nil, d, n, v, values, &r, new(int), 1, new(int))
	return t, r
}

func genTries(count, d, n, v int) (ret []*Trie, del [][]item_p) {
	for i := 0; i < count; i++ {
		t, r := genTrie(d, n, v)
		ret = append(ret, t)
		del = append(del, r)
	}
	return
}

func benchmarkLookup(b *testing.B, d, n, v int) {
	trie, deepest := genTrie(d, n, v)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := deepest[i%len(deepest)]
		trie.Lookup(item.p, func(v int) bool { return true })
	}
}

func benchmarkDelete(b *testing.B, d, n, v int) {
	var trie *Trie
	var del []item_p
	for i := 0; i < b.N; i++ {
		if len(del) == 0 {
			b.StopTimer()
			trie, del = genTrie(d, n, v)
			b.StartTimer()
		}
		item := del[len(del)-1]
		del = del[:len(del)-1]
		if !trie.Delete(item.p, item.v) {
			b.Fatalf("could not delete item: %v %v", item.p.String(), item.v)
		}
	}
}

//func BenchmarkMap(b *testing.B) {
//	m := make(map[string]int, 1000)
//	s := randStrn(1000, 16)
//	for i := 0; i < 1000; i++ {
//		m[s[i]] = i
//	}
//	b.ResetTimer()
//	for i := 0; i < b.N; i++ {
//		_ = m[s[i%1000]]
//	}
//}

func BenchmarkTrieDelete_1_1_100(b *testing.B)     { benchmarkDelete(b, 1, 1, 100) }
func BenchmarkTrieDelete_1_1_1000(b *testing.B)    { benchmarkDelete(b, 1, 1, 1000) }
func BenchmarkTrieDelete_1_1_10000(b *testing.B)   { benchmarkDelete(b, 1, 1, 10000) }
func BenchmarkTrieDelete_1_1_1000000(b *testing.B) { benchmarkDelete(b, 1, 1, 1000000) }

func BenchmarkTrieLookup_1_1_100(b *testing.B)     { benchmarkLookup(b, 1, 1, 100) }
func BenchmarkTrieLookup_1_1_1000(b *testing.B)    { benchmarkLookup(b, 1, 1, 1000) }
func BenchmarkTrieLookup_1_1_10000(b *testing.B)   { benchmarkLookup(b, 1, 1, 10000) }
func BenchmarkTrieLookup_1_1_1000000(b *testing.B) { benchmarkLookup(b, 1, 1, 1000000) }
