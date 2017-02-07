package radix

type Iterator func(uint) bool
type TraceIterator func(Path, uint) bool
type leafIterator func(*Leaf) bool
type TraceLeafIterator func(Path, *Leaf) bool

type Trie struct {
	root *Leaf
	//heap *Heap
}

func New() *Trie {
	return &Trie{
		root: newLeaf(nil, ""),
		//heap: NewHeap(2, 0),
	}
}

func (t *Trie) Insert(p Path, v uint) {
	if p.Len() == 0 {
		t.root.Append(v)
		return
	}
	LeafInsert(t.root, p, v, t.indexNode)
}

func cleanupBottomTop(leaf *Leaf) {
	var (
		n  *Node
		ok bool
	)
	for leaf.Empty() {
		if n = leaf.parent; n == nil {
			return
		}
		if _, ok = n.DeleteEmptyLeaf(leaf.Value()); !ok {
			return
		}
		if !n.Empty() || n.parent == nil {
			return
		}
		if _, ok = n.parent.RemoveEmptyChild(n.Key()); !ok {
			return
		}
		leaf = n.parent
	}
}

func (t *Trie) Delete(path Path, v uint) (ok bool) {
	leafLookup(t.root, path, lookupStrict, func(l *Leaf) bool {
		if l.Remove(v) {
			ok = true
			cleanupBottomTop(l)
		}
		return true
	})
	return
}

func (t *Trie) Lookup(search Path, it Iterator) {
	leafLookup(t.root, search, lookupGreedy, func(l *Leaf) bool {
		return l.Ascend(it)
	})
}

func (t *Trie) TraceLookup(search Path, it TraceIterator) {
	leafLookupTrace(t.root, search, Path{}, lookupGreedy, func(trace Path, leaf *Leaf) bool {
		return leaf.Ascend(func(val uint) bool {
			return it(trace, val)
		})
	})
}

func (t *Trie) ForEach(search Path, it TraceIterator) {
	leafLookup(t.root, search, lookupStrict, func(l *Leaf) bool {
		return dig(l, search, nil, func(trace Path, lf *Leaf) bool {
			return lf.Ascend(func(v uint) bool {
				return it(trace, v)
			})
		})
	})
}

type Visitor interface {
	VisitNode(*Node) bool
	VisitLeaf(Path, *Leaf) bool
}

func (t *Trie) Walk(p Path, v Visitor) {
	dig(
		t.root, p,
		func(path Path, n *Node) bool {
			return v.VisitNode(n)
		},
		func(path Path, lf *Leaf) bool {
			return v.VisitLeaf(path, lf)
		},
	)
}

type lookupStrategy int

const (
	lookupStrict lookupStrategy = iota
	lookupGreedy
)

func leafLookupTrace(lf *Leaf, search, trace Path, s lookupStrategy, it TraceLeafIterator) bool {
	switch s {
	case lookupStrict:
		if search.Len() == 0 {
			return it(trace, lf)
		}
	case lookupGreedy:
		if !it(trace, lf) {
			return false
		}
	}
	min, max := search.Min(), search.Max()
	return lf.AscendChildrenRange(min.Key, max.Key, func(n *Node) bool {
		if v, ok := search.Get(n.key); ok {
			if leaf := n.GetLeaf(v); leaf != nil {
				return leafLookupTrace(leaf,
					search.Without(n.key), trace.With(n.key, v),
					s, it,
				)
			}
		}
		return true
	})
}

func leafLookup(lf *Leaf, path Path, s lookupStrategy, it leafIterator) bool {
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
	min, max := path.Min(), path.Max()
	return lf.AscendChildrenRange(min.Key, max.Key, func(n *Node) bool {
		if v, ok := path.Get(n.key); ok {
			leaf := n.GetLeaf(v)
			if leaf != nil {
				return leafLookup(n.GetsertLeaf(v), path.Without(n.key), s, it)
			}
		}
		return true
	})
}

