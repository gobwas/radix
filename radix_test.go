package radix_test

//go:generate ppgo

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"testing"

	. "github.com/gobwas/radix"
	"github.com/gobwas/radix/graphviz"
	"github.com/gobwas/radix/listing"
)

type pairs []Pair

type item struct {
	p pairs
	v uint
}

type item_p struct {
	p Path
	v uint
}

type del struct {
	item
	ok bool
}

func TestTrieInsert(t *testing.T) {
	for i, test := range []struct {
		insert pairs
		values []uint
	}{
		{
			insert: pairs{{1, "a"}, {2, "b"}},
			values: []uint{1, 2, 3, 4},
		},
	} {
		trie := New(nil)
		for _, v := range test.values {
			trie.Insert(PathFromSlice(test.insert), v)
		}
		var data []uint
		trie.ForEach(Path{}, func(p []Pair, v uint) bool {
			data = append(data, v)
			return true
		})
		if !listEq(data, test.values) {
			t.Errorf("[%d] leaf values is %v; want %v", i, data, test.values)
		}
	}
}

func TestInserterInsert(t *testing.T) {
	for i, test := range []struct {
		inserter Inserter
		insert   []item
		exp      map[uint][]Pair
	}{
		{
			inserter: Inserter{
				NodeOrder: []uint{2},
			},
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
			},
			exp: map[uint][]Pair{
				1: pairs{{2, "b"}, {1, "a"}},
			},
		},
		{
			inserter: Inserter{
				NodeOrder: []uint{3, 2, 1},
			},
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
				{pairs{{1, "a"}}, 2},
				{pairs{{2, "b"}}, 3},
				{pairs{}, 4},
			},
			exp: map[uint][]Pair{
				1: pairs{{2, "b"}, {1, "a"}},
				2: pairs{{1, "a"}},
				3: pairs{{2, "b"}},
				4: nil,
			},
		},
		// Partial case.
		{
			inserter: Inserter{
				NodeOrder: []uint{3, 4, 5},
			},
			insert: []item{
				{pairs{{1, "a"}, {2, "st"}}, 1},
				{pairs{{1, "b"}, {2, "st"}}, 2},
				{pairs{{3, "c"}, {1, "a"}, {2, "st"}}, 3},
				{pairs{{3, "c"}, {1, "b"}, {2, "st"}}, 4},
			},
			exp: map[uint][]Pair{
				1: pairs{{1, "a"}, {2, "st"}},
				2: pairs{{1, "b"}, {2, "st"}},
				3: pairs{{3, "c"}, {1, "a"}, {2, "st"}},
				4: pairs{{3, "c"}, {1, "b"}, {2, "st"}},
			},
		},
	} {
		label := fmt.Sprintf("Insert#%d", i)

		t.Run(label, func(t *testing.T) {
			root := NewLeaf(nil, "root")
			for _, op := range test.insert {
				test.inserter.Insert(root, PathFromSliceBorrow(op.p), op.v)
			}

			act := map[uint][]Pair{}
			ForEach(root, Path{}, func(p []Pair, v uint) bool {
				act[v] = p
				return true
			})

			for v, actPairs := range act {
				if expPairs := test.exp[v]; !reflect.DeepEqual(actPairs, expPairs) {
					t.Errorf("After insertion got for %v:\n\t%#v\n\twant:\n\t%#v\n", v, actPairs, expPairs)

					expRoot := NewLeaf(nil, "root")
					for v, p := range test.exp {
						Inserter{}.ForceInsert(expRoot, p, v)
					}
					if err := graphviz.ShowLeaf(expRoot, label+"_exp"); err != nil {
						t.Logf("could not open trie representation: %s", err)
					}
					if err := graphviz.ShowLeaf(root, label+"_act"); err != nil {
						t.Logf("could not open trie representation: %s", err)
					}
				}
			}

		})
	}

}

