package radix

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

type pairs []PairStr
type item struct {
	p pairs
	v uint
}

func TestCapture(t *testing.T) {
	for i, test := range []struct {
		insert   []item
		lookup   []pairs
		capture  []uint
		strategy LookupStrategy
		greedy   bool
		exp      map[uint]Wildcard
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
			greedy:   true,
			exp: map[uint]Wildcard{
				1: Wildcard{2: "b"},
			},
		},
		{
			insert: []item{
				{pairs{{1, "a"}, {2, "b"}, {3, "c"}, {4, "d"}}, 1},
			},
			lookup: []pairs{
				pairs{{1, "a"}, {3, "c"}},
				pairs{{3, "c"}, {1, "a"}},
			},
			capture:  []uint{2},
			strategy: LookupStrategyGreedy,
			greedy:   false,
			exp:      map[uint]Wildcard{},
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
			greedy:   true,
			exp: map[uint]Wildcard{
				1: Wildcard{2: "b"},
				2: Wildcard{2: "b"},
				3: Wildcard{2: "b"},
				4: Wildcard{2: "b"},
				5: Wildcard{2: ""},
			},
		},
	} {
		label := fmt.Sprintf("#%d", i)

		t.Run(label, func(t *testing.T) {
			root := NewLeaf(nil, "root")
			for _, op := range test.insert {
				(&Inserter{}).ForceInsert(root, PairStrToPair(op.p), op.v)
			}

			wildcard := NewWildcard(test.capture...)

			for _, p := range test.lookup {
				var trace = map[uint]Wildcard{}
				capture(root, PathFromSliceStr(p), wildcard, test.greedy, test.strategy, func(c Wildcard, l *Leaf) bool {
					for _, v := range l.AppendTo(nil) {
						trace[v] = c.Copy()
					}
					return true
				})

				for v, trc := range trace {
					if exp := test.exp[v]; !reflect.DeepEqual(trc, exp) {
						var buf bytes.Buffer
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
