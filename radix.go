package radix

type (
	Iterator      func(uint) bool
	TraceIterator func([]Pair, uint) bool
	PathIterator  func(Path, uint) bool

	LeafIterator      func(*Leaf) bool
	TraceLeafIterator func([]Pair, *Leaf) bool
	PathLeafIterator  func(Path, *Leaf) bool
)

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

func (t *Trie) Delete(path Path, v uint) (ok bool) {
	Lookup(t.root, path, LookupStrategyStrict, func(l *Leaf) bool {
		if l.Remove(v) {
			ok = true
			cleanupBottomTop(l)
		}
		return true
	})
	return
}

// Lookup calls Lookup with trie root leaf and given query.
// If query does not contains all trie keys, use Select.
func (t *Trie) Lookup(query Path, it Iterator) {
	Lookup(t.root, query, LookupStrategyGreedy, func(l *Leaf) bool {
		return l.Ascend(it)
	})
}

// SelectGreedy calls Select with trie root leaf and given query and capture.
func (t *Trie) SelectGreedy(query, capture Path, it PathIterator) {
	Select(t.root, query, capture, LookupStrategyGreedy, func(captured Path, leaf *Leaf) bool {
		return leaf.Ascend(func(val uint) bool {
			return it(captured, val)
		})
	})
}

// SelectStrict calls Select with trie root leaf and given query and capture.
func (t *Trie) SelectStrict(query, capture Path, it PathIterator) {
	Select(t.root, query, capture, LookupStrategyStrict, func(captured Path, leaf *Leaf) bool {
		return leaf.Ascend(func(val uint) bool {
			return it(captured, val)
		})
	})
}

func (t *Trie) Root() *Leaf {
	return t.root
}

// ForEach searches all leafs by given query from root and then dig down
// calling it on every leaf. Note that trace argument of iterator call is valid
// only for a lifetime of call of iterator.
func (t *Trie) ForEach(query Path, it TraceIterator) {
	ForEach(t.root, query, it)
}

// Walk searches all leafs by given query from root and then dig down
// calling visitor methods on every leaf and node.
func (t *Trie) Walk(query Path, v Visitor) {
	Walk(t.root, query, v)
}

// SizeOf counts number of leafs and nodes of every leafs that matches query.
func (t *Trie) SizeOf(query Path) (leafs, nodes int) {
	return SizeOf(t.root, query)
}

func SizeOf(leaf *Leaf, query Path) (leafs, nodes int) {
	v := &InspectorVisitor{}
	Lookup(leaf, query, LookupStrategyStrict, func(l *Leaf) bool {
		Dig(leaf, v)
		return true
	})
	return v.Leafs(), v.Nodes()
}

func ForEach(leaf *Leaf, query Path, it TraceIterator) {
	Lookup(leaf, query, LookupStrategyStrict, func(l *Leaf) bool {
		return Dig(l, leafVisitor(func(trace []Pair, lf *Leaf) bool {
			return lf.Ascend(func(v uint) bool {
				return it(trace, v)
			})
		}))
	})
}

func Walk(leaf *Leaf, query Path, v Visitor) {
	Lookup(leaf, query, LookupStrategyStrict, func(l *Leaf) bool {
		return Dig(l, v)
	})
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

type lookupStrategy int

const (
	LookupStrategyStrict lookupStrategy = iota
	LookupStrategyGreedy
)

// Select traverses the trie starting from given leaf.
//
// If it founds node with key, that is not present in query, it begins to
// traverse all it childs (leafs).
//
// At every traverse iteration it fills path with pairs that are not listen in
// query and are listed in capture. This path is passed as argument to the
// given iterator.
//
// If you have query with all keys of trie, you could use Lookup,
// that is more efficient.
func Select(lf *Leaf, query, capture Path, s lookupStrategy, it PathLeafIterator) bool {
	switch s {
	case LookupStrategyStrict:
		if query.Len() == 0 {
			return it(capture, lf)
		}
	case LookupStrategyGreedy:
		if !it(capture, lf) {
			return false
		}
	}
	return lf.AscendChildren(func(n *Node) bool {
		// If query has filter for this node.
		if v, ok := query.Get(n.key); ok {
			if leaf := n.GetLeaf(v); leaf != nil {
				// We do not make capture.With(n.key, v) because it is already
				// exists in query. That is we fill capture only with keys and
				// values that are not exists in query.
				return Select(leaf, query.Without(n.key), capture, s, it)
			}
			// Filter this leaf cause it does not fit query.
			return true
		}

		// Must copy capture with node's key. That is done to prevent bugs when
		// node with some key is present in multiple places:
		//
		// -- root
		//      |-- 1
		//      |   |--a
		//      |      |--2
		//      |         |--b
		//      |-- 2
		//          |--c
		set := capture.Has(n.key)
		var cp Path
		if set {
			cp = capture.Copy()
		} else {
			cp = capture
		}
		return n.AscendLeafs(func(v string, leaf *Leaf) bool {
			if set {
				cp.Set(n.key, v)
			}
			return Select(leaf, query, cp, s, it)
		})
	})
}

// Lookup traverses the trie starting from given leaf.
//
// It expects query to be the full. That is, for every key that is stored in
// trie, there is a value in query. Due to the possibility of reordering in
// trie, it is possible to loose some values if query will not contain all
// keys.
//
// To search by a non-complete query, call Select, that is less efficient.
func Lookup(lf *Leaf, query Path, s lookupStrategy, it LeafIterator) bool {
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
				return Lookup(n.GetsertLeaf(v), query.Without(n.key), s, it)
			}
		}
		return true
	})
}

type Visitor interface {
	OnLeaf([]Pair, *Leaf) bool
	OnNode([]Pair, *Node) bool
}

type InspectorVisitor struct {
	// WithRoot is an option to include in leafs count rooted Leaf.
	WithRoot bool

	leafs, nodes int
}

func (v *InspectorVisitor) OnLeaf(path []Pair, _ *Leaf) bool {
	if len(path) != 0 || v.WithRoot {
		v.leafs++
	}
	return true
}

func (v *InspectorVisitor) OnNode(_ []Pair, _ *Node) bool {
	v.nodes++
	return true
}

func (v *InspectorVisitor) Leafs() int { return v.leafs }
func (v *InspectorVisitor) Nodes() int { return v.nodes }

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

type fnVisitor struct {
	onLeaf func([]Pair, *Leaf) bool
	onNode func([]Pair, *Node) bool
}

func (f fnVisitor) OnLeaf(p []Pair, l *Leaf) bool {
	if f.onLeaf != nil {
		return f.onLeaf(p, l)
	}
	return true
}
func (f fnVisitor) OnNode(p []Pair, n *Node) bool {
	if f.onNode != nil {
		return f.onNode(p, n)
	}
	return true
}

func VisitorFunc(onLeaf func([]Pair, *Leaf) bool, onNode func([]Pair, *Node) bool) fnVisitor {
	return fnVisitor{onLeaf, onNode}
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
