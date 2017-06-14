package radix

//go:generate ppgo

import (
	"bytes"
	"fmt"
	"sync"
)

type Pair struct {
	Key   uint
	Value []byte
}

func (p Pair) Equal(b Pair) bool {
	return p.Key == b.Key && bytes.Equal(p.Value, b.Value)
}

// PairStr is like Pair but contains string in Value field, signaling that it
// is immutable and retreived directly from storage internals.
type PairStr struct {
	Key   uint
	Value string
}

var pairsPool [MaxPathSize]sync.Pool

type PathBuilder struct {
	pairs []Pair
}

func (p *PathBuilder) Reset() {
	for i := range p.pairs {
		p.pairs[i] = Pair{}
	}
	p.pairs = p.pairs[:0]
}

func GetPathBuilder(n int) *PathBuilder {
	if pb, _ := pairsPool[n-1].Get().(*PathBuilder); pb != nil {
		return pb
	}
	return NewPathBuilder(n)
}

func PutPathBuilder(pb *PathBuilder) {
	pb.Reset()
	pairsPool[cap(pb.pairs)-1].Put(pb)
}

// NewPathBuilder creates PathBuilder with n capacity of pairs.
func NewPathBuilder(n int) *PathBuilder {
	return &PathBuilder{
		pairs: make([]Pair, 0, n),
	}
}

func (p *PathBuilder) Add(k uint, v []byte) {
	p.pairs = append(p.pairs, Pair{k, v})
}

func (p *PathBuilder) Build() (ret Path) {
	return PathFromSliceBorrow(p.pairs)
}

type PathCursor int

const MaxPathSize = 32

type Path struct {
	len      int
	pairs    []Pair
	excluded uint32
}

func PathFromSliceBorrow(data []Pair) (ret Path) {
	ret.pairs = data
	ret.len = len(data)
	SortPair(ret.pairs, 0, len(ret.pairs))
	return
}

func PathFromSlice(data []Pair) (ret Path) {
	if len(data) > MaxPathSize {
		panic("max path size limit overflow")
	}
	// TODO(s.kamardin) check for duplicates
	pairs := make([]Pair, len(data))
	copy(pairs, data)
	return PathFromSliceBorrow(pairs)
}

func PathFromMap(m map[uint][]byte) (ret Path) {
	if len(m) > MaxPathSize {
		panic("max path size limit overflow")
	}
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
	SortPair(ret.pairs, 0, i)
	return
}

// PathFromSliceStrmakes Path from slice of PairStr.
// It could be used as helper, it allocates new []Pair and copies every PairStr
// to []byte.
func PathFromSliceStr(data []PairStr) (ret Path) {
	return PathFromSliceBorrow(PairStrToPair(data))
}

// PathFromMapStr makes Path from map of uint to str.
// It could be used as helper, it allocates new []Pair and copies every map
// value to []byte.
func PathFromMapStr(m map[uint]string) (ret Path) {
	return PathFromSliceBorrow(MapStrToPair(m))
}

func PairStrToPair(p []PairStr) []Pair {
	d := make([]Pair, len(p))
	for i := 0; i < len(p); i++ {
		d[i] = Pair{p[i].Key, []byte(p[i].Value)}
	}
	return d
}

func MapStrToPair(m map[uint]string) []Pair {
	d := make([]Pair, 0, len(m))
	for key, value := range m {
		d = append(d, Pair{key, []byte(value)})
	}
	return d
}

func (p Path) Len() int { return p.len }

func (p Path) Has(k uint) bool {
	_, ok := p.has(k)
	return ok
}

func (p Path) Get(k uint) ([]byte, bool) {
	i, ok := p.has(k)
	if !ok {
		return nil, false
	}
	return p.pairs[i].Value, true
}

func (p Path) First() (PathCursor, Pair, bool) { return p.Next(p.Begin()) }
func (p Path) FirstKey() (uint, bool) {
	_, pr, ok := p.First()
	return pr.Key, ok
}

func (p Path) Last() (PathCursor, Pair, bool) { return p.Prev(p.End()) }
func (p Path) LastKey() (uint, bool) {
	_, pr, ok := p.Last()
	return pr.Key, ok
}

