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
		trie.ForEach(Path{}, func(p []PairStr, v uint) bool {
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
		exp      map[uint]Capture
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
			exp: map[uint]Capture{
				1: Capture{2: "b"},
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
			exp: map[uint]Capture{
				1: Capture{2: "b"},
				2: Capture{2: "b"},
				3: Capture{2: "b"},
				4: Capture{2: "b"},
				5: Capture{2: ""},
			},
		},
	} {
		label := fmt.Sprintf("#%d", i)

		t.Run(label, func(t *testing.T) {
			root := NewLeaf(nil, "root")
			for _, op := range test.insert {
				(&Inserter{}).ForceInsert(root, PairStrToPair(op.p), op.v)
			}

			capture := NewCapture(test.capture...)

			for _, p := range test.lookup {
				var trace = map[uint]Capture{}
				Select(root, PathFromSliceStr(p), capture, LookupStrategyGreedy, func(c Capture, l *Leaf) bool {
					for _, v := range l.Data() {
						trace[v] = c.Copy()
					}
					return true
				})

				for v, trc := range trace {
					if exp := test.exp[v]; !reflect.DeepEqual(trc, exp) {
						var buf bytes.Buffer
						listing.DumpLeaf(&buf, root)

						t.Errorf(
							"[%d] Select(%v) returned %#q with capture %#q; want %#q;\nTrie:\n%s\n",
							i, p, v, trace, exp, buf.String(),
						)
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
		NodeOrder: []uint{1, 2, 3},
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

	lookup := PathFromMapStr(map[uint]string{1: "a", 3: "c"})

	capture := Capture{2: ""}

	expStrict := map[string]uint{
		Capture{2: "key2"}.String(): 2,
		Capture{2: "key3"}.String(): 3,
		Capture{2: ""}.String():     0xbb02,
	}
	expGreedy := map[string]uint{
		Capture{2: "key2"}.String(): 2,
		Capture{2: "key3"}.String(): 3,
		Capture{2: ""}.String():     0xbb02,
		Capture{2: ""}.String():     0xff01,
		Capture{2: ""}.String():     0xff03,
	}

	actStrict := map[string]uint{}
	actGreedy := map[string]uint{}

	fmt.Println(listing.DumpString(trie))

	trie.SelectStrict(lookup, capture, func(c Capture, v uint) bool { actStrict[c.String()] = v; return true })
	trie.SelectGreedy(lookup, capture, func(c Capture, v uint) bool { actGreedy[c.String()] = v; return true })

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
}
