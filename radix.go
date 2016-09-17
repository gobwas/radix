package radix

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

// checks by boyer-moore voting algorithm.
func compress(parent *leaf) {
	n := parent.node
	if n == nil {
		return
	}

	var counter int
	var candidate *node
	for _, l := range n.values {
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
		return
	}

	var total int
	counter = 0
	for _, l := range n.values {
		total++
		if l.node == nil {
			continue
		}
		if l.node.key == candidate.key && l.node.has(candidate.val) {
			counter++
		}
	}

	// We have found more than n/2 duplicates.
	if counter > total/2 { // could optimize!
		nn := &node{
			key:    candidate.key,
			val:    candidate.val,
			values: map[string]*leaf{},
		}

		for k, l := range n.values {
			switch {
			case l.node == nil || l.node.key != nn.key:
				lf := nn.get(any)
				ch := lf.ensureChild(n.key)
				ch.addChain(k, n.cutChain(k))

			case l.node.key == nn.key:
				if len(l.data) > 0 {
					lf := nn.get(any)
					ch := lf.ensureChild(n.key)
					clf := ch.get(k)
					clf.data = append(clf.data, l.data...)
				}

				// merge l.node.values with nn.values
				for _, lf := range l.node.values {
					nlf := nn.get(candidate.val)
					ch := nlf.ensureChild(n.key)
					chlf := ch.get(k)
					chlf.data = lf.data
					chlf.node = lf.node
					lf.node = nil
					lf.data = nil
				}
			}
		}

		parent.node = nn
	}
}

func swap() {

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

func (n *node) get(v string) *leaf {
	l, ok := n.values[v]
	if !ok {
		if n.values == nil {
			n.values = make(map[string]*leaf)
		}
		l = &leaf{}
		n.values[v] = l
	}
	return l
}

func (n *node) cutChain(val string) (ret *leaf) {
	ret, ok := n.values[val]
	if ok {
		delete(n.values, val)
	}
	return
}

func (n *node) addChain(val string, m *leaf) {
	_, ok := n.values[val]
	if !ok {
		n.values[val] = m
		return
	}
	panic("chain already exists")
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
		panic("could not return child")
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
