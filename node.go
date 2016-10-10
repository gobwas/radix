package radix

import "fmt"

type node struct {
	key    uint
	values map[string]*leaf
	parent *leaf

	// first set value
	val string
}

func (n *node) has(k string) (ok bool) {
	_, ok = n.values[k]
	return
}

func (n *node) empty() bool {
	return len(n.values) == 0
}

func (n *node) set(val string, l *leaf) {
	if n.values == nil {
		n.values = make(map[string]*leaf)
	} else if _, ok := n.values[val]; ok {
		panic(fmt.Sprintf("branch %v is already exists on node %v", val, n.key))
	}
	n.values[val] = l
	l.parent = n
}

func (n *node) leaf(val string) *leaf {
	if val == "" {
		panic("empty leaf value")
	}
	l, ok := n.values[val]
	if !ok {
		l = newLeaf(n)
		n.set(val, l)
	}
	return l
}

func (n *node) remove(val string) *leaf {
	ret, ok := n.values[val]
	if ok {
		delete(n.values, val)
		ret.parent = nil
	}
	return ret
}
