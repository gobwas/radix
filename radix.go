package radix

import (
	"bytes"
	"strconv"
)

type (
	Iterator      func(uint) bool
	TraceIterator func([]PairStr, uint) bool
	PathIterator  func(Capture, uint) bool

	LeafIterator      func(*Leaf) bool
	TraceLeafIterator func([]PairStr, *Leaf) bool
	PathLeafIterator  func(Capture, *Leaf) bool
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

func (t *Trie) Insert(p Path, v uint) bool {
	return t.InsertTo(t.root, p, v)
}

func (t *Trie) At(p Path) *Leaf {
	return t.inserter.GetLeaf(t.root, p)
}

func (t *Trie) InsertTo(leaf *Leaf, p Path, v uint) bool {
	if p.Len() == 0 {
		return leaf.Append(v)
	}
	return t.inserter.Insert(leaf, p, v)
}

func (t *Trie) Delete(p Path, v uint) bool {
	return t.DeleteFrom(t.root, p, v)
}

func (t *Trie) DeleteFrom(leaf *Leaf, p Path, v uint) (ok bool) {
	Lookup(leaf, p, LookupStrategyStrict, func(l *Leaf) bool {
		if l.Remove(v) {
			ok = true
			cleanupBottomTop(l)
		}
		return true
	})
	return
}

// LookupStrict calls Lookup with trie root leaf and given query.
// If query does not contains all trie keys, use Select.
func (t *Trie) LookupStrict(query Path, it Iterator) {
	Lookup(t.root, query, LookupStrategyStrict, func(l *Leaf) bool {
		return l.Ascend(it)
	})
}

// LookupGreedy calls Lookup with trie root leaf and given query.
// If query does not contains all trie keys, use Select.
func (t *Trie) LookupGreedy(query Path, it Iterator) {
	Lookup(t.root, query, LookupStrategyGreedy, func(l *Leaf) bool {
		return l.Ascend(it)
	})
}

// SelectGreedy calls Select with trie root leaf and given query and capture.
func (t *Trie) SelectGreedy(query Path, capture Capture, it PathIterator) {
	Select(t.root, query, capture, LookupStrategyGreedy, func(captured Capture, leaf *Leaf) bool {
		return leaf.Ascend(func(val uint) bool {
			return it(captured, val)
		})
	})
}

// SelectStrict calls Select with trie root leaf and given query and capture.
func (t *Trie) SelectStrict(query Path, capture Capture, it PathIterator) {
	Select(t.root, query, capture, LookupStrategyStrict, func(captured Capture, leaf *Leaf) bool {
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

// ItemCount returns number of items on every Leaf which is reachable from
// found Leaf by a query.
func (t *Trie) ItemCount(query Path) int {
	v := ItemCountVisitor{}
	Walk(t.root, query, &v)
	return v.Count()
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
		return Dig(l, leafVisitor(func(trace []PairStr, lf *Leaf) bool {
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

type LookupStrategy int

const (
	LookupStrategyStrict LookupStrategy = iota
	LookupStrategyGreedy
)

type Capture map[uint]string

func NewCapture(keys ...uint) Capture {
	c := make(Capture, len(keys))
	for _, key := range keys {
		c[key] = ""
	}
	return c
}

func (c Capture) Copy() Capture {
	cp := make(Capture, len(c))
	for key, value := range c {
		cp[key] = value
	}
	return cp
}

func (c Capture) String() string {
	var buf bytes.Buffer

	var nonempty bool
	for key, value := range c {
		if nonempty {
			buf.WriteString(", ")
		}
		nonempty = true

		buf.WriteString(strconv.FormatUint(uint64(key), 16))
		buf.WriteString(":")
		buf.WriteByte('"')
		buf.WriteString(value)
		buf.WriteByte('"')
	}
	return buf.String()
}

// Select traverses the trie starting from given leaf.
//
// If it founds node with key, that is not present in query, it begins to
// traverse all it childs (leafs).
//
// At every traverse iteration it fills path with pairs that are not listed in
// query and are listed in capture. This path is passed as argument to the
// given iterator.
//
// Note that path (capture) passed to it is only valid until it returns.
//
// If you have query with all keys of trie, you could use Lookup,
// that is more efficient.
func Select(lf *Leaf, query Path, capture Capture, s LookupStrategy, it PathLeafIterator) bool {
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

		// Must reset capture with previous value after scanning current node.
		// That is done to prevent bugs when node with some key is present in
		// multiple places:
		//
		// -- root
		//      |-- 1
		//      |   |--a
		//      |      |--2
		//      |         |--b
		//      |-- 2
		//          |--c
		//
		// That is, when we looking up for items with {2:b} query and capturing
		// {1:""}, then we receive {1:a} for "item1" and {1:a} for "item2", but
		// we want {1:""} for item2.
		prev, set := capture[n.key]
		r := n.AscendLeafs(func(v string, leaf *Leaf) bool {
			if set {
				capture[n.key] = v
			}
			return Select(leaf, query, capture, s, it)
		})
		if set {
			// Reset capture to a previous value.
			capture[n.key] = prev
		}
		return r
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
func Lookup(lf *Leaf, query Path, s LookupStrategy, it LeafIterator) bool {
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

	handle := func(n *Node) bool {
		if v, ok := query.Get(n.key); ok {
			leaf := n.GetLeaf(v)
			if leaf != nil {
				return Lookup(n.GetsertLeaf(v), query.Without(n.key), s, it)
			}
		}
		return true
	}

	switch query.Len() {
	case 1:
		key, _ := query.FirstKey()
		if n := lf.GetChild(key); n != nil {
			return handle(n)
		}
		fallthrough
	case 0:
		return true
	}

	min, max := query.KeyRange()
	return lf.AscendChildrenRange(min, max, handle)
}

type Visitor interface {
	OnLeaf([]PairStr, *Leaf) bool
	OnNode([]PairStr, *Node) bool
}

type ItemCountVisitor struct {
	n int
}

func (v *ItemCountVisitor) Count() int {
	return v.n
}

func (v *ItemCountVisitor) OnLeaf(_ []PairStr, leaf *Leaf) bool {
	v.n += leaf.ItemCount()
	return true
}

func (v *ItemCountVisitor) OnNode(_ []PairStr, _ *Node) bool {
	return true
}

type InspectorVisitor struct {
	// WithRoot is an option to include in leafs count rooted Leaf.
	WithRoot bool

	leafs, nodes int
}

func (v *InspectorVisitor) OnLeaf(path []PairStr, _ *Leaf) bool {
	if len(path) != 0 || v.WithRoot {
		v.leafs++
	}
	return true
}

func (v *InspectorVisitor) OnNode(_ []PairStr, _ *Node) bool {
	v.nodes++
	return true
}

func (v *InspectorVisitor) Leafs() int { return v.leafs }
func (v *InspectorVisitor) Nodes() int { return v.nodes }

func Dig(leaf *Leaf, visitor Visitor) bool {
	return dig(leaf, nil, visitor)
}

func dig(leaf *Leaf, trace []PairStr, v Visitor) bool {
	if !v.OnLeaf(trace, leaf) {
		return false
	}
	return leaf.AscendChildren(func(n *Node) bool {
		if !v.OnNode(trace, n) {
			return false
		}
		for val, chLeaf := range n.values {
			if !dig(chLeaf, append(trace, PairStr{n.key, val}), v) {
				return false
			}
		}
		return true
	})
}

type fnVisitor struct {
	onLeaf func([]PairStr, *Leaf) bool
	onNode func([]PairStr, *Node) bool
}

func (f fnVisitor) OnLeaf(p []PairStr, l *Leaf) bool {
	if f.onLeaf != nil {
		return f.onLeaf(p, l)
	}
	return true
}
func (f fnVisitor) OnNode(p []PairStr, n *Node) bool {
	if f.onNode != nil {
		return f.onNode(p, n)
	}
	return true
}

func VisitorFunc(onLeaf func([]PairStr, *Leaf) bool, onNode func([]PairStr, *Node) bool) fnVisitor {
	return fnVisitor{onLeaf, onNode}
}

type nodeVisitor func([]PairStr, *Node) bool

func (self nodeVisitor) OnNode(p []PairStr, n *Node) bool {
	return self(p, n)
}
func (nodeVisitor) OnLeaf(_ []PairStr, _ *Leaf) bool { return true }

type leafVisitor func([]PairStr, *Leaf) bool

func (self leafVisitor) OnLeaf(p []PairStr, l *Leaf) bool {
	return self(p, l)
}

func (leafVisitor) OnNode(_ []PairStr, _ *Node) bool { return true }

func search(lf *Leaf, path Path) (ret []*Node) {
	min, max := path.KeyRange()
	lf.AscendChildrenRange(min, max, func(n *Node) bool {
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
				// TODO check this algrorithm with tests.
				//	case child.key == candidate.key && child.HasLeaf(candidate.val):
			case child.key == candidate.key:
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
			//if child.key == candidate.key && child.HasLeaf(candidate.val) {
			if child.key == candidate.key {
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
					nlf := nn.GetsertLeafStr(v)
					chn, _ := nlf.GetsertChild(pNode.key)
					chlf := chn.GetsertLeafStr(val)
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
