package radix

type Iterator func(uint) bool
type PathIterator func([]Pair, uint) bool
type TraceIterator func(Path, uint) bool
type TraceLeafIterator func(Path, *Leaf) bool

type leafIterator func(*Leaf) bool

type TrieConfig struct {
	NodeOrder []uint
}

type Trie struct {
	inserter *Inserter
	root     *Leaf
	//heap *Heap
}

func New(config *TrieConfig) *Trie {
	t := &Trie{
		inserter: &Inserter{},
		root:     NewLeaf(nil, ""),
		//heap: NewHeap(2, 0),
	}

	t.inserter.IndexNode = t.indexNode
	if config != nil {
		t.inserter.NodeOrder = config.NodeOrder
	}

	return t
}

func (t *Trie) Insert(p Path, v uint) {
	if p.Len() == 0 {
		t.root.Append(v)
		return
	}
	t.inserter.Insert(t.root, p, v)
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
	LookupComplete(t.root, path, LookupStrategyStrict, func(l *Leaf) bool {
		if l.Remove(v) {
			ok = true
			cleanupBottomTop(l)
		}
		return true
	})
	return
}

func (t *Trie) Lookup(search Path, it Iterator) {
	LookupComplete(t.root, search, LookupStrategyGreedy, func(l *Leaf) bool {
		return l.Ascend(it)
	})
}

func (t *Trie) TraceLookup(search Path, it TraceIterator) {
	LookupPartial(t.root, search, Path{}, LookupStrategyGreedy, func(trace Path, leaf *Leaf) bool {
		return leaf.Ascend(func(val uint) bool {
			return it(trace, val)
		})
	})
}

// trace is valid only for a lifetime of call of iterator.
func (t *Trie) ForEach(query Path, it PathIterator) {
	ForEach(t.root, query, it)
}

func (t *Trie) Walk(query Path, v Visitor) {
	LookupComplete(t.root, query, LookupStrategyStrict, func(l *Leaf) bool {
		return Dig(l, v)
	})
}

func ForEach(leaf *Leaf, query Path, it PathIterator) {
	LookupComplete(leaf, query, LookupStrategyStrict, func(l *Leaf) bool {
		return Dig(l, leafVisitor(func(trace []Pair, lf *Leaf) bool {
			return lf.Ascend(func(v uint) bool {
				return it(trace, v)
			})
		}))
	})
}

type lookupStrategy int

const (
	LookupStrategyStrict lookupStrategy = iota
	LookupStrategyGreedy
)

func leafLookupTrace(lf *Leaf, search, trace Path, s lookupStrategy, it TraceLeafIterator) bool {
	switch s {
	case LookupStrategyStrict:
		if search.Len() == 0 {
			return it(trace, lf)
		}
	case LookupStrategyGreedy:
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

// leafLookupPartial travaerses the trie starting from given leaf.
//
// If it founds node with key, that is not present in query, it begins to
// traverse all it childs (leafs).
//
// At every traverse iteration it fills whole path from starting leaf.
// This path is passed aas trace argument to the given iterator.
//
// If you have query with all keys of trie, you could use LookupComplete,
// that is more efficient.
func LookupPartial(lf *Leaf, query, trace Path, s lookupStrategy, it TraceLeafIterator) bool {
	switch s {
	case LookupStrategyStrict:
		if query.Len() == 0 {
			return it(trace, lf)
		}
	case LookupStrategyGreedy:
		if !it(trace, lf) {
			return false
		}
	}
	return lf.AscendChildren(func(n *Node) bool {
		// If query has filter for this node.
		if v, ok := query.Get(n.key); ok {
			if leaf := n.GetLeaf(v); leaf != nil {
				return LookupPartial(leaf, query.Without(n.key), trace.With(n.key, v), s, it)
			}
			// Filter this leaf cause it does not fit query.
			return true
		}
		return n.AscendLeafs(func(v string, leaf *Leaf) bool {
			return LookupPartial(leaf, query, trace.With(n.key, v), s, it)
		})
	})
}

// LookupComplete traverses the trie starting from given leaf.
//
// It expects query to be the full. That is, for every key that is stored in trie,
// there is a value in query.
// Due to the possibility of reordering in trie, it is possible to loose some values if
// query will not contain all keys.
//
// To search by a non complete query, call LookupPartial, that is less efficient.
func LookupComplete(lf *Leaf, query Path, s lookupStrategy, it leafIterator) bool {
	switch s {
	case LookupStrategyStrict:
		if query.Len() == 0 {
			return it(lf)
		}
	case LookupStrategyGreedy:
		if !it(lf) {
			return false
		}
	}
	min, max := query.Min(), query.Max()
	return lf.AscendChildrenRange(min.Key, max.Key, func(n *Node) bool {
		if v, ok := query.Get(n.key); ok {
			leaf := n.GetLeaf(v)
			if leaf != nil {
				return LookupComplete(n.GetsertLeaf(v), query.Without(n.key), s, it)
			}
		}
		return true
	})
}

type Visitor interface {
	OnNode([]Pair, *Node) bool
	OnLeaf([]Pair, *Leaf) bool
}

func Dig(leaf *Leaf, visitor Visitor) bool {
	return dig(leaf, nil, visitor)
}

func dig(leaf *Leaf, trace []Pair, v Visitor) bool {
	if !v.OnLeaf(trace, leaf) {
		return false
	}
	return leaf.AscendChildren(func(n *Node) bool {
		if !v.OnNode(trace, n) {
			return false
		}
		for val, chLeaf := range n.values {
			if !dig(chLeaf, append(trace, Pair{n.key, val}), v) {
				return false
			}
		}
		return true
	})
}

type nodeVisitor func([]Pair, *Node) bool

func (self nodeVisitor) OnNode(p []Pair, n *Node) bool {
	return self(p, n)
}
func (nodeVisitor) OnLeaf(_ []Pair, _ *Leaf) bool { return true }

type leafVisitor func([]Pair, *Leaf) bool

func (self leafVisitor) OnLeaf(p []Pair, l *Leaf) bool {
	return self(p, l)
}

func (leafVisitor) OnNode(_ []Pair, _ *Node) bool { return true }

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
					chn, _ := nlf.GetsertChild(pNode.key)
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
