package radix

import (
	"fmt"
	"io"
	"sync/atomic"
)

func graphviz(w io.Writer, label string, t *Trie) {
	id := new(int64)
	fmt.Fprintf(w, `digraph G {graph[label="%s"];`, label)
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
	if l.child != nil {
		cid := graphvizNode(w, l.child, id)
		fmt.Fprintf(w, `"%v"->"%v";`, i, cid)
	}
	return i
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
