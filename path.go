package radix

import "fmt"

type Pair struct {
	Key   uint
	Value string
}

type Path struct {
	size     int
	len      int
	pairs    []Pair
	excluded uint32
}

func PathFromSlice(data ...Pair) (ret Path) {
	// TODO(s.kamardin) what if len(data)>32?
	ret.pairs = make([]Pair, len(data))
	copy(ret.pairs, data)
	if len(data) > len(ret.pairs) {
		ret.size = len(ret.pairs)
	} else {
		ret.size = len(data)
	}
	ret.len = ret.size
	doSort(ret.pairs, 0, ret.size)
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
	ret.size = i
	ret.len = i
	doSort(ret.pairs, 0, i)
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

func (p Path) Last() (Pair, int, bool) {
	for i := p.size - 1; i >= 0; i-- {
		if p.includes(i) {
			return p.pairs[i], i, true
		}
	}
	return Pair{}, -1, false
}

func (p Path) Descend(cur int, cb func(Pair)) {
	for i := cur - 1; i >= 0; i-- {
		if p.includes(i) {
			cb(p.pairs[i])
		}
	}
}

func (p Path) Without(k uint) Path {
	if i, ok := p.has(k); ok {
		p.exclude(i)
		p.len--
	}
	return p
}

func (p Path) String() (ret string) {
	for i := 0; i < p.size; i++ {
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

func (p Path) exclude(i int) {
	p.excluded |= 1 << uint(i)
}

func (p Path) has(k uint) (i int, ok bool) {
	i = bsearch(p.pairs[:p.size], k)
	ok = i > -1 && p.includes(i)
	return
}

func partition(data []Pair, l, r int) int {
	x := data[l] // pivot
	j := l
	for i := l + 1; i < r; i++ {
		if data[i].Key <= x.Key {
			j++
			data[j], data[i] = data[i], data[j]
		}
	}
	data[j], data[l] = data[l], data[j]
	return j
}

func quickSort(data []Pair, lo, hi int) {
	if lo >= hi {
		return
	}
	p := partition(data, lo, hi)
	quickSort(data, lo, p)
	quickSort(data, p+1, hi)
}

func insertionSort(data []Pair, l, r int) {
	for i := l + 1; i < r; i++ {
		for j := i; j > l && data[j-1].Key > data[j].Key; j-- {
			data[j], data[j-1] = data[j-1], data[j]
		}
	}
}

func doSort(data []Pair, l, r int) {
	if r-l > 12 {
		quickSort(data, l, r)
	} else {
		insertionSort(data, l, r)
	}
}

func bsearch(data []Pair, key uint) int {
	l := 0
	r := len(data)
	for l < r {
		m := l + (r-l)/2
		switch {
		case data[m].Key == key:
			return m
		case data[m].Key < key:
			l = m + 1
		case data[m].Key > key:
			r = m
		}
	}
	return -1
}
