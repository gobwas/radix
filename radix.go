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
func checkNode(parent *leaf) {
	n := parent.child
	if n == nil {
		return
	}

	var counter int
	var candidate *node
	for k, l := range n.values {
		if l.node == nil || l.node.values[l.node.val] != nil {
			continue
		}
		if len(l.node.values) > 1 {
			// todo
		}
		switch {
		case counter == 0:
			candiate = l.node
			counter = 1
		case candidate.key == l.node.key && candidate.val == l.node.val:
			counter++
		default:
			counter--
		}
	}
	var n int
	counter = 0
	for k, l := range n.values {
		if l.node == nil {
			continue
		}
		n++
		if candidate.key == l.node.key && candidate.val == l.node.val {
			counter++
		}
	}
	// We have found more than n/2 duplicates.
	if counter > n/2 { // could optimize!
		nn = &node{
			key: candidate.key,
			val: candidate.val,
			values: map[string]*leaf{
				candidate.val: &leaf{},
			},
		}
		for k, l := range n.values {
			switch {
			case l.child == nil:
				lf := nn.get(any)
				lf.data = append(l.data, l.data...)

			case l.child.key == nn.key:
				// merge l.child.values with nn.values
				// append each l.child.values[n]...

			default: // skip
			}
		}
		parent.child = nn
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

func (n *node) get(k string) *leaf {
	l, ok := n.values[k]
	if !ok {
		l = &leaf{}
		n.values[v] = l
	}
	return l
}

type leaf struct {
	data  []int
	child *node
}

func (l *leaf) insert(p Pairs, v int) {
	for len(p) > 0 {
		n := l.child
		if n == nil {
			// Create whole chain of p with v at the end.
			l.child = makeTree(p, v)
			return
		}
		w, v, ok := p.without(n.key)
		if ok {
			p = w
		} else {
			v = any
		}
		l = n.get(v)
		//if l, ok = n.values[v]; !ok {
		//	l = &leaf{}
		//	n.values[v] = l
		//}
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
					child: ret,
				},
			},
			val: p[i].Value,
		}
	}
	return
}
