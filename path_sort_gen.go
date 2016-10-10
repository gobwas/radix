package radix

func pairPartition(data []Pair, l, r int) int {
	x := data[l]
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

func pairQuickSort(data []Pair, lo, hi int) {
	if lo >= hi {
		return
	}
	p := pairPartition(data, lo, hi)
	pairQuickSort(data, lo, p)
	pairQuickSort(data, p+1, hi)
}

func pairInsertionSort(data []Pair, l, r int) {

	for i := l + 1; i < r; i++ {
		for j := i; j > l && data[j-1].Key > data[j].Key; j-- {
			data[j], data[j-1] = data[j-1], data[j]
		}
	}

}

func pairSort(data []Pair, l, r int) {
	if r-l > 12 {
		pairQuickSort(data, l, r)
		return
	}

	for i := l + 1; i < r; i++ {
		for j := i; j > l && data[j-1].Key > data[j].Key; j-- {
			data[j], data[j-1] = data[j-1], data[j]
		}
	}

}

func pairSearch(data []Pair, key uint) (int, bool) {

	var ok bool
	i := len(data)
	{
		l := 0
		for !ok && l < i {
			m := l + (i-l)/2
			switch {
			case data[m].Key == key:
				ok = true
				i = m
			case data[m].Key < key:
				l = m + 1
			case data[m].Key > key:
				i = m
			}
		}
	}

	return i, ok
}
