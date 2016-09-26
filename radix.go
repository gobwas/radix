package radix

import "fmt"

type Trie struct {
	root leaf
}

func New() *Trie {
	return &Trie{}
}

const any = "*"

func (t *Trie) Insert(p Pairs, v int) {
	t.root.insert(p, v)
}

// major searches for highest majority element in n's values.
// It applies boyer-moore voting algorithm.
func major(n *node) (*node, int, int) {
	var total int
	var counter int
	var candidate *node
	for _, l := range n.values {
		total++
		if l.node == nil {
			continue
		}
		switch {
		case counter == 0:
			candidate = l.node
			counter = 1
		case l.node.key == candidate.key && l.node.has(candidate.val):
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
		if l.node != nil && l.node.key == candidate.key && l.node.has(candidate.val) {
			counter++
		}
	}

	return candidate, counter, total
}

func siftUp(parent *leaf, key uint) {
	pn := parent.node
	nn := &node{key: key}
	for val, l := range pn.values {
		switch {
		case l.node == nil || l.node.key != key:
			lf := nn.get(any)
			ch := lf.ensureChild(pn.key)
			ch.set(val, pn.remove(val))

		case l.node.key == key:
			if len(l.data) > 0 {
				lf := nn.get(any)
				ch := lf.ensureChild(pn.key)
				clf := ch.get(val)
				clf.data = append(clf.data, l.data...)
			}

			// merge l.node.values with nn.values
			for v, lf := range l.node.values {
				nlf := nn.get(v)
				ch := nlf.ensureChild(pn.key)
				chlf := ch.get(val)
				chlf.data = lf.data
				chlf.node = lf.node
				lf.node = nil
				lf.data = nil
			}
		}
	}
	parent.node = nn
}

func compress(parent *leaf) {
	pn := parent.node
	if pn == nil {
		return
	}
	m, met, total := major(pn)
	if met > total/2 {
		siftUp(parent, m.key)
	}
}

func (t *Trie) Delete() {

}

func (t *Trie) Lookup() {

}

type node struct {
	key    uint
	values map[string]*leaf

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
}

func (n *node) get(val string) *leaf {
	l, ok := n.values[val]
	if !ok {
		l = &leaf{}
		n.set(val, l)
	}
	return l
}

func (n *node) remove(val string) *leaf {
	ret, ok := n.values[val]
	if ok {
		delete(n.values, val)
	}
	return ret
}

type leaf struct {
	data []int
	node *node
}

func (l *leaf) ensureChild(key uint) (ret *node) {
	if l.node == nil {
		ret = &node{key: key}
		l.node = ret
	} else if l.node.key == key {
		ret = l.node
	} else {
		panic(fmt.Sprintf("leaf has child %v; want %v", l.node.key, key))
	}
	return
}

func (l *leaf) insert(p Pairs, v int) {
	for len(p) > 0 {
		n := l.node
		if n == nil {
			// Create whole chain of p with v at the end.
			l.node = makeTree(p, v)
			return
		}
		w, v, ok := p.without(n.key)
		if ok {
			p = w
		} else {
			v = any
		}
		l = n.get(v)
	}
	l.data = append(l.data, v)
}

type Pair struct {
	Key   uint
	Value string
}

type Pairs []Pair

func (pairs Pairs) without(k uint) (ret Pairs, val string, ok bool) {
	for i, p := range pairs {
		if ok = p.Key == k; ok {
			n := len(pairs)
			pairs[i] = pairs[n-1]
			ret = pairs[:n-1]
			val = p.Value
			return
		}
	}
	return
}

func makeTree(p Pairs, v int) (ret *node) {
	n := len(p)
	ret = &node{
		key: p[n-1].Key,
		values: map[string]*leaf{
			p[n-1].Value: &leaf{
				data: []int{v},
			},
		},
		val: p[n-1].Value,
	}
	for i := n - 2; i >= 0; i-- {
		ret = &node{
			key: p[i].Key,
			values: map[string]*leaf{
				p[i].Value: &leaf{
					node: ret,
				},
			},
			val: p[i].Value,
		}
	}
	return
}
