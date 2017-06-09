package graphviz

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sync/atomic"
	"unsafe"

	"github.com/gobwas/radix"
)

var markup = map[*radix.Node]bool{}

func MarkNode(n *radix.Node) {
	markup[n] = true
}

func UnmarkNode(n *radix.Node) {
	delete(markup, n)
}

var i *uint32 = new(uint32)

func WriteFile(r io.Reader, label string) (*os.File, error) {
	f, err := ioutil.TempFile("", label)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("dot", "-Tpng", "-o", f.Name())
	cmd.Stdin = r
	return f, cmd.Run()
}

func ShowFile(f *os.File) error {
	cmd := exec.Command("open", f.Name())
	return cmd.Run()
}

func Show(t *radix.Trie, label string) error {
	buf := &bytes.Buffer{}
	Render(t, label, buf)

	f, err := WriteFile(buf, label)
	if err != nil {
		return err
	}
	return ShowFile(f)
}

func ShowLeaf(l *radix.Leaf, label string) error {
	buf := &bytes.Buffer{}
	RenderLeaf(l, label, buf)

	f, err := WriteFile(buf, label)
	if err != nil {
		return err
	}
	return ShowFile(f)
}

func RenderLeaf(leaf *radix.Leaf, label string, w io.Writer) {
	fmt.Fprintf(w, `digraph Trie%d {graph[label="%s"]; node[style=filled];`, atomic.AddUint32(i, 1), label)
	v := &visitor{
		w:  w,
		id: newID(),
	}
	radix.Dig(leaf, v)
	v.done()
	fmt.Fprint(w, `}`)
}

func Render(t *radix.Trie, label string, w io.Writer) {
	fmt.Fprintf(w, `digraph Trie%d {graph[label="%s"]; node[style=filled];`, atomic.AddUint32(i, 1), label)
	v := &visitor{
		w:  w,
		id: newID(),
	}
	t.Walk(radix.Path{}, v)
	v.done()
	fmt.Fprint(w, `}`)
}

type visitor struct {
	w    io.Writer
	id   *id
	size uintptr
	root *radix.Leaf
}

func (v *visitor) OnNode(trace []radix.PairStr, n *radix.Node) bool {
	i := v.id.node(n)
	if markup[n] {
		fmt.Fprintf(v.w, `"%v"[label="%v" fillcolor="#ffffff" color="red"];`, i, n.Key())
	} else {
		fmt.Fprintf(v.w, `"%v"[label="%v" fillcolor="#ffffff"];`, i, n.Key())
	}
	v.size += nodeSize(n)
	n.AscendLeafs(func(_ string, l *radix.Leaf) bool {
		child(v.w, i, v.id.leaf(l))
		return true
	})
	if p := n.Parent(); p != nil {
		parent(v.w, i, v.id.leaf(p))
	}
	return true
}

func (v *visitor) OnLeaf(trace []radix.PairStr, l *radix.Leaf) bool {
	i := v.id.leaf(l)
	if v.root == nil {
		v.root = l
	}
	var value []byte
	if p := l.Parent(); p != nil {
		var ok bool
		value, ok = radix.PathFromSliceStr(trace).Get(l.Parent().Key())
		if !ok {
			panic(fmt.Sprintf("trie is broken"))
		}
		parent(v.w, i, v.id.node(p))
	}
	fmt.Fprintf(v.w, `"%v"[label="%s" fillcolor="#bef1cf"];`, i, value)
	if data := l.Data(); len(data) > 0 {
		d := graphvizData(v.w, data, v.id)
		relation(v.w, i, d)
	}
	l.AscendChildren(func(n *radix.Node) bool {
		child(v.w, i, v.id.node(n))
		return true
	})
	return true
}

func (v *visitor) done() {
	if v.root == nil {
		return
	}
	i := v.id.next()
	fmt.Fprintf(v.w, `"%v"[label="%vb" fillcolor="#efefef" shape=cylinder penwidth=0.3];`, i, v.size)
	relation(v.w, i, v.id.leaf(v.root))
}

func parent(w io.Writer, a, b int64) {
	fmt.Fprintf(w, `"%v"->"%v"[dir=forward style=dashed color="#cccccc"];`, a, b)
}

func child(w io.Writer, a, b int64) {
	fmt.Fprintf(w, `"%v"->"%v"[dir=forward];`, a, b)
}

func relation(w io.Writer, a, b int64) {
	fmt.Fprintf(w, `"%v"->"%v"[style=dashed,dir=none];`, a, b)
}

func leafSize(l *radix.Leaf) (s uintptr) {
	for _, v := range l.Data() {
		s += unsafe.Sizeof(v)
	}
	l.AscendChildren(func(n *radix.Node) bool {
		s += nodeSize(n)
		return true
	})
	return
}

func nodeSize(n *radix.Node) (s uintptr) {
	s += unsafe.Sizeof(n.Key())
	n.AscendLeafs(func(v string, l *radix.Leaf) bool {
		s += uintptr(len(v))
		s += leafSize(l)
		return true
	})
	return
}

func graphvizData(w io.Writer, data []uint, id *id) int64 {
	var str string
	for _, v := range data {
		str += fmt.Sprintf("%v;", v)
	}
	n := id.next()
	fmt.Fprintf(w, `"%v"[label="%v" fillcolor="#cccccc" shape=polygon];`, n, str)
	return n
}

type id struct {
	c int64
	n map[*radix.Node]int64
	l map[*radix.Leaf]int64
}

func newID() *id {
	return &id{
		c: 1000000,
		n: make(map[*radix.Node]int64),
		l: make(map[*radix.Leaf]int64),
	}
}

func (id *id) node(n *radix.Node) int64 {
	if _, ok := id.n[n]; !ok {
		id.n[n] = id.next()
	}
	return id.n[n]
}

func (id *id) leaf(l *radix.Leaf) int64 {
	if _, ok := id.l[l]; !ok {
		id.l[l] = id.next()
	}
	return id.l[l]
}

func (id *id) next() int64 {
	return atomic.AddInt64(&id.c, 1)
}
