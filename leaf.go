package radix

//go:generate ppgo

import (
	"fmt"
	"sync"

	"github.com/google/btree"
)

const (
	arrayLimit = 12
	degree     = 128
)

type Leaf struct {
	parent *Node
	value  string

	// dmu holds mutex for data manipulation.
	dmu sync.RWMutex
	// If leaf data is at most arrayLimit, uint sorted array is used.
	// Otherwise BTree will hold the data.
	array uintSortedArray
	btree *btree.BTree

	children *nodeArray
}

// newLeaf creates leaf with parent node.
func newLeaf(parent *Node, value string) *Leaf {
	return &Leaf{
		parent:   parent,
		value:    value,
		children: newNodeArray(),
	}
}

func (l *Leaf) Parent() *Node {
	return l.parent
}

func (l *Leaf) Value() string {
	return l.value
}

func (l *Leaf) HasChild(key uint) bool {
	return l.children.Has(key)
}

func (l *Leaf) AddChild(n *Node) {
	prev := l.children.Upsert(n)
	n.parent = l
	if prev != nil {
		panic(fmt.Sprintf("leaf already has child with key %v", n.key))
	}
}

func (l *Leaf) GetChild(key uint) *Node {
	n, _ := l.children.Get(key)
	return n
}

func (l *Leaf) GetsertChild(key uint) *Node {
	return l.children.GetsertFn(key, func() *Node { return &Node{key: key} })
}

func (l *Leaf) RemoveChild(key uint) *Node {
	prev, _ := l.children.Delete(key)
	return prev
}

func (l *Leaf) RemoveEmptyChild(key uint) (*Node, bool) {
	return l.children.DeleteCond(key, (*Node).Empty)
}

func (l *Leaf) AscendChildren(cb func(*Node) bool) (ok bool) {
	return l.children.Ascend(cb)
}

func (l *Leaf) AscendChildrenRange(a, b uint, cb func(*Node) bool) (ok bool) {
	return l.children.AscendRange(a, b, cb)
}

func (l *Leaf) GetAny(it func() (uint, bool)) (*Node, bool) {
	return l.children.GetAny(it)
}

func (l *Leaf) GetsertAny(it func() (uint, bool), add func() *Node) *Node {
	return l.children.GetsertAnyFn(it, add)
}

func (l *Leaf) Data() []uint {
	l.dmu.RLock()
	if l.btree != nil {
		ret := make([]uint, l.btree.Len())
		var i int
		l.btree.Ascend(func(x btree.Item) bool {
			ret[i] = uint(x.(btreeUint))
			i++
			return true
		})
		l.dmu.RUnlock()
		return ret
	}
	arr := l.array
	l.dmu.RUnlock()

	return arr.Copy()
}

func (l *Leaf) Empty() bool {
	if l.children.Len() > 0 {
		return false
	}
	l.dmu.RLock()
	var n int
	if l.btree != nil {
		n = l.btree.Len()
	} else {
		n = l.array.Len()
	}
	l.dmu.RUnlock()
	return n == 0
}

func (l *Leaf) Append(v uint) {
	l.dmu.Lock()
	switch {
	case l.btree != nil:
		l.btree.ReplaceOrInsert(btreeUint(v))

	case l.array.Len() == arrayLimit:
		l.btree = btree.New(degree)
		l.array.Ascend(func(v uint) bool {
			l.btree.ReplaceOrInsert(btreeUint(v))
			return true
		})
		l.btree.ReplaceOrInsert(btreeUint(v))
		l.array = l.array.Reset()

	default:
		l.array, _ = l.array.Upsert(v)
	}
	l.dmu.Unlock()
}

func (l *Leaf) Remove(v uint) (ok bool) {
	l.dmu.Lock()
	if l.btree != nil {
		ok = l.btree.Delete(btreeUint(v)) != nil
		if l.btree.Len() == 0 {
			l.btree = nil
		}
	} else {
		l.array, _, ok = l.array.Delete(v)
	}
	l.dmu.Unlock()
	return
}

func (l *Leaf) Ascend(it Iterator) (ok bool) {
	ok = true
	l.dmu.RLock()
	if l.btree != nil {
		l.btree.Ascend(func(i btree.Item) bool {
			ok = it(uint(i.(btreeUint)))
			return ok
		})
		l.dmu.RUnlock()
		return
	}
	arr := l.array
	l.dmu.RUnlock()

	arr.Ascend(func(v uint) bool {
		ok = it(v)
		return ok
	})

	return
}

func LeafInsert(l *Leaf, path Path, value uint, cb nodeIndexer) {
	for {
		if path.Len() == 0 {
			l.Append(value)
			return
		}
		// TODO(s.kamardin): use heap sort here to detect max miss factored node.
		// TODO(s.kamardin): when n.key is not in path, maybe get next element in Path
		//                   and seek children to ascend after the index of next pair

		// First try to find an existance of any path key in leaf children.
		// Due to the trie usage pattern, it is probably exists already.
		// If we do just lookup, leaf will not be locked for other goroutine lookups.
		// When we decided to insert new node to the leaf, we do the same thing, except
		// the locking leaf for lookups and other writes.
		n, ok := l.GetAny(path.AscendKeyIterator())
		if !ok {
			var insert bool
			n = l.GetsertAny(path.AscendKeyIterator(), func() *Node {
				insert = true
				n := makeTree(path, value, cb)
				n.parent = l
				return n
			})
			if insert {
				return
			}
		}
		v, ok := path.Get(n.key)
		if !ok {
			panic("inconsistent path state")
		}
		l = n.GetsertLeaf(v)
		path = path.Without(n.key)
	}
}

// Int implements the Item interface for integers.
type btreeUint uint

func (a btreeUint) Less(b btree.Item) bool {
	return a < b.(btreeUint)
}
