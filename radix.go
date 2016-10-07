package radix

import (
	"fmt"
	"github.com/google/btree"
	//"github.com/gobwas/array"
)

const any = "*"

type Iterator func(int) bool
type leafIterator func(*leaf) bool

type Trie struct {
	root *leaf
	heap *Heap
}

func New() *Trie {
	return &Trie{
		root: newLeaf(nil),
		heap: NewHeap(2, 0),
	}
}

func (t *Trie) Insert(p Path, v int) {
	if p.Len() == 0 {
		t.root.append(v)
		return
	}
	// We want to find node with maximum miss factor.
	var n *node
	var lv string
	for _, c := range t.root.children {
		v, has := p.Get(c.key)
		if has && (n == nil || t.heap.Less(n, c)) {
			lv = v
			n = c
		}
	}
	if n != nil {
		n.leaf(lv).insert(p.Without(n.key), v, t.indexNode)
		return
	}
	n = makeTree(p, v, t.indexNode)
	t.root.addChild(n)
}

func (t *Trie) Delete(path Path, v int) (ok bool) {
	leafLookup(t.root, path, lookupStrict, func(l *leaf) bool {
		// todo use storage interface
		// todo cleanup empty leafs Without nodes
		if l.remove(v) {
			ok = true
		}
		return true
	})
	return
}

func (t *Trie) Lookup(path Path, it Iterator) {
	leafLookup(t.root, path, lookupGreedy, func(l *leaf) bool {
		if !l.iterate(it) {
			return false
		}
		return true
	})
}

type lookupStrategy int

const (
	lookupStrict lookupStrategy = iota
	lookupGreedy
)

func leafLookup(lf *leaf, path Path, s lookupStrategy, it leafIterator) bool {
	if !it(lf) {
		return false
	}
	switch s {
	case lookupStrict:
		for _, child := range lf.children {
			if !strictNodeLookup(child, path, it) {
				return false
			}
		}
	case lookupGreedy:
		for _, child := range lf.children {
			if !greedyNodeLookup(child, path, it) {
				return false
			}
		}
	}
	return true
}

func strictNodeLookup(n *node, path Path, it leafIterator) (ret bool) {
	if !path.Has(n.key) {
		return true
	}
	for _, lf := range n.values {
		if !leafLookup(lf, path.Without(n.key), lookupStrict, it) {
			return false
		}
	}
	return true
}

// nodeLookup searches values in greedy manner.
// It iterates over data of leafs, that strict equal to path.
// If node has key k, and it is not present in path, then it will
// dig in all leafs of node.
func greedyNodeLookup(n *node, path Path, it leafIterator) (ret bool) {
	v, ok := path.Get(n.key)
	if !ok {
		for _, lf := range n.values {
			if !leafLookup(lf, path, lookupGreedy, it) {
				return false
			}
		}
		return true
	}
	if n.has(v) && !leafLookup(n.leaf(v), path.Without(n.key), lookupGreedy, it) {
		return false
	}
	return true
}

func search(lf *leaf, path Path) (ret []*node) {
	for _, child := range lf.children {
		v, ok := path.Get(child.key)
		if !ok {
			continue
		}
		if path.Len() == 1 {
			ret = append(ret, child)
		}
		if child.has(v) {
			ret = append(ret, search(child.leaf(v), path.Without(child.key))...)
		}
	}
	return
}

func searchNode(t *Trie, path Path) *node {
	if n := search(t.root, path); len(n) > 0 {
		return n[0]
	}
	return nil
}

func (t *Trie) indexNode(n *node) {
	t.heap.Insert(n)
}

type nodeIndexer func(n *node)

// major searches for highest majority element in node values.
// It applies boyer-moore voting algorithm.
func major(n *node) (*node, int, int) {
	var total int
	var counter int
	var candidate *node
	for _, l := range n.values {
		for _, child := range l.children {
			total++
			switch {
			case counter == 0:
				candidate = child
				counter = 1
			case child.key == candidate.key && child.has(candidate.val):
				counter++
			default:
				counter--
			}
		}
	}
	if candidate == nil {
		return nil, -1, total
	}
	counter = 0
	for _, l := range n.values {
		for _, child := range l.children {
			if child.key == candidate.key && child.has(candidate.val) {
				counter++
			}
		}
	}
	return candidate, counter, total
}

