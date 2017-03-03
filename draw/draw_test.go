package draw

import (
	"os"
	"testing"

	"github.com/gobwas/radix"
)

type item struct {
	pairs []radix.Pair
	value uint
}

func TestDraw(t *testing.T) {
	for _, test := range []struct {
		insert []item
	}{
		{
			insert: []item{
				{[]radix.Pair{{1, "a"}, {2, "b"}}, 42},
				{[]radix.Pair{{1, "a"}, {3, "c"}}, 10},
				{[]radix.Pair{{4, "foo"}}, 10},
				//{[]radix.Pair{{4, "foo"}, {5, "c"}}, 10},
				//{[]radix.Pair{{4, "foo"}, {5, "bar"}}, 10},
				//{[]radix.Pair{{4, "foo"}, {6, "d"}}, 10},
			},
		},
	} {
		t.Run("", func(t *testing.T) {
			trie := radix.New(nil)
			for _, item := range test.insert {
				trie.Insert(radix.PathFromSlice(item.pairs), item.value)
			}
			Draw(os.Stdout, trie)
		})
	}
}
