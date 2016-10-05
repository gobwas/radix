package radix

type record struct {
	node  *node
	score int64
}

// Heap implements d-ary heap.
type Heap struct {
	d     int
	size  int
	data  []record
	index map[*node]int
}

func NewHeap(d, n int) *Heap {
	return &Heap{
		d:     d,
		data:  make([]record, 0, n),
		index: make(map[*node]int),
	}
}

func HeapFromSlice(data []record, d int) *Heap {
	h := &Heap{
		d:    d,
		data: data,
		size: len(data),
	}
	for i := len(data)/d - 1; i >= 0; i-- {
		h.SiftDown(i)
	}
	return h
}

func (h *Heap) Head() *node {
	return h.data[0].node
}

// Ascend iterates on all elements in heap starting from min.
func (h *Heap) Ascend(cb func(x *node) bool) {
	for i := 0; i < h.size; i++ {
		if !cb(h.data[i].node) {
			return
		}
	}
}

func (h *Heap) Len() int {
	return h.size
}

func (h *Heap) Update(i int, x record) {
	prev := h.data[i]
	h.data[i] = x
	if x.score > prev.score {
		h.SiftUp(i)
	} else {
		h.SiftDown(i)
	}
}

func (h *Heap) Modify(x *node, delta int64) {
	i, ok := h.index[x]
	if !ok {
		panic("could not update record out of heap")
	}
	h.Update(i, record{x, h.data[i].score + delta})
}

func (h *Heap) Less(a, b *node) bool {
	var i, j int
	i, ok := h.index[a]
	if ok {
		j, ok = h.index[b]
	}
	if !ok {
		panic("comparing record that not in heap")
	}
	return i > j
}

func (h *Heap) Insert(x *node) {
	i := h.size
	if h.size == len(h.data) {
		h.data = append(h.data, record{x, 0})
	} else {
		h.data[i] = record{x, 0}
	}
	h.index[x] = i
	h.size++
	h.SiftUp(i)
}

func (h *Heap) Pop() *node {
	ret := h.data[0]
	h.data[0] = h.data[h.size-1]
	h.size--
	h.SiftDown(0)
	return ret.node
}

func (h *Heap) Remove(i int) {
	h.siftTop(i)
	h.Pop()
}

func (h Heap) SiftDown(root int) {
	for {
		min := root
		for i := 1; i <= h.d; i++ {
			child := h.d*root + i
			if child >= h.size { // out of bounds
				break
			}
			if h.data[child].score > h.data[min].score {
				min = child
			}
		}
		if min == root {
			return
		}
		h.swap(root, min)
		root = min
	}
}

func (h Heap) SiftUp(root int) {
	for root > 0 {
		parent := root / (h.d + 1)
		if h.data[root].score <= h.data[parent].score {
			return
		}
		h.swap(parent, root)
		root = parent
	}
}

func (h Heap) siftTop(root int) {
	for root > 0 {
		parent := root / h.d
		h.swap(parent, root)
		root = parent
	}
}

func (h Heap) swap(i, j int) {
	a, b := h.data[i], h.data[j]
	h.index[a.node], h.index[b.node] = j, i
	h.data[i], h.data[j] = h.data[j], h.data[i]
}
