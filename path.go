package radix

type Pair struct {
	Key   uint
	Value string
}

type Path struct {
	size    int
	pairs   [32]Pair
	exclude uint32
}

func PathFromSlice(data ...Pair) (ret Path) {
	copy(ret.pairs[:], data)
	if len(data) > len(ret.pairs) {
		ret.size = len(ret.pairs)
	} else {
		ret.size = len(data)
	}
	doSort(ret.pairs[:], 0, ret.size)
	return
}

func PathFromMap(m map[uint]string) (ret Path) {
	var i int
	for k, v := range m {
		ret.pairs[i] = Pair{k, v}
		i++
		if i == len(ret.pairs) {
			break
		}
	}
	ret.size = i
	doSort(ret.pairs[:], 0, i)
	return
}

func (p Path) Len() int { return p.size }

func (p Path) Has(k uint) bool {
	_, ok := p.has(k)
	return ok
}

func (p Path) has(k uint) (i int, ok bool) {
	i = bsearch(p.pairs[:p.size], k)
	ok = i > -1 && p.exclude&1<<uint(i) == 0
	return
}

func (p Path) At(i int) Pair {
	if i >= p.size {
		panic("index out of range")
	}
	return p.pairs[i]
}

func (p Path) Without(k uint) (Path, string, bool) {
	if i, ok := p.has(k); ok {
		p.exclude |= 1 << uint(i)
		p.size--
		return p, p.pairs[i].Value, ok
	}
	return p, "", false
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
			r = m - 1
		}
	}
	return -1
}
