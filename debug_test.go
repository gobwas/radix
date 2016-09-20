package radix

import (
	"fmt"
	"io"
	"sync/atomic"
	"unsafe"
)

func graphviz(w io.Writer, label string, t *Trie) {
	id := new(int64)
	fmt.Fprintf(w, `digraph G {graph[label="%s(%vb)"];`, label, t.root.size())
	graphvizLeaf(w, "root", &t.root, id)
	fmt.Fprint(w, `}`)
}

func graphvizNode(w io.Writer, n *node, id *int64) int64 {
	i := nextID(id)
	fmt.Fprintf(w, `"%v"[label="%v"];`, i, n.key)
	for key, l := range n.values {
		lid := graphvizLeaf(w, key, l, id)
		fmt.Fprintf(w, `"%v"->"%v";`, i, lid)
	}
	return i
}

func graphvizLeaf(w io.Writer, value string, l *leaf, id *int64) int64 {
	i := nextID(id)
	fmt.Fprintf(w, `"%v"[label="%v"];`, i, value)
	if len(l.data) > 0 {
		d := graphvizData(w, l.data, id)
		fmt.Fprintf(w, `"%v"->"%v"[style=dashed,dir=none];`, i, d)
	}
	if l.node != nil {
		cid := graphvizNode(w, l.node, id)
		fmt.Fprintf(w, `"%v"->"%v";`, i, cid)
	}
	return i
}

func (l *leaf) size() (s uintptr) {
	s += unsafe.Sizeof(l)
	s += unsafe.Sizeof(l.data)
	for _, v := range l.data {
		s += unsafe.Sizeof(v)
	}
	s += unsafe.Sizeof(l.node)
	if l.node != nil {
		s += l.node.size()
	}
	return
}

func (n *node) size() (s uintptr) {
	s += unsafe.Sizeof(n)
	s += unsafe.Sizeof(n.key)
	s += uintptr(len(n.val)) + unsafe.Sizeof(n.val)
	s += unsafe.Sizeof(n.values)
	for val, leaf := range n.values {
		s += uintptr(len(val)) + unsafe.Sizeof(val)
		s += leaf.size()
	}
	return
}

func graphvizData(w io.Writer, data []int, id *int64) int64 {
	var str string
	for i, v := range data {
		if i > 0 {
			str += ","
		}
		str += fmt.Sprintf("%v", v)
	}
	n := nextID(id)
	fmt.Fprintf(w, `"%v"[label="%v",shape=polygon];`, n, str)
	return n
}

func nextID(id *int64) int64 { return atomic.AddInt64(id, 1) }
