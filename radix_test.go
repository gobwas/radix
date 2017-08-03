package radix_test

//go:generate ppgo

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"testing"

	. "github.com/gobwas/radix"
	"github.com/gobwas/radix/graphviz"
	"github.com/gobwas/radix/listing"
)

type pairs []PairStr

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
			trie.Insert(PathFromSliceStr(test.insert), v)
		}
		var data []uint
		trie.LookupStrict(PathFromSliceStr(test.insert), func(v uint) bool {
			data = append(data, v)
			return true
		})
		if !listEq(data, test.values) {
			t.Errorf("[%d] leaf values is %v; want %v", i, data, test.values)
		}
	}
}

func TestTrieInsertTo(t *testing.T) {
	for i, test := range []struct {
		root   pairs
		insert pairs
		values []uint
	}{
		{
			root:   pairs{{1, "a"}, {2, "b"}},
			insert: pairs{{3, "c"}, {4, "d"}},
			values: []uint{1, 2, 3, 4},
		},
	} {
		trie := New(nil)
		root := trie.At(PathFromSliceStr(test.root))

		for _, v := range test.values {
			trie.InsertTo(root, PathFromSliceStr(test.insert), v)
		}
		var data []uint
		trie.LookupStrict(
			PathFromSliceStr(append(test.root, test.insert...)),
			func(v uint) bool {
				data = append(data, v)
				return true
			},
		)
		if !listEq(data, test.values) {
			t.Errorf(
				"[%d] leaf values is %v; want %v\nTrie:\n%s\n",
				i, data, test.values, listing.DumpString(trie),
			)
		}
	}
}

func pairsMaker(total int) func(n int) []Pair {
	values := randBtsn(total, 16)
	var (
		key   uint
		index int
	)
	return func(n int) []Pair {
		ret := make([]Pair, n)
		for i := range ret {
			ret[i] = Pair{
				Key:   key,
				Value: values[index],
			}
			key++
			index++
		}
		return ret
	}
}

func getPartedPaths(size, suffixSize, count int) (prefix Path, suffix, full []Path) {
	prefixSize := size - suffixSize
	if prefixSize < 0 {
		panic("size of full path is less that suffix")
	}

	makePairs := pairsMaker(prefixSize + (suffixSize * count))

	prefixPairs := makePairs(prefixSize)
	suffixes := make([][]Pair, count)
	for i := range suffixes {
		suffixes[i] = makePairs(suffixSize)
	}

	full = make([]Path, count)
	suffix = make([]Path, count)
	for i, suffixPairs := range suffixes {
		suffix[i] = PathFromSlice(suffixPairs)
		full[i] = PathFromSlice(append(prefixPairs, suffixPairs...))
	}

	prefix = PathFromSlice(prefixPairs)

	return
}

var insertCases = []struct {
	path   int
	suffix int
	n      int
}{
	{3, 1, 4},
}

func BenchmarkTrieInsert(b *testing.B) {
	for _, test := range insertCases {
		b.Run(fmt.Sprintf("prefix=%d", test.path-test.suffix), func(b *testing.B) {
			_, _, full := getPartedPaths(test.path, test.suffix, test.n)

			var trie *Trie
			for i := 0; i < b.N/test.n; i++ {
				b.StopTimer()
				trie = New(nil)
				b.StartTimer()

				for j, path := range full {
					trie.Insert(path, uint(j))
				}
			}
		})
	}
}

func BenchmarkTrieInsertTo(b *testing.B) {
	for _, test := range insertCases {
		b.Run(fmt.Sprintf("prefix=%d", test.path-test.suffix), func(b *testing.B) {
			prefix, suffixes, _ := getPartedPaths(test.path, test.suffix, test.n)

			var trie *Trie
			for i := 0; i < b.N/test.n; i++ {
				b.StopTimer()
				trie = New(nil)
				b.StartTimer()

				root := trie.At(prefix)
				for j, suffix := range suffixes {
					trie.InsertTo(root, suffix, uint(j))
				}
			}
		})
	}
}

