package draw

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/gobwas/radix"
)

// Draw draws given trie to given writer.
// Return error only when write fails.
func Draw(w io.Writer, t *radix.Trie) error {
	d := New(w)
	return d.Draw(t)
}

// Drawer holds options for drawing trie.
type Drawer struct {
	leafs map[*radix.Node]int
	nodes map[*radix.Leaf]int

	leafTab map[*radix.Leaf]int
	nodeTab map[*radix.Node]int

	bw *bufio.Writer
}

// New creates new Drawer that writes to w.
func New(w io.Writer) *Drawer {
	d := &Drawer{}
	d.bw = bufio.NewWriter(w)
	d.resetState()
	return d
}

// Reset completely resets Drawer state.
func (d *Drawer) Reset(w io.Writer) {
	d.bw.Reset(w)
	d.resetState()
}

func (d *Drawer) resetState() {
	d.leafs = map[*radix.Node]int{}
	d.nodes = map[*radix.Leaf]int{}
	d.leafTab = map[*radix.Leaf]int{}
	d.nodeTab = map[*radix.Node]int{}
}

// Draw draws given trie to underlying destination.
func (d *Drawer) Draw(t *radix.Trie) error {
	radix.Dig(t.Root(), radix.VisitorFunc(
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

func (d *Drawer) writeLeaf(leaf *radix.Leaf) {
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

func (d *Drawer) writeNode(node *radix.Node) {
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

func (d *Drawer) writeLeafPrefix(leaf *radix.Leaf) {
	var p int
	if gp := grandLeafParent(leaf); gp != nil {
		p = d.writePathPrefixLeaf(gp)
	}

	d.tab(d.nodeTab[leaf.Parent()] - p)

	return
}

func (d *Drawer) writeNodePrefix(node *radix.Node) {
	var p int
	if gp := grandNodeParent(node); gp != nil {
		p = d.writePathPrefixNode(gp)
	}

	d.tab(d.leafTab[node.Parent()] - p)

	return
}

func (d *Drawer) writePathPrefixNode(node *radix.Node) (p int) {
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

func (d *Drawer) writePathPrefixLeaf(leaf *radix.Leaf) (p int) {
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

func (d *Drawer) tab(n int) {
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
