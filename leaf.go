package radix

import (
	"fmt"
	"sync"

	"github.com/google/btree"
)

const degree = 128

type Leaf struct {
	parent *Node

	dmu  sync.RWMutex
	data *btree.BTree

	children *nodeArray
}

func newLeaf(parent *Node) *Leaf {
	return &Leaf{
		data:     btree.New(degree),
		children: newNodeArray(),
		parent:   parent,
	}
}

func (l *Leaf) Parent() *Node {
	return l.parent
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
	defer l.dmu.RUnlock()

	ret := make([]uint, l.data.Len())
	var i int
	l.data.Ascend(func(x btree.Item) bool {
		ret[i] = uint(x.(btreeUint))
		i++
		return true
	})
	return ret
}

func (l *Leaf) Empty() bool {
	if l.children.Len() > 0 {
		return false
	}
	l.dmu.RLock()
	dl := l.data.Len()
	l.dmu.RUnlock()
	return dl == 0
}

func (l *Leaf) Append(v uint) {
	l.dmu.Lock()
	l.data.ReplaceOrInsert(btreeUint(v))
	l.dmu.Unlock()
}

// todo use store
func (l *Leaf) Remove(v uint) (ok bool) {
	l.dmu.Lock()
	ok = l.data.Delete(btreeUint(v)) != nil
	l.dmu.Unlock()
	return
}

func (l *Leaf) Ascend(it Iterator) (ok bool) {
	ok = true
	l.dmu.RLock()
	l.data.Ascend(func(i btree.Item) bool {
		ok = it(uint(i.(btreeUint)))
		return ok
	})
	l.dmu.RUnlock()
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
				return makeTree(path, value, cb)
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