func TestInserterInsert(t *testing.T) {
	for i, test := range []struct {
		inserter Inserter
		insert   []item
		exp      map[uint][]PairStr
	}{
		{
			inserter: Inserter{
				NodeOrder: []uint{2},
			},
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
			},
			exp: map[uint][]PairStr{
				1: []PairStr{{2, "b"}, {1, "a"}},
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
			exp: map[uint][]PairStr{
				1: []PairStr{{2, "b"}, {1, "a"}},
				2: []PairStr{{1, "a"}},
				3: []PairStr{{2, "b"}},
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
			exp: map[uint][]PairStr{
				1: []PairStr{{1, "a"}, {2, "st"}},
				2: []PairStr{{1, "b"}, {2, "st"}},
				3: []PairStr{{3, "c"}, {1, "a"}, {2, "st"}},
				4: []PairStr{{3, "c"}, {1, "b"}, {2, "st"}},
			},
		},
	} {
		label := fmt.Sprintf("Insert#%d", i)

		t.Run(label, func(t *testing.T) {
			root := NewLeaf(nil, "root")
			for _, op := range test.insert {
				test.inserter.Insert(root, PathFromSliceStr(op.p), op.v)
			}

			act := map[uint][]PairStr{}
			ForEach(root, Path{}, func(p []PairStr, v uint) bool {
				act[v] = p
				return true
			})

			for v, actPairs := range act {
				if expPairs := test.exp[v]; !reflect.DeepEqual(actPairs, expPairs) {
					t.Errorf("After insertion got for %v:\n\t%#v\n\twant:\n\t%#v\n", v, actPairs, expPairs)

					expRoot := NewLeaf(nil, "root")
					for v, p := range test.exp {
						Inserter{}.ForceInsert(expRoot, PairStrToPair(p), v)
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
				(&Inserter{}).ForceInsert(root, PairStrToPair(op.p), op.v)
			}

			for i, p := range test.lookup {
				var value []uint
				Lookup(root, PathFromSliceStr(p), LookupStrategyGreedy, func(leaf *Leaf) bool {
					value = leaf.AppendTo(value)
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

type countVisitor struct {
	nodes int
	leafs int
}

func (c *countVisitor) OnNode(p []PairStr, n *Node) bool { c.nodes++; return true }
func (c *countVisitor) OnLeaf(p []PairStr, l *Leaf) bool { c.leafs++; return true }

func TestTrieDeleteCleanup(t *testing.T) {
	trie := New(nil)
	path := PathFromSliceStr(pairs{{1, "a"}, {2, "b"}})

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
			trie.Insert(PathFromSliceStr(op.p), op.v)
		}

		before := listing.DumpString(trie)

		for _, del := range test.delete {
			if del.ok != trie.Delete(PathFromSliceStr(del.p), del.v) {
				t.Errorf("[%d] Delete(%v, %v) = %v; want %v", i, del.p, del.v, !del.ok, del.ok)
			}
		}
		var result []uint
		trie.ForEach(Path{}, func(p []PairStr, v uint) bool {
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

func TestTrieItemCount(t *testing.T) {
	for _, test := range []struct {
		insert []item
		query  Path
		expect int
	}{
		{
			insert: []item{
				{pairs{{0x1, "a"}, {0x2, "b"}, {0x3, "c"}}, 42},
			},
			expect: 1,
		},
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
				{pairs{{1, "a"}, {2, "b"}}, 2},
				{pairs{{1, "a"}}, 3},
				{pairs{{2, "b"}}, 4},
				{pairs{}, 5},
			},
			expect: 5,
		},
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
				{pairs{{1, "a"}, {2, "b"}}, 2},
				{pairs{{1, "a"}}, 3},
				{pairs{{2, "b"}}, 4},
				{pairs{}, 5},
			},
			query: PathFromSliceStr([]PairStr{
				{1, "a"},
			}),
			expect: 3,
		},
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
				{pairs{{1, "a"}, {2, "b"}}, 2},
				{pairs{{1, "a"}}, 3},
				{pairs{{2, "b"}}, 4},
				{pairs{}, 5},
			},
			query: PathFromSliceStr([]PairStr{
				{1, "a"}, {2, "b"},
			}),
			expect: 2,
		},
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}}, 1},
				{pairs{{1, "a"}, {2, "b"}}, 2},
				{pairs{{1, "a"}}, 3},
				{pairs{{2, "b"}}, 4},
				{pairs{}, 5},
			},
			query: PathFromSliceStr([]PairStr{
				{2, "b"},
			}),
			expect: 1,
		},
	} {
		trie := New(nil)
		for _, op := range test.insert {
			trie.Insert(PathFromSliceStr(op.p), op.v)
		}
		if act := trie.ItemCount(test.query); act != test.expect {
			t.Errorf(
				"ItemCount(%v) = %v; want %v\nTrie:\n%s",
				test.query, act, test.expect, listing.DumpString(trie),
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

func randBts(n int) [][]byte {
	return randBtsn(n, 8)
}

func randBtsn(n, m int) [][]byte {
	dup := make(map[string]bool, n)
	ret := make([][]byte, n)
	for i := 0; i < n; i++ {
		b := make([]byte, m)
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
				ret[i] = b
				break
			}
		}
	}
	return ret
}

func randStr(n int) (ret []string) {
	return randStrn(n, 8)
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

func TestTrieSelect(t *testing.T) {
	trie := New(&TrieConfig{
		NodeOrder: []uint{1, 2},
	})

	trie.Insert(PathFromMapStr(map[uint]string{1: "a"}), 0xff01)
	trie.Insert(PathFromMapStr(map[uint]string{3: "b"}), 0xff02)
	trie.Insert(PathFromMapStr(map[uint]string{3: "c"}), 0xff03)
	trie.Insert(PathFromMapStr(map[uint]string{1: "a", 3: "b"}), 0xbb01)
	trie.Insert(PathFromMapStr(map[uint]string{1: "a", 3: "c"}), 0xbb02)

	for i := 0; i < 2; i++ {
		k := "key" + strconv.FormatInt(int64(i), 16)
		trie.Insert(PathFromMapStr(map[uint]string{1: "a", 2: k, 3: "b"}), uint(i))
	}
	for i := 2; i < 4; i++ {
		k := "key" + strconv.FormatInt(int64(i), 16)
		trie.Insert(PathFromMapStr(map[uint]string{1: "a", 2: k, 3: "c"}), uint(i))
	}
	for i := 4; i < 6; i++ {
		k := "key" + strconv.FormatInt(int64(i), 16)
		trie.Insert(PathFromMapStr(map[uint]string{1: "a", 2: k, 4: "d"}), uint(i))
	}

	lookup := PathFromMapStr(map[uint]string{1: "a", 3: "c"})

	capture := NewWildcard(2)

	expStrict := map[string]uint{
		Wildcard{2: "key2"}.String(): 2,
		Wildcard{2: "key3"}.String(): 3,
		Wildcard{2: ""}.String():     0xbb02,
	}
	expGreedy := map[string]uint{
		Wildcard{2: "key2"}.String(): 2,
		Wildcard{2: "key3"}.String(): 3,
		Wildcard{2: "key4"}.String(): 4,
		Wildcard{2: "key5"}.String(): 5,
		Wildcard{2: ""}.String():     0xbb02,
		Wildcard{2: ""}.String():     0xff01,
		Wildcard{2: ""}.String():     0xff03,
	}

	actStrict := map[string]uint{}
	actGreedy := map[string]uint{}

	trie.SelectStrict(lookup, capture, func(c Wildcard, v uint) bool { actStrict[c.String()] = v; return true })
	trie.SelectGreedy(lookup, capture, func(c Wildcard, v uint) bool { actGreedy[c.String()] = v; return true })

	if !reflect.DeepEqual(actStrict, expStrict) {
		t.Errorf(
			"unexpected strict results:\n\tlookup: %#q; capture %#q;\n\t%#v;\n\twant:\n\t%#v",
			lookup.String(), capture.String(), actStrict, expStrict,
		)
	}
	if !reflect.DeepEqual(actGreedy, expGreedy) {
		t.Errorf(
			"unexpected greedy results:\n\tlookup: %#q; capture %#q;\n\t%#v;\n\twant:\n\t%#v",
			lookup.String(), capture.String(), actGreedy, expGreedy,
		)
	}
	if t.Failed() {
		fmt.Println(listing.DumpString(trie))
	}
}

func TestTrieLookupWildcard(t *testing.T) {
	trie := New(&TrieConfig{
		NodeOrder: []uint{1, 2},
	})

	trie.Insert(PathFromMapStr(map[uint]string{1: "a"}), 0xff01)
	trie.Insert(PathFromMapStr(map[uint]string{3: "b"}), 0xff02)
	trie.Insert(PathFromMapStr(map[uint]string{3: "c"}), 0xff03)
	trie.Insert(PathFromMapStr(map[uint]string{1: "a", 3: "b"}), 0xbb01)
	trie.Insert(PathFromMapStr(map[uint]string{1: "a", 3: "c"}), 0xbb02)

	for i := 0; i < 2; i++ {
		k := "key" + strconv.FormatInt(int64(i), 16)
		trie.Insert(PathFromMapStr(map[uint]string{1: "a", 2: k, 3: "b"}), uint(i))
	}
	for i := 2; i < 4; i++ {
		k := "key" + strconv.FormatInt(int64(i), 16)
		trie.Insert(PathFromMapStr(map[uint]string{1: "a", 2: k, 3: "c"}), uint(i))
	}
	for i := 4; i < 6; i++ {
		k := "key" + strconv.FormatInt(int64(i), 16)
		trie.Insert(PathFromMapStr(map[uint]string{1: "a", 2: k, 4: "d"}), uint(i))
	}

	lookup := PathFromMapStr(map[uint]string{1: "a", 3: "c"})

	capture := NewWildcard(2)

	expStrict := map[string]uint{
		Wildcard{2: "key2"}.String(): 2,
		Wildcard{2: "key3"}.String(): 3,
		Wildcard{2: ""}.String():     0xbb02,
	}
	expGreedy := map[string]uint{
		Wildcard{2: "key2"}.String(): 2,
		Wildcard{2: "key3"}.String(): 3,
		Wildcard{2: ""}.String():     0xbb02,
		Wildcard{2: ""}.String():     0xff01,
		Wildcard{2: ""}.String():     0xff03,
	}

	actStrict := map[string]uint{}
	actGreedy := map[string]uint{}

	trie.LookupWildcardStrict(lookup, capture, func(c Wildcard, v uint) bool { actStrict[c.String()] = v; return true })
	trie.LookupWildcardGreedy(lookup, capture, func(c Wildcard, v uint) bool { actGreedy[c.String()] = v; return true })

	if !reflect.DeepEqual(actStrict, expStrict) {
		t.Errorf(
			"unexpected strict results:\n\tlookup: %#q; capture %#q;\n\t%#v;\n\twant:\n\t%#v",
			lookup.String(), capture.String(), actStrict, expStrict,
		)
	}
	if !reflect.DeepEqual(actGreedy, expGreedy) {
		t.Errorf(
			"unexpected greedy results:\n\tlookup: %#q; capture %#q;\n\t%#v;\n\twant:\n\t%#v",
			lookup.String(), capture.String(), actGreedy, expGreedy,
		)
	}
	if t.Failed() {
		fmt.Println(listing.DumpString(trie))
	}
}
