package radix

type Trie struct {
	root leaf
}

func New() *Trie {
	return &Trie{}
}

const any = "*"

func (t *Trie) Insert(p Pairs, v int) {
	l := &t.root
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
		if l, ok = n.values[v]; !ok {
			l = &leaf{}
			n.values[v] = l
		}
	}
	l.data = append(l.data, v)
}

func (t *Trie) Delete() {

}

func (t *Trie) Lookup() {

}

type node struct {
	key    uint
	values map[string]*leaf
}

type leaf struct {
	data  []int
	child *node
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
	}
	for i := n - 2; i >= 0; i-- {
		ret = &node{
			key: p[i].Key,
			values: map[string]*leaf{
				p[i].Value: &leaf{
					child: ret,
				},
			},
		}
	}
	return
}
