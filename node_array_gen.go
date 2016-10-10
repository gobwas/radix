package radix

func nodePartition(data []*node, l, r int) int {
	// Inlined partition algorithm
	var j int
	{
		// Let x be a pivot
		x := data[l]
		j = l
		for i := l + 1; i < r; i++ {
			if data[i].key <= x.key {
				j++
				data[j], data[i] = data[i], data[j]
			}
		}
		data[j], data[l] = data[l], data[j]
	}
	return j
}

func nodeQuickSort(data []*node, lo, hi int) {
	if lo >= hi {
		return
	}
	// Inlined partition algorithm
	var p int
	{
		// Let x be a pivot
		x := data[lo]
		p = lo
		for i := lo + 1; i < hi; i++ {
			if data[i].key <= x.key {
				p++
				data[p], data[i] = data[i], data[p]
			}
		}
		data[p], data[lo] = data[lo], data[p]
	}
	nodeQuickSort(data, lo, p)
	nodeQuickSort(data, p+1, hi)
}

func nodeInsertionSort(data []*node, l, r int) {
	// Inlined insertion sort
	for i := l + 1; i < r; i++ {
		for j := i; j > l && data[j-1].key > data[j].key; j-- {
			data[j], data[j-1] = data[j-1], data[j]
		}
	}
}

func nodeSort(data []*node, l, r int) {
	if r-l > 12 {
		nodeQuickSort(data, l, r)
		return
	}
	// Inlined insertion sort
	for i := l + 1; i < r; i++ {
		for j := i; j > l && data[j-1].key > data[j].key; j-- {
			data[j], data[j-1] = data[j-1], data[j]
		}
	}
}

func nodeSearch(data []*node, key uint) (int, bool) {
	// Inlined binary search
	var ok bool
	i := len(data)
	{
		l := 0
		for !ok && l < i {
			m := l + (i-l)/2
			switch {
			case data[m].key == key:
				ok = true
				i = m
			case data[m].key < key:
				l = m + 1
			case data[m].key > key:
				i = m
			}
		}
	}
	return i, ok
}

type nodeArray struct {
	data []*node
}

func (a nodeArray) Has(x uint) bool {
	// Inlined binary search
	var ok bool
	i := len(a.data)
	{
		l := 0
		for !ok && l < i {
			m := l + (i-l)/2
			switch {
			case a.data[m].key == x:
				ok = true
				i = m
			case a.data[m].key < x:
				l = m + 1
			case a.data[m].key > x:
				i = m
			}
		}
	}
	return ok
}

func (a nodeArray) Get(x uint) *node {
	// Inlined binary search
	var ok bool
	i := len(a.data)
	{
		l := 0
		for !ok && l < i {
			m := l + (i-l)/2
			switch {
			case a.data[m].key == x:
				ok = true
				i = m
			case a.data[m].key < x:
				l = m + 1
			case a.data[m].key > x:
				i = m
			}
		}
	}
	if !ok {
		return nil
	}
	return a.data[i]
}

func (a nodeArray) Upsert(x *node) (cp nodeArray, prev *node) {
	var with []*node
	// Inlined binary search
	var has bool
	i := len(a.data)
	{
		l := 0
		for !has && l < i {
			m := l + (i-l)/2
			switch {
			case a.data[m].key == x.key:
				has = true
				i = m
			case a.data[m].key < x.key:
				l = m + 1
			case a.data[m].key > x.key:
				i = m
			}
		}
	}
	if has {
		with = make([]*node, len(a.data))
		copy(with, a.data)
		a.data[i], prev = x, a.data[i]
	} else {
		with = make([]*node, len(a.data)+1)
		copy(with[:i], a.data[:i])
		copy(with[i+1:], a.data[i:])
		with[i] = x
	}
	return nodeArray{with}, prev
}

func (a nodeArray) Delete(x uint) (cp nodeArray, prev *node) {
	// Inlined binary search
	var has bool
	i := len(a.data)
	{
		l := 0
		for !has && l < i {
			m := l + (i-l)/2
			switch {
			case a.data[m].key == x:
				has = true
				i = m
			case a.data[m].key < x:
				l = m + 1
			case a.data[m].key > x:
				i = m
			}
		}
	}
	if !has {
		return a, nil
	}
	without := make([]*node, len(a.data)-1)
	copy(without[:i], a.data[:i])
	copy(without[i:], a.data[i+1:])
	return nodeArray{without}, a.data[i]
}

func (a nodeArray) Ascend(cb func(x *node) bool) bool {
	for _, x := range a.data {
		if !cb(x) {
			return false
		}
	}
	return true
}

func (a nodeArray) AscendRange(x, y uint, cb func(x *node) bool) bool { // Inlined binary search
	var ok0 bool
	i := len(a.data)
	{
		l := 0
		for !ok0 && l < i {
			m := l + (i-l)/2
			switch {
			case a.data[m].key == x:
				ok0 = true
				i = m
			case a.data[m].key < x:
				l = m + 1
			case a.data[m].key > x:
				i = m
			}
		}
	}
	// Inlined binary search
	var ok1 bool
	j := len(a.data)
	{
		l := 0
		for !ok1 && l < j {
			m := l + (j-l)/2
			switch {
			case a.data[m].key == y:
				ok1 = true
				j = m
			case a.data[m].key < y:
				l = m + 1
			case a.data[m].key > y:
				j = m
			}
		}
	}
	for ; i < len(a.data) && i <= j; i++ {
		if !cb(a.data[i]) {
			return false
		}
	}
	return true
}

func (a nodeArray) Len() int {
	return len(a.data)
}
