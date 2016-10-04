package radix

import (
	"sort"
	"testing"
)

type record struct {
	p Pairs
	v int
}

type del struct {
	record
	ok bool
}

func TestTrieInsert(t *testing.T) {
	for i, test := range []struct {
		insert Pairs
		values []int
	}{
		{
			insert: Pairs{{1, "a"}, {2, "b"}},
			values: []int{1, 2, 3, 4},
		},
	} {
		trie := New()
		for _, v := range test.values {
			trie.Insert(test.insert, v)
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
		insert []record
		lookup []Pairs
		expect []int
	}{
		{
			insert: []record{
				{Pairs{{1, "a"}, {2, "b"}}, 1},
			},
			lookup: []Pairs{
				Pairs{{1, "a"}, {2, "b"}},
				Pairs{{2, "b"}, {1, "a"}},
			},
			expect: []int{1},
		},
		{
			insert: []record{
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
		insert []record
		delete []del
		expect []int
	}{
		{
			insert: []record{
				{Pairs{{1, "a"}, {2, "b"}}, 1},
			},
			delete: []del{
				{record{Pairs{{1, "a"}, {2, "b"}}, 1}, true},
			},
			expect: []int{},
		},
		{
			insert: []record{
				{Pairs{{1, "a"}, {2, "b"}}, 1},
				{Pairs{{1, "a"}, {2, "b"}}, 2},
				{Pairs{{1, "a"}}, 3},
				{Pairs{{2, "b"}}, 4},
				{Pairs{}, 5},
			},
			delete: []del{
				{record{Pairs{{1, "a"}, {2, "b"}}, 1}, true},
				{record{Pairs{{1, "a"}}, 3}, true},
				{record{Pairs{{1, "a"}}, 4}, false},
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
