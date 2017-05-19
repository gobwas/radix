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
func DumpLeaf(w io.Writer, leaf *radix.Leaf) error {
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

func Dump(w io.Writer, t *radix.Trie) error {
	p := New(w)
	return p.Print(t.Root())
}

func DumpLimit(w io.Writer, t *radix.Trie, nodes, leafs int) error {
	p := New(w)
	return p.PrintLimit(t.Root(), nodes, leafs)
}

func DumpString(t *radix.Trie) string {
	return DumpLeafString(t.Root())
}

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

type Digger struct {
	MaxLeafsPerNode int
	MaxNodesPerLeaf int

	OnNode func(*radix.Node) bool
	OnLeaf func(*radix.Leaf) bool
}

func (d *Digger) Dig(leaf *radix.Leaf) {
	d.OnLeaf(leaf)
	d.dig(leaf)
}

func (d *Digger) dig(leaf *radix.Leaf) {
	nodeRemain := d.MaxNodesPerLeaf
	if nodeRemain == 0 {
		nodeRemain = leaf.ChildrenCount()
	}
	leaf.AscendChildren(func(node *radix.Node) bool {
		if d.OnNode(node) {
			nodeRemain--
		}

		leafRemain := d.MaxLeafsPerNode
		if leafRemain == 0 {
			leafRemain = node.LeafCount()
		}
		node.AscendLeafs(func(_ string, leaf *radix.Leaf) bool {
			if d.OnLeaf(leaf) {
				leafRemain--
			}
			d.dig(leaf)
			return leafRemain > 0
		})

		return nodeRemain > 0
	})
}

func (d *Printer) getDigger(n, l int) *Digger {
	limit := func(n, limit int) int {
		if n < limit || limit == 0 {
			return n
		}
		return limit
	}

	return &Digger{
		MaxNodesPerLeaf: n,
		MaxLeafsPerNode: l,

		OnNode: func(node *radix.Node) bool {
			d.leafs[node] = limit(node.LeafCount(), l)
			d.nodes[node.Parent()] = limit(d.nodes[node.Parent()]-1, n)

			d.writeNode(node)

			return true
		},
		OnLeaf: func(leaf *radix.Leaf) bool {
			d.nodes[leaf] = limit(leaf.ChildrenCount(), n)
			if p := leaf.Parent(); p != nil {
				d.leafs[p] = limit(d.leafs[p]-1, l)
			}

			d.writeLeaf(leaf)

			return true
		},
	}

}

// Print draws given trie to underlying destination.
func (d *Printer) Print(leaf *radix.Leaf) error {
	d.getDigger(0, 0).Dig(leaf)
	return d.bw.Flush()
}

func (d *Printer) PrintLimit(leaf *radix.Leaf, nodes, leafs int) error {
	d.getDigger(nodes, leafs).Dig(leaf)
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

	d.bw.WriteString(fmt.Sprintf("%#x", leaf.Data()))
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
