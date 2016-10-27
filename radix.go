package radix

type Iterator func(uint) bool
type leafIterator func(*Leaf) bool

type Trie struct {
	root *Leaf
	heap *Heap
}

func New() *Trie {
	return &Trie{
		root: newLeaf(nil),
		heap: NewHeap(2, 0),
	}
}

func (t *Trie) Insert(p Path, v uint) {
	if p.Len() == 0 {
		t.root.Append(v)
		return
	}
	LeafInsert(t.root, p, v, t.indexNode)
}

func (t *Trie) Delete(path Path, v uint) (ok bool) {
	leafLookup(t.root, path, lookupStrict, func(l *Leaf) bool {
		// TODO(s.kamardin) cleanup empty leafs Without nodes
		if l.Remove(v) {
			ok = true
		}
		return true
	})
	return
}

func (t *Trie) Lookup(path Path, it Iterator) {
	leafLookup(t.root, path, lookupGreedy, func(l *Leaf) bool {
		if !l.Ascend(it) {
			return false
		}
		return true
	})
}

func (t *Trie) ForEach(path Path, it func(Path, uint) bool) {
	leafLookup(t.root, path, lookupStrict, func(l *Leaf) bool {
		return dig(l, path, func(path Path, lf *Leaf) bool {
			return lf.Ascend(func(v uint) bool {
				return it(path, v)
			})
		})
	})
}

type Visitor interface {
	VisitNode(*Node) bool
	VisitLeaf(Path, *Leaf) bool
}

func (t *Trie) Walk(p Path, v Visitor) {
	var prev *Node
	dig(t.root, p, func(path Path, lf *Leaf) bool {
		if lf.parent != nil && lf.parent != prev {
			if !v.VisitNode(lf.parent) {
				return false
			}
			prev = lf.parent
		}
		return v.VisitLeaf(path, lf)
	})
}

type lookupStrategy int

const (
	lookupStrict lookupStrategy = iota
	lookupGreedy
)

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
		v, ok := path.Get(n.key)
		if ok && n.HasLeaf(v) && !leafLookup(n.GetsertLeaf(v), path.Without(n.key), s, it) {
			return false
		}
		return true
	})
}

func dig(lf *Leaf, path Path, it func(Path, *Leaf) bool) bool {
	if !it(path, lf) {
		return false
	}
	return lf.AscendChildren(func(n *Node) bool {
		for k, lf := range n.values {
			if !dig(lf, path.With(n.key, k), it) {
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
	t.heap.Insert(n)
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
					pNode.DeleteLeaf(val)
					if pNode.Empty() {
						root.RemoveChild(pNode.key)
					}
				}
				for v, lf := range child.values {
					nlf := nn.GetsertLeaf(v)
					chn := nlf.GetsertChild(pNode.key)
					chlf := chn.GetsertLeaf(val)
					chlf.data = lf.data
					chlf.children = lf.children
					chlf.AscendChildren(func(c *Node) bool {
						c.parent = chlf
						return true
					})
					// cleanup
					lf.data = nil
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
