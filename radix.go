package radix

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
	t.root.insert(p, v, t.indexNode)
}

func (t *Trie) Delete(path Path, v int) (ok bool) {
	leafLookup(t.root, path, lookupStrict, func(l *leaf) bool {
		// TODO(s.kamardin) cleanup empty leafs Without nodes
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

func dig(path Path, lf *leaf, it func(Path, int) bool) bool {
	ok := lf.iterate(func(v int) bool {
		return it(path, v)
	})
	if !ok {
		return false
	}
	lf.ascendChildren(func(n *node) bool {
		for k, lf := range n.values {
			if !dig(path.With(n.key, k), lf, it) {
				ok = false
				return false
			}
		}
		return true
	})
	return ok
}

func (t *Trie) ForEach(it func(Path, int) bool) { dig(Path{}, t.root, it) }

type lookupStrategy int

const (
	lookupStrict lookupStrategy = iota
	lookupGreedy
)

func leafLookup(lf *leaf, path Path, s lookupStrategy, it leafIterator) bool {
	switch s {
	case lookupStrict:
		if path.Len() == 0 {
			return it(lf)
		}
	case lookupGreedy:
		if !it(lf) {
			return false
		}
	}
	return lf.ascendChildrenRange(path.Min(), path.Max(), func(n *node) bool {
		v, ok := path.Get(n.key)
		if ok && n.has(v) && !leafLookup(n.leaf(v), path.Without(n.key), s, it) {
			return false
		}
		return true
	})
}

func search(lf *leaf, path Path) (ret []*node) {
	lf.ascendChildrenRange(path.Min(), path.Max(), func(n *node) bool {
		if v, ok := path.Get(n.key); ok {
			if path.Len() == 1 {
				ret = append(ret, n)
			}
			if n.has(v) {
				ret = append(ret, search(n.leaf(v), path.Without(n.key))...)
			}
		}
		return true
	})
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
		l.ascendChildren(func(child *node) bool {
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
			return true
		})
	}
	if candidate == nil {
		return nil, -1, total
	}
	counter = 0
	for _, l := range n.values {
		l.ascendChildren(func(child *node) bool {
			if child.key == candidate.key && child.has(candidate.val) {
				counter++
			}
			return true
		})
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
		l.ascendChildren(func(child *node) bool {
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
					chlf.ascendChildren(func(c *node) bool {
						c.parent = chlf
						return true
					})
					// cleanup
					lf.data = nil
					lf.children = nodeArray{}
					lf.parent = nil
				}
			}
			return true
		})
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

	p.Descend(cur, func(p Pair) bool {
		n := &node{
			key: p.Key,
			val: p.Value,
		}
		l := n.leaf(p.Value)
		l.addChild(cn)

		cb(n)
		cn, cl = n, l
		return true
	})
	return cn
}
