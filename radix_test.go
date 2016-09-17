package radix

import (
	"fmt"
	"os"
	"testing"
)

type insert struct {
	p Pairs
	v int
}

func TestTrieInsert(t *testing.T) {
	for i, test := range []struct {
		insert   []insert
		compress bool
	}{
		{
			insert: []insert{
				{
					p: Pairs{},
					v: 0,
				},
				{
					p: Pairs{
						{1, "foo"},
						{2, "bar"},
					},
					v: 1,
				},
				{
					p: Pairs{
						{1, "foo"},
						{2, "bar"},
					},
					v: 2,
				},
				{
					p: Pairs{
						{1, "foo"},
						{2, "baz"},
					},
					v: 3,
				},
				{
					p: Pairs{
						{3, "goo"},
					},
					v: 4,
				},
				{
					p: Pairs{
						{1, "foo"},
					},
					v: 5,
				},
				{
					p: Pairs{
						{1, "foo"},
						{2, "bar"},
						{3, "goo"},
					},
					v: 6,
				},
			},
		},
		{
			insert: []insert{
				// many emails
				{Pairs{{1, "a@example.com"}, {2, "domain.org"}}, 1},
				{Pairs{{1, "b@example.com"}, {2, "domain.org"}}, 2},
				{Pairs{{1, "c@example.com"}, {2, "domain.org"}}, 3},
				{Pairs{{1, "d@example.com"}, {2, "domain.org"}}, 4},
				{Pairs{{1, "d@example.com"}, {2, "example.com"}}, 5},
			},
			compress: true,
		},
	} {
		trie := New()
		for _, op := range test.insert {
			trie.Insert(op.p, op.v)
		}
		fmt.Fprint(os.Stdout, "\n\n")
		graphviz(os.Stdout, fmt.Sprintf("test#%d", i), trie)
		fmt.Fprint(os.Stdout, "\n\n")

		if test.compress {
			fmt.Fprintf(os.Stdout, "compressing...\n\n")
			compress(&trie.root)
			graphviz(os.Stdout, fmt.Sprintf("test#%d", i), trie)
			fmt.Fprint(os.Stdout, "\n\n")
		}
	}
}

//func sizeOfTrie(l *leaf) (s int) {
//	s = unsafe.SizeOf(l)
//	for {
//
//	}
//}