func TestLookup(t *testing.T) {
	for i, test := range []struct {
		insert []item
		lookup []pairs
		exp    []uint
	}{
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
			},
			lookup: []pairs{
				pairs{{1, "a"}, {2, "b"}},
				pairs{{2, "b"}, {1, "a"}},
			},
			exp: []uint{1},
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

			exp: []uint{1, 2, 3, 4, 5},
		},

		// Partial case.
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "st"}}, 1},
				{pairs{{1, "b"}, {2, "st"}}, 2},
				{pairs{{3, "c"}, {1, "a"}, {2, "st"}}, 3},
				{pairs{{3, "c"}, {1, "b"}, {2, "st"}}, 4},
			},
			lookup: []pairs{
				pairs{{1, "a"}, {2, "st"}},
			},
			exp: []uint{1},
		},
	} {
		label := fmt.Sprintf("Lookup#%d", i)

		t.Run(label, func(t *testing.T) {
			root := NewLeaf(nil, "root")
			for _, op := range test.insert {
				(&Inserter{}).ForceInsert(root, op.p, op.v)
			}

			for i, p := range test.lookup {
				var value []uint
				Lookup(root, PathFromSlice(p), LookupStrategyGreedy, func(leaf *Leaf) bool {
					value = append(value, leaf.Data()...)
					return true
				})
				if !listEq(value, test.exp) {
					t.Errorf(
						"[%d] Lookup(%v) = %v; want %v",
						i, p, value, test.exp,
					)
					if err := graphviz.ShowLeaf(root, label); err != nil {
						t.Logf("could not open trie representation: %s", err)
					}
				}
			}
		})
	}
}

func TestSelect(t *testing.T) {
	for i, test := range []struct {
		insert   []item
		lookup   []pairs
		capture  []uint
		strategy LookupStrategy
		exp      map[uint]Path
	}{
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}, {3, "c"}}, 1},
			},
			lookup: []pairs{
				pairs{{1, "a"}, {3, "c"}},
				pairs{{3, "c"}, {1, "a"}},
			},
			capture:  []uint{2},
			strategy: LookupStrategyGreedy,
			exp: map[uint]Path{
				1: PathFromSliceBorrow([]Pair{{2, "b"}}),
			},
		},
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}, {3, "c"}}, 1},
				{pairs{{1, "a"}, {2, "b"}, {3, "c"}}, 2},
				{pairs{{2, "b"}, {3, "c"}}, 3},
				{pairs{{2, "b"}, {3, "c"}}, 4},
				{pairs{}, 5},
			},
			lookup: []pairs{
				pairs{{1, "a"}, {3, "c"}},
				pairs{{3, "c"}, {1, "a"}},
			},
			capture:  []uint{2},
			strategy: LookupStrategyGreedy,
			exp: map[uint]Path{
				1: PathFromSliceBorrow([]Pair{{2, "b"}}),
				2: PathFromSliceBorrow([]Pair{{2, "b"}}),
				3: PathFromSliceBorrow([]Pair{{2, "b"}}),
				4: PathFromSliceBorrow([]Pair{{2, "b"}}),
				5: PathFromSliceBorrow([]Pair{{2, ""}}),
			},
		},
	} {
		label := fmt.Sprintf("Select#%d", i)

		t.Run(label, func(t *testing.T) {
			root := NewLeaf(nil, "root")
			for _, op := range test.insert {
				(&Inserter{}).ForceInsert(root, op.p, op.v)
			}

			var c []Pair
			for _, k := range test.capture {
				c = append(c, Pair{Key: k})
			}
			capture := PathFromSlice(c)

			for _, p := range test.lookup {
				var trace = map[uint]Path{}
				Select(root, PathFromSlice(p), capture, LookupStrategyGreedy, func(c Path, l *Leaf) bool {
					for _, v := range l.Data() {
						trace[v] = c
					}
					return true
				})

				for v, trace := range trace {
					if exp := test.exp[v]; !trace.Equal(exp) {
						t.Errorf(
							"[%d] Select(%v) returned %v with %v trace; want %v",
							i, p, v, trace, exp,
						)
						if err := graphviz.ShowLeaf(root, label); err != nil {
							t.Logf("could not open trie representation: %s", err)
						}
					}
				}
			}
		})
	}
}

type countVisitor struct {
	nodes int
	leafs int
}

func (c *countVisitor) OnNode(p []Pair, n *Node) bool { c.nodes++; return true }
func (c *countVisitor) OnLeaf(p []Pair, l *Leaf) bool { c.leafs++; return true }

