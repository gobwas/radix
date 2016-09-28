package radix

import (
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
