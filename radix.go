package radix

import "fmt"

const any = "*"

type Iterator func(int) bool
type leafIterator func(*leaf) bool

type Trie struct {
	root leaf
	heap *Heap
}

func New() *Trie {
	return &Trie{
		heap: NewHeap(2, 0),
	}
}

func (t *Trie) Insert(p Pairs, v int) {
	if p == nil || p.Len() == 0 {
		t.root.append(v)
		return
	}
	var n *node
	for _, c := range t.root.children {
		if p.has(c.key) && (n == nil || t.heap.Less(n, c)) {
			n = c
		}
	}
	if n != nil {
		n.insert(p, v, t.indexNode)
		return
	}
	n = makeTree(p, v, t.indexNode)
	t.root.addChild(n)
}

func (t *Trie) Delete(path Pairs, v int) (ok bool) {
	if path == nil || path.Len() == 0 {
		return t.root.remove(v)
	}
	for _, child := range t.root.children {
		strictLookup(child, path, func(l *leaf) bool {
			// todo use storage interface
			if l.remove(v) {
				ok = true
			}
			return true
		})
	}
	return
}

func (t *Trie) Lookup(path Pairs, it Iterator) {
	if !t.root.iterate(it) {
		return
	}
	for _, child := range t.root.children {
		greedyLookup(child, path, func(l *leaf) bool {
			if !l.iterate(it) {
				return false
			}
			return true
		})
	}
}

type lookupFn func(*node, Pairs, leafIterator) bool

func checkLeaf(lf *leaf, path Pairs, it leafIterator, lookup lookupFn) bool {
	if !it(lf) {
		return false
	}
	for _, child := range lf.children {
		if !lookup(child, path, it) {
			return false
		}
	}
	return true
}

// greedyLookup searches values in greedy manner.
// It first searches all strict equal leafs.
// Then in searches all 'any' valued leafs.
// If node has key k, and it is not present in path, then it will
// dig in all leafs of node.
func greedyLookup(n *node, path Pairs, it leafIterator) (ret bool) {
	pw, v, ok := path.without(n.key)
	if !ok {
		for _, lf := range n.values {
			if !checkLeaf(lf, pw, it, greedyLookup) {
				return false
			}
		}
		return true
	}
	if n.has(v) && !checkLeaf(n.leaf(v), pw, it, greedyLookup) {
		return false
	}
	if n.has(any) && !checkLeaf(n.leaf(any), pw, it, greedyLookup) {
		return false
	}
	return true
}

func strictLookup(n *node, path Pairs, it leafIterator) bool {
	pw, v, ok := path.without(n.key)
	if !ok {
		for _, lf := range n.values {
			if !checkLeaf(lf, pw, it, strictLookup) {
				return false
			}
		}
		return true
	}
	if n.has(v) && !checkLeaf(n.leaf(v), pw, it, strictLookup) {
		return false
	}
	return true
}

func search(lf *leaf, path Pairs) (ret []*node) {
	for _, child := range lf.children {
		p, v, ok := path.without(child.key)
		if ok && child.has(v) {
			if p.Len() == 0 {
				ret = append(ret, child)
			} else {
				ret = append(ret, search(child.leaf(v), p)...)
			}
		}
	}
	return
}

func searchNode(t *Trie, path Pairs) *node {
	if n := search(&t.root, path); len(n) > 0 {
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
			l.removeChild(child.key)
			if l.empty() {
				pNode.remove(val)
			}

			switch {
			case child.key != n.key:

				lf := nn.leaf(any)
				ch := lf.getChild(pNode.key)
				chlf := ch.leaf(val)
				chlf.addChild(child)
				//ch.set(val, pNode.remove(val)) // todo could copy pNode's val, to be like immutable

			case child.key == n.key:
				if len(l.data) > 0 {
					lf := nn.leaf(any)
					chn := lf.getChild(pNode.key)
					clf := chn.leaf(val)
					clf.append(l.data...)
				}

				// merge l.child.values with nn.values
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
	l, ok := n.values[val]
	if !ok {
		l = &leaf{parent: n}
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

func (n *node) insert(path Pairs, value int, cb nodeIndexer) {
insertion:
	for {
		pw, v, ok := path.without(n.key)
		if ok {
			path = pw
		} else {
			v = any
		}
		l := n.leaf(v)
		if path.Len() == 0 {
			l.append(value)
			return
		}
		for _, child := range l.children {
			if path.has(child.key) {
				n = child
				continue insertion
			}
		}
		// Create whole chain of p with v at the end.
		l.addChild(makeTree(path, value, cb))
		return
	}
}

type leaf struct {
	data     []int
	children map[uint]*node
	parent   *node
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

func (l *leaf) empty() bool {
	return len(l.children) == 0
}

func (l *leaf) append(v ...int) {
	l.data = append(l.data, v...)
}

// todo use store
func (l *leaf) remove(v int) (ok bool) {
	for i, e := range l.data {
		if ok = e == v; ok {
			n := len(l.data)
			d := make([]int, n-1)
			copy(d[:i], l.data[:i])
			copy(d[i:], l.data[i+1:])
			l.data = d
			return
		}
	}
	return
}

func (l *leaf) iterate(it Iterator) bool {
	for _, v := range l.data {
		if !it(v) {
			return false
		}
	}
	return true
}

func makeTree(p Pairs, v int, cb nodeIndexer) *node {
	n := p.Len()
	cn := &node{
		key: p[n-1].Key,
		val: p[n-1].Value,
	}
	cl := cn.leaf(p[n-1].Value)
	cl.append(v)
	cb(cn)
	for i := n - 2; i >= 0; i-- {
		n := &node{
			key: p[i].Key,
			val: p[i].Value,
		}
		l := n.leaf(p[i].Value)
		l.addChild(cn)

		cb(n)
		cn, cl = n, l
	}
	return cn
}

type Pair struct {
	Key   uint
	Value string
}

type Pairs []Pair

func (p Pairs) Len() int { return len(p) }

func (pairs Pairs) has(k uint) bool {
	for _, p := range pairs {
		if p.Key == k {
			return true
		}
	}
	return false
}

func (pairs Pairs) without(k uint) (ret Pairs, val string, ok bool) {
	for i, p := range pairs {
		if ok = p.Key == k; ok {
			n := len(pairs)
			ret = make(Pairs, n-1)
			copy(ret[:i], pairs[:i])
			copy(ret[i:], pairs[i+1:])
			val = p.Value
			return
		}
	}
	ret = pairs
	return
}
