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

	cmu      sync.RWMutex
	children nodeArray
}

func newLeaf(parent *Node) *Leaf {
	return &Leaf{
		data:   btree.New(degree),
		parent: parent,
	}
}

func (l *Leaf) Parent() *Node {
	return l.parent
}

func (l *Leaf) HasChild(key uint) bool {
	l.cmu.RLock()
	children := l.children
	l.cmu.RUnlock()
	return children.Has(key)
}

func (l *Leaf) AddChild(n *Node) {
	var prev *Node
	l.cmu.Lock()
	l.children, prev = l.children.Upsert(n)
	l.cmu.Unlock()
	n.parent = l
	if prev != nil {
		panic(fmt.Sprintf("leaf already has child with key %v", n.key))
	}
}

func (l *Leaf) GetChild(key uint) (ret *Node) {
	l.cmu.RLock()
	children := l.children
	l.cmu.RUnlock()
	return children.Get(key)
}

func (l *Leaf) GetsertChild(key uint) (ret *Node) {
	l.cmu.Lock()
	ret = l.children.Get(key)
	if ret == nil {
		ret = &Node{key: key}
		l.children.Upsert(ret)
	}
	l.cmu.Unlock()
	return
}

func (l *Leaf) RemoveChild(key uint) {
	l.cmu.Lock()
	l.children, _ = l.children.Delete(key)
	l.cmu.Unlock()
}

func (l *Leaf) AscendChildren(cb func(*Node) bool) (ok bool) {
	l.cmu.RLock()
	children := l.children
	l.cmu.RUnlock()
	return children.Ascend(cb)
}

func (l *Leaf) AscendChildrenRange(a, b uint, cb func(*Node) bool) (ok bool) {
	l.cmu.RLock()
	children := l.children
	l.cmu.RUnlock()
	return children.AscendRange(a, b, cb)
}

func (l *Leaf) Data() []int {
	l.dmu.RLock()
	defer l.dmu.RUnlock()

	ret := make([]int, l.data.Len())
	var i int
	l.data.Ascend(func(x btree.Item) bool {
		ret[i] = int(x.(btree.Int))
		i++
		return true
	})
	return ret
}

func (l *Leaf) Empty() bool {
	l.cmu.RLock()
	cl := l.children.Len()
	l.cmu.RUnlock()
	if cl > 0 {
		return false
	}

	l.dmu.RLock()
	dl := l.data.Len()
	l.dmu.RUnlock()

	return dl == 0
}

func (l *Leaf) Append(v int) {
	l.dmu.Lock()
	l.data.ReplaceOrInsert(btree.Int(v))
	l.dmu.Unlock()
}

// todo use store
func (l *Leaf) Remove(v int) (ok bool) {
	l.dmu.Lock()
	ok = l.data.Delete(btree.Int(v)) != nil
	l.dmu.Unlock()
	return
}

func (l *Leaf) Ascend(it Iterator) (ok bool) {
	ok = true
	l.dmu.RLock()
	l.data.Ascend(func(i btree.Item) bool {
		ok = it(int(i.(btree.Int)))
		return ok
	})
	l.dmu.RUnlock()
	return
}

func LeafInsert(l *Leaf, path Path, value int, cb nodeIndexer) {
	for {
		if path.Len() == 0 {
			l.Append(value)
			return
		}
		var has bool
		min, max := path.Min(), path.Max()
		// TODO(s.kamardin): use heap sort here to detect max miss factored node.
		l.AscendChildrenRange(min.Key, max.Key, func(n *Node) bool {
			if v, ok := path.Get(n.key); ok {
				l = n.GetsertLeaf(v)
				path = path.Without(n.key)
				has = true
				return false
			}
			return true
		})
		if !has {
			l.AddChild(makeTree(path, value, cb))
			return
		}
	}
}