func dig(lf *Leaf, path Path, onNode func(Path, *Node) bool, onLeaf func(Path, *Leaf) bool) bool {
	if onLeaf != nil && !onLeaf(path, lf) {
		return false
	}
	return lf.AscendChildren(func(n *Node) bool {
		if onNode != nil && !onNode(path, n) {
			return false
		}
		for k, lf := range n.values {
			if !dig(lf, path.With(n.key, k), onNode, onLeaf) {
				return false
			}
		}
		return true
	})
}

func search(lf *Leaf, path Path) (ret []*Node) {
	min, max := path.Min(), path.Max()
	lf.AscendChildrenRange(min.Key, max.Key, func(n *Node) bool {
		if v, ok := path.Get(n.key); ok {
			if path.Len() == 1 {
				ret = append(ret, n)
			}
			if n.HasLeaf(v) {
				ret = append(ret, search(n.GetsertLeaf(v), path.Without(n.key))...)
			}
		}
		return true
	})
	return
}

func SearchNode(t *Trie, path Path) *Node {
	if n := search(t.root, path); len(n) > 0 {
		return n[0]
	}
	return nil
}

func (t *Trie) indexNode(n *Node) {
	//TODO(s.kamardin): use heap from ppgo here and sift up less hit nodes up
	//t.heap.Insert(n)
}

type nodeIndexer func(n *Node)

// major searches for highest majority element in node values.
// It applies boyer-moore voting algorithm.
func major(n *Node) (*Node, int, int) {
	var total int
	var counter int
	var candidate *Node
	for _, l := range n.values {
		l.AscendChildren(func(child *Node) bool {
			total++
			switch {
			case counter == 0:
				candidate = child
				counter = 1
			case child.key == candidate.key && child.HasLeaf(candidate.val):
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
		l.AscendChildren(func(child *Node) bool {
			if child.key == candidate.key && child.HasLeaf(candidate.val) {
				counter++
			}
			return true
		})
	}
	return candidate, counter, total
}

// SiftUp pulls up given node in the tree.
// Its like rotate left in the tree when the node is on the right side. =)
func SiftUp(n *Node) *Node {
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
	nn := &Node{
		key:    n.key,
		parent: root,
	}
	for val, l := range pNode.values {
		l.AscendChildren(func(child *Node) bool {
			switch {
			//	case child.key != n.key:
			//		lf := nn.leaf(any)
			//		ch := lf.getChild(pNode.key)
			//		chlf := ch.leaf(val)
			//		chlf.addChild(child)
			//ch.set(val, pNode.remove(val)) // todo could copy pNode's val, to be like immutable

			case child.key == n.key:
				l.RemoveChild(child.key)
				if l.Empty() {
					pNode.DeleteEmptyLeaf(val)
					if pNode.Empty() {
						root.RemoveEmptyChild(pNode.key)
					}
				}
				for v, lf := range child.values {
					nlf := nn.GetsertLeaf(v)
					chn := nlf.GetsertChild(pNode.key)
					chlf := chn.GetsertLeaf(val)
					chlf.btree = lf.btree
					chlf.children = lf.children
					chlf.AscendChildren(func(c *Node) bool {
						c.parent = chlf
						return true
					})
					// cleanup
					lf.btree = nil
					lf.children = nil
					lf.parent = nil
				}
			}
			return true
		})
	}
	root.AddChild(nn)
	return nn
}

func compress(n *Node) {
	m, met, total := major(n)
	if met > total/2 {
		SiftUp(m)
	}
}

func makeTree(p Path, v uint, cb nodeIndexer) *Node {
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
	cb(cn)

	p.Descend(cur, func(p Pair) bool {
		n := &Node{
			key: p.Key,
			val: p.Value,
		}
		l := n.GetsertLeaf(p.Value)
		l.AddChild(cn)

		cb(n)
		cn, cl = n, l
		return true
	})
	return cn
}
