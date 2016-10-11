package radix

import "fmt"

type Pair struct {
	Key   uint
	Value string
}

type PathBuilder struct {
	size  int
	pairs []Pair
}

func NewPathBuilder(n int) *PathBuilder {
	return &PathBuilder{
		pairs: make([]Pair, n),
	}
}

func (p *PathBuilder) Add(k uint, v string) {
	p.pairs[p.size] = Pair{k, v}
	p.size++
}

func (p *PathBuilder) Build() (ret Path, err error) {
	ret = PathFromSlice(p.pairs)
	if p.size != len(p.pairs) {
		err = fmt.Errorf("requested builder to store %d pairs, but set only %d", len(p.pairs), p.size)
	}
	return
}

type PathCursor int

type Path struct {
	len      int
	pairs    []Pair
	excluded uint32
}

func PathFromSlice(data []Pair) (ret Path) {
	// TODO(s.kamardin) what if len(data)>32?
	// TODO(s.kamardin) check for duplicates
	ret.pairs = make([]Pair, len(data))
	copy(ret.pairs, data)
	ret.len = len(data)
	pairSort(ret.pairs, 0, len(ret.pairs))
	return
}

func PathFromMap(m map[uint]string) (ret Path) {
	ret.pairs = make([]Pair, len(m))
	var i int
	for k, v := range m {
		ret.pairs[i] = Pair{k, v}
		i++
		if i == len(ret.pairs) {
			break
		}
	}
	ret.len = i
	pairSort(ret.pairs, 0, i)
	return
}

func (p Path) Len() int { return p.len }

func (p Path) Has(k uint) bool {
	_, ok := p.has(k)
	return ok
}

func (p Path) Get(k uint) (string, bool) {
	i, ok := p.has(k)
	if !ok {
		return "", false
	}
	return p.pairs[i].Value, true
}

func (p Path) First() (Pair, PathCursor, bool) {
	for i := 0; i < len(p.pairs); i++ {
		if p.includes(i) {
			return p.pairs[i], PathCursor(i), true
		}
	}
	return Pair{}, PathCursor(-1), false
}

func (p Path) Last() (Pair, PathCursor, bool) {
	for i := len(p.pairs) - 1; i >= 0; i-- {
		if p.includes(i) {
			return p.pairs[i], PathCursor(i), true
		}
	}
	return Pair{}, PathCursor(-1), false
}

func (p Path) Ascend(cur PathCursor, cb func(Pair) bool) {
	for i := int(cur); i < len(p.pairs); i++ {
		if p.includes(i) && !cb(p.pairs[i]) {
			return
		}
	}
}

func (p Path) Descend(cur PathCursor, cb func(Pair) bool) {
	for i := int(cur) - 1; i >= 0; i-- {
		if p.includes(i) && !cb(p.pairs[i]) {
			return
		}
	}
}

func (p Path) Begin() PathCursor {
	return PathCursor(0)
}

func (p Path) End() PathCursor {
	return PathCursor(len(p.pairs))
}

func (p Path) AscendRange(a, b uint, cb func(Pair) bool) {
	i, _ := pairSearch(p.pairs, a)
	j, _ := pairSearch(p.pairs, b)
	for ; i <= j; i++ {
		if p.includes(i) && !cb(p.pairs[i]) {
			return
		}
	}
}

func (p Path) Min() (r Pair) {
	r, _, _ = p.First()
	return
}

func (p Path) Max() (r Pair) {
	r, _, _ = p.Last()
	return r
}

func (p Path) With(k uint, v string) Path {
	i, ok := pairSearch(p.pairs, k)
	if ok {
		p.include(i)
		return p
	}
	with := make([]Pair, len(p.pairs)+1)
	copy(with[:i], p.pairs[:i])
	copy(with[i+1:], p.pairs[i:])
	with[i] = Pair{k, v}
	p.pairs = with
	p.len++
	return p
}

func (p Path) Without(k uint) Path {
	if i, ok := p.has(k); ok {
		p.exclude(i)
		p.len--
	}
	return p
}

func (p Path) String() (ret string) {
	for i := 0; i < len(p.pairs); i++ {
		if p.includes(i) {
			pair := p.pairs[i]
			ret += fmt.Sprintf("%v:%s; ", pair.Key, pair.Value)
		}
	}
	return
}

func (p Path) includes(i int) bool {
	return p.excluded&(1<<uint(i)) == 0
}

func (p *Path) include(i int) {
	p.excluded &^= 1 << uint(i)
}

func (p *Path) exclude(i int) {
	p.excluded |= 1 << uint(i)
}
