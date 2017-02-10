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
func NewLeaf(parent *Node, value string) *Leaf {
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

func (l *Leaf) GetsertChild(key uint) (node *Node, inserted bool) {
	node = l.children.GetsertFn(key, func() *Node {
		inserted = true
		return &Node{key: key}
	})
	return
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

// Inserter contains options for inserting values into the tree.
type Inserter struct {
	// IndexNode is a callback that will be called on every newly created Node.
	IndexNode func(*Node)

	// NodeOrder is an order of node keys, that should be kept during insertion.
	// That is, when we insert path {1:a;2:b;3:c} and NodeOrder is [2,3],
	// the tree will looks like 2:b -> 3:c -> 1:a.
	NodeOrder []uint
}

// Insert inserts value to the leaf that exists (or not and will be created) at
// the given path starting with the leaf as root.
//
// It first inserts/creates nodes from the Inserter's NodeOrder field.
// Then it takes first node for which there are key and value in the path.
// If at the current level there are no such nodes, it creates one with some
// key from the path.
func (c Inserter) Insert(leaf *Leaf, path Path, value uint) {
	// First we should save the fixed order of nodes.
	for _, key := range c.NodeOrder {
		if val, ok := path.Get(key); ok {
			n, inserted := leaf.GetsertChild(key)
			if inserted && c.IndexNode != nil {
				c.IndexNode(n)
			}
			leaf = n.GetsertLeaf(val)
			path = path.Without(key)
		}
	}

	for path.Len() > 0 {
		// First try to find an existance of any path key in leaf children.
		// Due to the trie usage pattern, it is probably exists already.
		// If we do just lookup, leaf will not be locked for other goroutine lookups.
		// When we decided to insert new node to the leaf, we do the same thing, except
		// the locking leaf for lookups and other writes.
		n, ok := leaf.GetAny(path.AscendKeyIterator())
		if !ok {
			var insert bool
			n = leaf.GetsertAny(path.AscendKeyIterator(), func() *Node {
				insert = true
				n := c.makeTree(path, value)
				n.parent = leaf
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
		leaf = n.GetsertLeaf(v)
		path = path.Without(n.key)
	}

	leaf.Append(value)
}

// ForceInsert inserts value to the leaf that exists (or not and will be
// created) at the given path starting with the leaf as root.
//
// Note that path is inserted as is, without any optimizations.
func (c Inserter) ForceInsert(leaf *Leaf, pairs []Pair, value uint) {
	cb := c.IndexNode
	for _, pair := range pairs {
		n, inserted := leaf.GetsertChild(pair.Key)
		if inserted && cb != nil {
			cb(n)
		}
		leaf = n.GetsertLeaf(pair.Value)
	}
	leaf.Append(value)
}

func (c Inserter) makeTree(p Path, v uint) *Node {
	last, cur, ok := p.Last()
	if !ok {
		panic("could not make tree with empty path")
	}
	cn := &Node{
		key: last.Key,
		val: last.Value,
	}
	cl := cn.GetsertLeaf(last.Value)
	cl.Append(v)

	cb := c.IndexNode
	if cb != nil {
		cb(cn)
	}

	p.Descend(cur, func(p Pair) bool {
		n := &Node{
			key: p.Key,
			val: p.Value,
		}
		l := n.GetsertLeaf(p.Value)
		l.AddChild(cn)

		if cb != nil {
			cb(cn)
		}
		cn, cl = n, l
		return true
	})
	return cn
}

// Int implements the Item interface for integers.
type btreeUint uint

func (a btreeUint) Less(b btree.Item) bool {
	return a < b.(btreeUint)
}
