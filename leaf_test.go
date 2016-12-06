package radix

import (
	"fmt"
	"reflect"
	"testing"
)

func TestLeafAscend(t *testing.T) {
	for _, test := range []struct {
		append []uint
		remove []uint
		expect []uint
	}{
		{
			append: []uint{0, 1, 2, 3},
			remove: []uint{2},
			expect: []uint{0, 1, 3},
		},
		{
			append: []uint{0, 1, 2, 3},
			remove: []uint{0, 1, 2, 3},
			expect: []uint{},
		},
		{
			append: []uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			remove: nil,
			expect: []uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
		{
			append: []uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			remove: []uint{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			expect: []uint{},
		},
	} {
		t.Run(fmt.Sprintf("append:%d remove:%d", len(test.append), len(test.remove)), func(t *testing.T) {
			leaf := newLeaf(nil)
			for _, v := range test.append {
				leaf.Append(v)
			}
			for _, v := range test.remove {
				leaf.Remove(v)
			}
			data := make([]uint, 0, len(test.expect))
			leaf.Ascend(func(v uint) bool {
				data = append(data, v)
				return true
			})
			if !reflect.DeepEqual(data, test.expect) {
				t.Errorf("result data is: %v; want %v", data, test.expect)
			}
		})
	}
}

func TestLeafAppend(t *testing.T) {
	for _, test := range []struct {
		append int
		remove int
		btree  int
		arr    int
	}{
		{
			append: 1,
			remove: 0,
			btree:  0,
			arr:    1,
		},
		{
			append: 1,
			remove: 1,
			btree:  0,
			arr:    0,
		},
		{
			append: 12,
			remove: 0,
			btree:  0,
			arr:    12,
		},
		{
			append: arrayLimit + 1,
			remove: 0,
			btree:  arrayLimit + 1,
			arr:    0,
		},
		{
			append: arrayLimit + 1,
			remove: 1,
			btree:  arrayLimit,
			arr:    0,
		},
		{
			append: arrayLimit + 1,
			remove: arrayLimit + 1,
			btree:  0,
			arr:    0,
		},
		{
			append: arrayLimit,
			remove: arrayLimit,
			btree:  0,
			arr:    0,
		},
	} {
		t.Run(fmt.Sprintf("append:%d remove:%d", test.append, test.remove), func(t *testing.T) {
			leaf := newLeaf(nil)
			for i := 0; i < test.append; i++ {
				leaf.Append(uint(i))
			}
			for i := 0; i < test.remove; i++ {
				leaf.Remove(uint(i))
			}
			if leaf.btree == nil {
				if test.btree > 0 {
					t.Errorf("btree is nil")
				}
			} else {
				if test.btree == 0 {
					t.Errorf("btree is not nil")
				} else if n := leaf.btree.Len(); n != test.btree {
					t.Errorf("btree len is %d; want %d", n, test.btree)
				}
			}
			if n := leaf.array.Len(); n != test.arr {
				t.Errorf("array len is %d; want %d", n, test.btree)
			}
		})
	}
}