// siftUp pulls up given node in the tree.
// Its like rotate left in the tree when the node is on the right side. =)
func siftUp(n *node) *node {
	pLeaf := n.parent     // parent leaf
	pNode := pLeaf.parent // parent node
	if pNode == nil {
		return n
	}
	root := pNode.parent
	if root == nil { // could not perform rotation
		return n
	}
	// twin clone of n
	nn := &node{
		key:    n.key,
		parent: root,
	}
	for val, l := range pNode.values {
		for _, child := range l.children {
			switch {
			//	case child.key != n.key:
			//		lf := nn.leaf(any)
			//		ch := lf.getChild(pNode.key)
			//		chlf := ch.leaf(val)
			//		chlf.addChild(child)
			//ch.set(val, pNode.remove(val)) // todo could copy pNode's val, to be like immutable

			case child.key == n.key:
				l.removeChild(child.key)
				if l.empty() {
					pNode.remove(val)
					if pNode.empty() {
						root.removeChild(pNode.key)
					}
				}
				for v, lf := range child.values {
					nlf := nn.leaf(v)
					chn := nlf.getChild(pNode.key)
					chlf := chn.leaf(val)
					chlf.data = lf.data
					chlf.children = lf.children
					for _, c := range chlf.children {
						c.parent = chlf
					}
					// cleanup
					lf.data = nil
					lf.children = nil
					lf.parent = nil
				}
			}
		}
	}
	root.addChild(nn)
	return nn
}

func compress(n *node) {
	m, met, total := major(n)
	if met > total/2 {
		siftUp(m)
	}
}

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

type leaf struct {
	data     *btree.BTree
	children map[uint]*node
	parent   *node
}

func newLeaf(parent *node) *leaf {
	return &leaf{
		data:   btree.New(128),
		parent: parent,
	}
}

func (l *leaf) insert(path Path, value int, cb nodeIndexer) {
insertion:
	for {
		if path.Len() == 0 {
			l.append(value)
			return
		}
		for _, child := range l.children {
			if v, ok := path.Get(child.key); ok {
				l = child.leaf(v)
				path = path.Without(child.key)
				continue insertion
			}
		}
		// Create whole chain of p with v at the end.
		l.addChild(makeTree(path, value, cb))
		return
	}
}
func (l *leaf) has(key uint) bool {
	_, ok := l.children[key]
	return ok
}

func (l *leaf) addChild(n *node) {
	if _, has := l.children[n.key]; has {
		panic(fmt.Sprintf("leaf already has child with key %v", n.key))
	}
	if l.children == nil {
		l.children = make(map[uint]*node)
	}
	l.children[n.key] = n
	n.parent = l
}

func (l *leaf) removeChild(key uint) {
	delete(l.children, key)
}

func (l *leaf) getChild(key uint) (ret *node) {
	var ok bool
	ret, ok = l.children[key]
	if !ok {
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
	return len(l.children) == 0 && l.data.Len() == 0
}

func (l *leaf) append(v int) {
	l.data.ReplaceOrInsert(btree.Int(v))
}

// todo use store
func (l *leaf) remove(v int) (ok bool) {
	return l.data.Delete(btree.Int(v)) != nil
}

func (l *leaf) iterate(it Iterator) (ok bool) {
	l.data.Ascend(func(i btree.Item) bool {
		ok = it(int(i.(btree.Int)))
		return ok
	})
	return
}

func makeTree(p Path, v int, cb nodeIndexer) *node {
	last, cur, ok := p.Last()
	if !ok {
		panic("could not make tree with empty path")
	}
	cn := &node{
		key: last.Key,
		val: last.Value,
	}
	cl := cn.leaf(last.Value)
	cl.append(v)
	cb(cn)

	p.Descend(cur, func(p Pair) {
		n := &node{
			key: p.Key,
			val: p.Value,
		}
		l := n.leaf(p.Value)
		l.addChild(cn)

		cb(n)
		cn, cl = n, l
	})
	return cn
}