func TestTrieDeleteCleanup(t *testing.T) {
	trie := New(nil)
	path := PathFromSlice(pairs{{1, "a"}, {2, "b"}})

	trie.Insert(path, 1)

	before := &countVisitor{}
	trie.Walk(Path{}, before)
	if n, l := before.nodes, before.leafs; n != 2 || l != 3 { // leafs 3 is with root leaf
		listing.Dump(os.Stderr, trie)
		buf := &bytes.Buffer{}
		listing.Dump(buf, trie)
		t.Errorf("after insertion: nodes: %d; leafs: %d; want 2 and 2;\ntrie:\n %s", n, l, buf.String())
	}

	trie.Delete(path, 1)

	after := &countVisitor{}
	trie.Walk(Path{}, after)
	if n, l := after.nodes, after.leafs; n != 0 || l != 1 {
		buf := &bytes.Buffer{}
		listing.Dump(buf, trie)
		t.Errorf("after deletion: nodes: %d; leafs: %d; want 0 and 1;\ntrie:\n %s", n, l, buf.String())
	}
}

func TestTrieInsertDelete(t *testing.T) {
	for i, test := range []struct {
		config *TrieConfig
		insert []item
		delete []del
		expect []uint
	}{
		{
			config: &TrieConfig{
				NodeOrder: []uint{0x10000, 0x10001},
			},
			insert: []item{
				{pairs{{0x1, "test"}, {0x10000, "1"}, {0x10001, "1"}}, 42},
			},
			delete: []del{
				{item{pairs{{0x1, "test"}, {0x10000, "1"}, {0x10001, "1"}}, 42}, true},
			},
			expect: []uint{},
		},
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
			},
			delete: []del{
				{item{pairs{{1, "a"}, {2, "b"}}, 1}, true},
			},
			expect: []uint{},
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
			expect: []uint{2, 4, 5},
		},
	} {
		trie := New(test.config)
		for _, op := range test.insert {
			trie.Insert(PathFromSlice(op.p), op.v)
		}

		before := listing.DumpString(trie)

		for _, del := range test.delete {
			if del.ok != trie.Delete(PathFromSlice(del.p), del.v) {
				t.Errorf("[%d] Delete(%v, %v) = %v; want %v", i, del.p, del.v, !del.ok, del.ok)
			}
		}
		var result []uint
		trie.ForEach(Path{}, func(p []Pair, v uint) bool {
			result = append(result, v)
			return true
		})
		if !listEq(result, test.expect) {
			after := listing.DumpString(trie)
			t.Errorf(
				"[%d] after Delete; Lookup(%v) = %v; want %v\nTrie before:\n%s\nTrie after:\n%s\n",
				i, pairs{}, result, test.expect, before, after,
			)
		}
	}
}

func listEq(a, b []uint) bool {
	if len(a) != len(b) {
		return false
	}
	uintSort(a, 0, len(a))
	uintSort(b, 0, len(a))
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
	t := New(nil)
	values := randStr(exists + 1)
	for i := 0; i < exists; i++ {
		t.Insert(PathFromSlice([]Pair{{1, values[i]}}), uint(i))
	}
	insert := values[len(values)-1]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(insert); j++ {
			t.Insert(PathFromSlice([]Pair{{1, insert}}), uint(exists))
		}
	}
}

func BenchmarkTrieInsert_0(b *testing.B)       { benchmarkInsert(b, 0) }
func BenchmarkTrieInsert_1000(b *testing.B)    { benchmarkInsert(b, 1000) }
func BenchmarkTrieInsert_100000(b *testing.B)  { benchmarkInsert(b, 100000) }
func BenchmarkTrieInsert_1000000(b *testing.B) { benchmarkInsert(b, 1000000) }

