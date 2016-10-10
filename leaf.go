package radix

import (
	"fmt"

	"github.com/google/btree"
)

type leaf struct {
	data     *btree.BTree
	children nodeArray
	parent   *node
}

func newLeaf(parent *node) *leaf {
	return &leaf{
		data:   btree.New(128),
		parent: parent,
	}
}

func (l *leaf) insert(path Path, value int, cb nodeIndexer) {
	for {
		if path.Len() == 0 {
			l.append(value)
			return
		}
		var has bool
		// TODO(s.kamardin): use heap sort here to detect max miss factored node.
		l.ascendChildrenRange(path.Min(), path.Max(), func(n *node) bool {
			if v, ok := path.Get(n.key); ok {
				l = n.leaf(v)
				path = path.Without(n.key)
				has = true
				return false
			}
			return true
		})
		if !has {
			l.addChild(makeTree(path, value, cb))
			return
		}
	}
}

func (l *leaf) has(key uint) bool {
	return l.children.Has(key)
}

func (l *leaf) addChild(n *node) {
	if l.children.Has(n.key) {
		panic(fmt.Sprintf("leaf already has child with key %v", n.key))
	}
	l.children, _ = l.children.Upsert(n)
	n.parent = l
}

func (l *leaf) removeChild(key uint) {
	l.children, _ = l.children.Delete(key)
}

func (l *leaf) ascendChildren(cb func(*node) bool) (ok bool) {
	return l.children.Ascend(cb)
}

func (l *leaf) ascendChildrenRange(a, b uint, cb func(*node) bool) (ok bool) {
	return l.children.AscendRange(a, b, cb)
}

func (l *leaf) getChild(key uint) (ret *node) {
	ret = l.children.Get(key)
	if ret == nil {
		ret = &node{key: key}
		l.addChild(ret)
	}
	return
}

func (l *leaf) dataToSlice() []int {
	ret := make([]int, l.data.Len())
	var i int
	l.data.Ascend(func(x btree.Item) bool {
		ret[i] = int(x.(btree.Int))
		i++
		return true
	})
	return ret
}

func (l *leaf) empty() bool {
	return l.children.Len() == 0 && l.data.Len() == 0
}

func (l *leaf) append(v int) {
	l.data.ReplaceOrInsert(btree.Int(v))
}

// todo use store
func (l *leaf) remove(v int) (ok bool) {
	return l.data.Delete(btree.Int(v)) != nil
}

func (l *leaf) iterate(it Iterator) (ok bool) {
	ok = true
	l.data.Ascend(func(i btree.Item) bool {
		ok = it(int(i.(btree.Int)))
		return ok
	})
	return
}