func (p Path) Next(cur PathCursor) (PathCursor, Pair, bool) {
	for i := int(cur); i < len(p.pairs); i++ {
		if p.includes(i) {
			return PathCursor(i + 1), p.pairs[i], true
		}
	}
	return PathCursor(-1), Pair{}, false
}

func (p Path) NextKey(cur PathCursor) (PathCursor, uint, bool) {
	cur, pr, ok := p.Next(cur)
	return cur, pr.Key, ok
}

func (p Path) Prev(cur PathCursor) (PathCursor, Pair, bool) {
	for i := int(cur); i >= 0; i-- {
		if p.includes(i) {
			return PathCursor(i - 1), p.pairs[i], true
		}
	}
	return PathCursor(-1), Pair{}, false
}

func (p Path) PrevKey(cur PathCursor) (PathCursor, uint, bool) {
	cur, pr, ok := p.Prev(cur)
	return cur, pr.Key, ok
}

func (p Path) Ascend(cur PathCursor, cb func(Pair) bool) {
	for i := int(cur); i < len(p.pairs); i++ {
		if p.includes(i) && !cb(p.pairs[i]) {
			return
		}
	}
}

func (p Path) Descend(cur PathCursor, cb func(Pair) bool) {
	for i := int(cur); i >= 0; i-- {
		if p.includes(i) && !cb(p.pairs[i]) {
			return
		}
	}
}

func (p Path) Begin() PathCursor {
	return PathCursor(0)
}

func (p Path) End() PathCursor {
	return PathCursor(len(p.pairs) - 1)
}

func (p Path) AscendRange(a, b uint, cb func(Pair) bool) {
	i, _ := SearchPair(p.pairs, a)
	j, _ := SearchPair(p.pairs, b)
	for ; i <= j; i++ {
		if p.includes(i) && !cb(p.pairs[i]) {
			return
		}
	}
}

func (p Path) KeyRange() (min, max uint) {
	_, f, _ := p.First()
	_, l, _ := p.Last()
	return f.Key, l.Key
}

func (p Path) Copy() Path {
	cp := make([]Pair, len(p.pairs))
	copy(cp, p.pairs)
	p.pairs = cp
	return p
}

func (p Path) With(k uint, v []byte) Path {
	var with []Pair

	i, ok := SearchPair(p.pairs, k)
	if ok {
		p.include(i)
		if bytes.Equal(p.pairs[i].Value, v) {
			return p
		}
		with = make([]Pair, len(p.pairs))
	} else if len(p.pairs) == MaxPathSize {
		panic("path if full")
	} else {
		with = make([]Pair, len(p.pairs)+1)
	}

	copy(with[:i], p.pairs[:i])
	copy(with[i+1:], p.pairs[i:])
	with[i] = Pair{k, v}

	p.pairs = with
	p.len = len(p.pairs)

	return p
}

func (p *Path) Remove(k uint) {
	i, ok := SearchPair(p.pairs, k)
	if !ok {
		return
	}

	without := make([]Pair, len(p.pairs)-1)
	copy(without[:i], p.pairs[:i])
	copy(without[i:], p.pairs[i+1:])

	p.removeIndex(i)

	p.pairs = without
	p.len = len(p.pairs)

	return
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
			ret += fmt.Sprintf("%#x:%s; ", pair.Key, pair.Value)
		}
	}
	return
}

func (a Path) Equal(b Path) bool {
	if a.len != b.len || a.excluded != b.excluded {
		return false
	}
	for i := 0; i < len(a.pairs); i++ {
		ap, bp := a.pairs[i], b.pairs[i]
		if !ap.Equal(bp) {
			return false
		}
	}
	return true
}

func (p Path) includes(i int) bool {
	return p.excluded&(1<<uint(i)) == 0
}

func (p *Path) removeIndex(i int) {
	part := p.excluded >> uint(i)        // get part that should be moved;
	p.excluded &^= part << uint(i)       // clear bits that will be moved;
	p.excluded |= (part >> 1) << uint(i) // set bits;
}

func (p *Path) include(i int) {
	p.excluded &^= 1 << uint(i)
}

func (p *Path) exclude(i int) {
	p.excluded |= 1 << uint(i)
}
