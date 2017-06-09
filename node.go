package radix

//go:generate ppgo

import "sync"

type Node struct {
	mu sync.RWMutex

	key    uint
	values map[string]*Leaf
	parent *Leaf
}

func (n *Node) Key() uint {
	return n.key
}

func (n *Node) Parent() *Leaf {
	return n.parent
}

func (n *Node) LeafCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.values)
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

func (n *Node) HasLeaf(k []byte) (ok bool) {
	n.mu.RLock()
	_, ok = n.values[string(k)]
	n.mu.RUnlock()
	return
}

func (n *Node) GetLeaf(k []byte) (ret *Leaf) {
	n.mu.RLock()
	ret = n.values[string(k)]
	n.mu.RUnlock()
	return
}

func (n *Node) GetsertLeaf(k []byte) (ret *Leaf) {
	var ok bool
	n.mu.Lock()
	ret, ok = n.values[string(k)]
	if ok {
		n.mu.Unlock()
		return
	}
	if n.values == nil {
		n.values = make(map[string]*Leaf, 1)
	}

	cp := string(k)
	ret = NewLeaf(n, cp)
	n.values[cp] = ret

	n.mu.Unlock()
	return
}

func (n *Node) GetsertLeafStr(k string) (ret *Leaf) {
	var ok bool
	n.mu.Lock()
	ret, ok = n.values[k]
	if ok {
		n.mu.Unlock()
		return
	}
	if n.values == nil {
		n.values = make(map[string]*Leaf, 1)
	}

	ret = NewLeaf(n, k)
	n.values[k] = ret

	n.mu.Unlock()
	return
}

func (n *Node) DeleteLeaf(k []byte) *Leaf {
	n.mu.Lock()
	ret, ok := n.values[string(k)]
	if ok {
		delete(n.values, string(k))
		ret.parent = nil
	}
	n.mu.Unlock()
	return ret
}

func (n *Node) DeleteEmptyLeaf(k string) (leaf *Leaf, ok bool) {
	n.mu.Lock()
	leaf, has := n.values[string(k)]
	if has && leaf.Empty() {
		delete(n.values, string(k))
		leaf.parent = nil
		ok = true
	}
	n.mu.Unlock()
	return
}

func (n *Node) Empty() (ok bool) {
	n.mu.RLock()
	ok = len(n.values) == 0
	n.mu.RUnlock()
	return
}