func fill(t *Trie, path []Pair, d, n, v int, values []string, ret *[]item_p, k *int, m int, val *uint) {
	if d == 0 {
		return
	}
	for i := 0; i < n; i++ {
		for j := 0; j < v; j++ {
			np := append(path, Pair{uint(m + i), values[*k]})
			ps := PathFromSlice(np)
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
	t := New(nil)
	r := make([]item_p, 0)
	fill(t, nil, d, n, v, values, &r, new(int), 1, new(uint))
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

type lookupBench struct {
	depth, nodes, values int
}

var lookupBenches = []lookupBench{
	{1, 1, 100},
	{1, 1, 1000},
	{1, 1, 10000},
	{1, 1, 1000000},
}

func BenchmarkTrieLookup(b *testing.B) {
	for _, bench := range lookupBenches {
		b.Run(fmt.Sprintf("%d_%d_%d", bench.depth, bench.nodes, bench.values), func(b *testing.B) {
			trie, deepest := genTrie(bench.depth, bench.nodes, bench.values)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				item := deepest[i%len(deepest)]
				trie.LookupGreedy(item.p, func(v uint) bool { return true })
			}
		})
	}
}

func TestTrieSelect(t *testing.T) {
	trie := New(&TrieConfig{
		NodeOrder: []uint{1, 2, 3},
	})

	trie.Insert(PathFromMap(map[uint]string{1: "a"}), 0xff01)
	trie.Insert(PathFromMap(map[uint]string{3: "b"}), 0xff02)
	trie.Insert(PathFromMap(map[uint]string{3: "c"}), 0xff03)
	trie.Insert(PathFromMap(map[uint]string{1: "a", 3: "b"}), 0xbb01)
	trie.Insert(PathFromMap(map[uint]string{1: "a", 3: "c"}), 0xbb02)

	for i := 0; i < 2; i++ {
		k := "key" + strconv.FormatInt(int64(i), 16)
		trie.Insert(PathFromMap(map[uint]string{1: "a", 2: k, 3: "b"}), uint(i))
	}
	for i := 2; i < 4; i++ {
		k := "key" + strconv.FormatInt(int64(i), 16)
		trie.Insert(PathFromMap(map[uint]string{1: "a", 2: k, 3: "c"}), uint(i))
	}

	lookup := PathFromMap(map[uint]string{1: "a", 3: "c"})
	capture := PathFromMap(map[uint]string{2: ""})

	expStrict := map[string]uint{
		PathFromMap(map[uint]string{2: "key2"}).String(): 2,
		PathFromMap(map[uint]string{2: "key3"}).String(): 3,
		PathFromMap(map[uint]string{2: ""}).String():     0xbb02,
	}
	expGreedy := map[string]uint{
		PathFromMap(map[uint]string{2: "key2"}).String(): 2,
		PathFromMap(map[uint]string{2: "key3"}).String(): 3,
		PathFromMap(map[uint]string{2: ""}).String():     0xbb02,
		PathFromMap(map[uint]string{2: ""}).String():     0xff01,
		PathFromMap(map[uint]string{2: ""}).String():     0xff03,
	}

	actStrict := map[string]uint{}
	actGreedy := map[string]uint{}

	fmt.Println(listing.DumpString(trie))

	trie.SelectStrict(lookup, capture.Copy(), func(c Path, v uint) bool { actStrict[c.String()] = v; return true })
	trie.SelectGreedy(lookup, capture.Copy(), func(c Path, v uint) bool { actGreedy[c.String()] = v; return true })

	if !reflect.DeepEqual(actStrict, expStrict) {
		t.Errorf(
			"unexpected strict results:\n\tlookup: %#v; capture %#v;\n\t%#v;\n\twant:\n\t%#v",
			lookup.String(), capture.String(), actStrict, expStrict,
		)
	}
	if !reflect.DeepEqual(actGreedy, expGreedy) {
		t.Errorf(
			"unexpected greedy results:\n\tlookup: %#v; capture %#v;\n\t%#v;\n\twant:\n\t%#v",
			lookup.String(), capture.String(), actGreedy, expGreedy,
		)
	}
}

func BenchmarkTrieSelectStrict(b *testing.B) {
	for _, test := range []struct {
		items   int
		capture int
	}{
		{items: 1, capture: 1},
		{items: 4, capture: 1},
		{items: 16, capture: 1},
		{items: 64, capture: 1},
		{items: 256, capture: 1},
	} {
		b.Run(fmt.Sprintf("i%dc%d", test.items, test.capture), func(b *testing.B) {
			trie := New(nil)

			cm := make(map[uint]string)
			for j := 0; j < test.capture; j++ {
				cm[uint(j+2)] = ""
			}

			capture := PathFromMap(cm)
			lookup := PathFromMap(map[uint]string{1: "a", uint(test.capture + 2): "b"})

			for i := 0; i < test.items; i++ {
				prefix := "key" + strconv.FormatInt(int64(i), 16)
				mp := map[uint]string{
					1: "a",
					uint(test.capture + 2): "b",
				}
				for j := 0; j < test.capture; j++ {
					mp[uint(j+2)] = prefix + "_" + strconv.FormatInt(int64(j), 16)
				}
				trie.Insert(PathFromMap(mp), uint(i))
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				trie.SelectStrict(lookup, capture, func(c Path, _ uint) bool { return true })
			}
		})
	}
}
