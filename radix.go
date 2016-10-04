package radix

import "fmt"

const any = "*"

type Iterator func(int) bool
type leafIterator func(*leaf) bool

type Trie struct {
	root  leaf
	nodes map[uint]*node
}

func New() *Trie {
	return &Trie{
		nodes: make(map[uint]*node),
	}
}

func (t *Trie) Insert(p Pairs, v int) {
	if p == nil || p.Len() == 0 {
		t.root.data = append(t.root.data, v)
		return
	}
	if t.root.child == nil {
		t.root.child = makeTree(p, v, t.indexNode)
		t.root.child.parent = &t.root
		return
	}
	t.root.child.insert(p, v, t.indexNode)
}

func (t *Trie) Delete(path Pairs, v int) (ok bool) {
	if path == nil || path.Len() == 0 {
		return t.root.remove(v)
	}
	if t.root.child == nil {
		return
	}
	strictLookup(t.root.child, path, func(l *leaf) bool {
		// todo use storage interface
		if l.remove(v) {
			ok = true
		}
		return true
	})
	return
}

func (t *Trie) Lookup(path Pairs, it Iterator) {
	if !t.root.iterate(it) || t.root.child == nil {
		return
	}
	greedyLookup(t.root.child, path, func(l *leaf) bool {
		if !l.iterate(it) {
			return false
		}
		return true
	})
}

type lookupFn func(*node, Pairs, leafIterator) bool

func checkLeaf(lf *leaf, path Pairs, it leafIterator, lookup lookupFn) bool {
	if !it(lf) {
		return false
	}
	if lf.child != nil && !lookup(lf.child, path, it) {
		return false
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

func searchNode(t *Trie, path Pairs) *node {
	if t.root.child == nil {
		return nil
	}
	n := t.root.child
	var v string
	var ok bool
	for path.Len() > 0 && n != nil {
		path, v, ok = path.without(n.key)
		if !ok || !n.has(v) {
			return nil
		}
		n = n.leaf(v).child
	}
	return n
}

func (t *Trie) indexNode(n *node) {
	t.nodes[n.key] = n
}

type nodeIndexer func(n *node)

// major searches for highest majority element in node values.
// It applies boyer-moore voting algorithm.
func major(n *node) (*node, int, int) {
	var total int
	var counter int
	var candidate *node
	for _, l := range n.values {
		total++
		if l.child == nil {
			continue
		}
		switch {
		case counter == 0:
			candidate = l.child
			counter = 1
		case l.child.key == candidate.key && l.child.has(candidate.val):
			counter++
		default:
			counter--
		}
	}
	if candidate == nil {
		return nil, -1, total
	}
	counter = 0
	for _, l := range n.values {
		if l.child != nil && l.child.key == candidate.key && l.child.has(candidate.val) {
			counter++
		}
	}
	return candidate, counter, total
}

// siftUp pulls up given node in the tree.
// Its like rotate left in the tree when the node is on the right side. =)
func siftUp(n *node) *node {
	pLeaf := n.parent     // parent leaf
	pNode := pLeaf.parent // parent node
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
		switch {
		case l.child == nil || l.child.key != n.key:
			lf := nn.leaf(any)
			ch := lf.ensureChild(pNode.key)
			ch.set(val, pNode.remove(val)) // todo could copy pNode's val, to be like immutable

		case l.child.key == n.key:
			if len(l.data) > 0 {
				lf := nn.leaf(any)
				chn := lf.ensureChild(pNode.key)
				clf := chn.leaf(val)
				clf.data = append(clf.data, l.data...)
			}

			// merge l.child.values with nn.values
			for v, lf := range l.child.values {
				nlf := nn.leaf(v)
				chn := nlf.ensureChild(pNode.key)

				chlf := chn.leaf(val)
				chlf.data = lf.data
				chlf.child = lf.child
				if chlf.child != nil {
					chlf.child.parent = chlf
				}

				// cleanup
				//lf.data = nil
				//lf.child = nil
				//lf.parent = nil
			}
		}
	}
	nn.parent = root
	root.child = nn
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
	for {
		pw, v, ok := path.without(n.key)
		if ok {
			path = pw
		} else {
			v = any
		}
		l := n.leaf(v)
		if path.Len() == 0 {
			l.data = append(l.data, value)
			return
		}
		if n = l.child; n == nil {
			// Create whole chain of p with v at the end.
			l.child = makeTree(path, value, cb)
			l.child.parent = l
			return
		}
	}
}

type leaf struct {
	data   []int
	child  *node
	parent *node
}

func (l *leaf) ensureChild(key uint) (ret *node) {
	if l.child == nil {
		ret = &node{
			key:    key,
			parent: l,
		}
		l.child = ret
	} else if l.child.key == key {
		ret = l.child
	} else {
		panic(fmt.Sprintf("leaf has child %v; want %v", l.child.key, key))
	}
	return
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
	// Make the last one node.
	cl := &leaf{
		data: []int{v},
	}
	cn := &node{
		key: p[n-1].Key,
		values: map[string]*leaf{
			p[n-1].Value: cl,
		},
		val: p[n-1].Value,
	}
	cl.parent = cn
	cb(cn)
	for i := n - 2; i >= 0; i-- {
		l := &leaf{
			child: cn,
		}
		n := &node{
			key: p[i].Key,
			values: map[string]*leaf{
				p[i].Value: l,
			},
			val: p[i].Value,
		}
		l.parent = n
		cn.parent = l
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
