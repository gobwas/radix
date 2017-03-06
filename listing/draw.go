package listing

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/gobwas/radix"
)

// DumpLeaf draws given leaf to given writer.
// Return error only when write fails.
func DumpLeaf(leaf *radix.Leaf, w io.Writer) error {
	p := New(w)
	return p.Print(leaf)
}

// DumpLeafString draws given leaf to a string.
func DumpLeafString(leaf *radix.Leaf) string {
	buf := bytes.Buffer{}
	p := New(&buf)
	p.Print(leaf)
	return buf.String()
}

func Dump(t *radix.Trie, w io.Writer) error { return DumpLeaf(t.Root(), w) }
func DumpString(t *radix.Trie) string       { return DumpLeafString(t.Root()) }

// Printer contains options for dumping a trie.
type Printer struct {
	leafs map[*radix.Node]int
	nodes map[*radix.Leaf]int

	leafTab map[*radix.Leaf]int
	nodeTab map[*radix.Node]int

	bw *bufio.Writer
}

// New creates new Printer that writes to w.
func New(w io.Writer) *Printer {
	d := &Printer{}
	d.bw = bufio.NewWriter(w)
	d.resetState()
	return d
}

// Reset completely resets Printer state.
func (d *Printer) Reset(w io.Writer) {
	d.bw.Reset(w)
	d.resetState()
}

// Print draws given trie to underlying destination.
func (d *Printer) Print(leaf *radix.Leaf) error {
	radix.Dig(leaf, radix.VisitorFunc(
		func(path []radix.Pair, leaf *radix.Leaf) bool {
			d.nodes[leaf] = leaf.ChildrenCount()
			d.leafs[leaf.Parent()] = d.leafs[leaf.Parent()] - 1

			d.writeLeaf(leaf)

			return true
		},
		func(path []radix.Pair, node *radix.Node) bool {
			d.leafs[node] = node.LeafCount()
			d.nodes[node.Parent()] = d.nodes[node.Parent()] - 1

			d.writeNode(node)

			return true
		},
	))
	return d.bw.Flush()
}

func (d *Printer) resetState() {
	d.leafs = map[*radix.Node]int{}
	d.nodes = map[*radix.Leaf]int{}
	d.leafTab = map[*radix.Leaf]int{}
	d.nodeTab = map[*radix.Node]int{}
}

func (d *Printer) writeLeaf(leaf *radix.Leaf) {
	key := leaf.Value()
	if key == "" {
		key = "/"
	}

	var prefix string
	if d.leafs[leaf.Parent()] <= 0 {
		prefix = "└──"
	} else {
		prefix = "├──"
	}

	suffix := "──"

	d.leafTab[leaf] = utf8.RuneCountInString(prefix) + middle(utf8.RuneCountInString(key))

	d.writeLeafPrefix(leaf)

	d.bw.WriteString(prefix)
	d.bw.WriteString(key)
	d.bw.WriteString(suffix)

	d.bw.WriteString(fmt.Sprintf("%v", leaf.Data()))
	d.bw.WriteByte('\n')
}

func (d *Printer) writeNode(node *radix.Node) {
	key := "0x" + strconv.FormatUint(uint64(node.Key()), 16)

	var prefix string
	if d.nodes[node.Parent()] <= 0 {
		prefix = "└──"
	} else {
		prefix = "├──"
	}

	d.nodeTab[node] = utf8.RuneCountInString(prefix) + middle(utf8.RuneCountInString(key))

	d.writeNodePrefix(node)

	d.bw.WriteString(prefix)
	d.bw.WriteString(key)
	d.bw.WriteString("\n")
}

func (d *Printer) writeLeafPrefix(leaf *radix.Leaf) {
	var p int
	if gp := grandLeafParent(leaf); gp != nil {
		p = d.writePathPrefixLeaf(gp)
	}

	d.tab(d.nodeTab[leaf.Parent()] - p)

	return
}

func (d *Printer) writeNodePrefix(node *radix.Node) {
	var p int
	if gp := grandNodeParent(node); gp != nil {
		p = d.writePathPrefixNode(gp)
	}

	d.tab(d.leafTab[node.Parent()] - p)

	return
}

func (d *Printer) writePathPrefixNode(node *radix.Node) (p int) {
	if node == nil {
		return 0
	}

	p = d.writePathPrefixLeaf(node.Parent())

	var suffix string
	if d.leafs[node] > 0 {
		suffix = "│"
	}

	t := d.nodeTab[node]
	d.tab(t - p)
	d.bw.WriteString(suffix)

	return utf8.RuneCountInString(suffix)
}

func (d *Printer) writePathPrefixLeaf(leaf *radix.Leaf) (p int) {
	if leaf == nil {
		return 0
	}

	p = d.writePathPrefixNode(leaf.Parent())

	var suffix string
	if d.nodes[leaf] > 0 {
		suffix = "│"
	}

	t := d.leafTab[leaf]
	d.tab(t - p)
	d.bw.WriteString(suffix)

	return utf8.RuneCountInString(suffix)
}

func (d *Printer) tab(n int) {
	d.bw.Write(bytes.Repeat([]byte{' '}, n))
}

func grandLeafParent(leaf *radix.Leaf) *radix.Leaf {
	n := leaf.Parent()
	if n == nil {
		return nil
	}
	return n.Parent()
}

func grandNodeParent(node *radix.Node) *radix.Node {
	l := node.Parent()
	if l == nil {
		return nil
	}
	return l.Parent()
}

func middle(n int) int {
	return int(float64(n) / 2)
}
