package radix

//go:generate ppgo

import (
	"fmt"
	"sync"
)

type Node struct {
	mu sync.RWMutex

	key    uint
	values map[string]*Leaf
	parent *Leaf

	// first set value
	val string
}

func (n *Node) Key() uint {
	return n.key
}

func (n *Node) Parent() *Leaf {
	return n.parent
}

func (n *Node) AscendLeafs(it func(string, *Leaf) bool) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for k, l := range n.values {
		if !it(k, l) {
			return false
		}
	}
	return true
}

func (n *Node) HasLeaf(k string) (ok bool) {
	n.mu.RLock()
	_, ok = n.values[k]
	n.mu.RUnlock()
	return
}

func (n *Node) GetLeaf(k string) (ret *Leaf) {
	n.mu.RLock()
	ret = n.values[k]
	n.mu.RUnlock()
	return
}

func (n *Node) InsertLeaf(k string, l *Leaf) {
	var has bool
	n.mu.Lock()
	if n.values == nil {
		n.values = make(map[string]*Leaf)
	} else {
		_, has = n.values[k]
	}
	n.values[k] = l
	n.mu.Unlock()
	l.parent = n
	if has {
		panic(fmt.Sprintf("node %v is already has %v", n.key, k))
	}
}

func (n *Node) GetsertLeaf(k string) (ret *Leaf) {
	var ok bool
	n.mu.Lock()
	ret, ok = n.values[k]
	if ok {
		n.mu.Unlock()
		return
	}
	if n.values == nil {
		n.values = make(map[string]*Leaf)
	}
	ret = newLeaf(n)
	n.values[k] = ret
	n.mu.Unlock()
	return
}

func (n *Node) DeleteLeaf(k string) *Leaf {
	n.mu.Lock()
	ret, ok := n.values[k]
	if ok {
		delete(n.values, k)
		ret.parent = nil
	}
	n.mu.Unlock()
	return ret
}

func (n *Node) Empty() (ok bool) {
	n.mu.RLock()
	ok = len(n.values) == 0
	n.mu.RUnlock()
	return
}
