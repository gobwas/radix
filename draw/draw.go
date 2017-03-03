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

type record struct {
	children int
	tab      int
}

func Draw(w io.Writer, t *radix.Trie) error {
	d := New(w)
	return d.Draw(t)
}

type Drawer struct {
	leafs map[*radix.Node]int
	nodes map[*radix.Leaf]int

	leafTab map[*radix.Leaf]int
	nodeTab map[*radix.Node]int

	bw *bufio.Writer
}

func New(w io.Writer) *Drawer {
	d := &Drawer{}

	d.leafs = map[*radix.Node]int{}
	d.nodes = map[*radix.Leaf]int{}

	d.leafTab = map[*radix.Leaf]int{}
	d.nodeTab = map[*radix.Node]int{}

	d.bw = bufio.NewWriter(w)

	return d
}

func (d *Drawer) Reset(w io.Writer) {
	d.bw.Reset(w)
}

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

	d.leafTab[leaf] = d.nodeTab[leaf.Parent()] + utf8.RuneCountInString(prefix) + utf8.RuneCountInString(key)
	d.writeLeafPrefix(leaf)

	d.bw.WriteString(prefix)
	d.bw.WriteString(key)
	d.bw.WriteString(suffix)

	d.bw.WriteString(fmt.Sprintf("%v", leaf.Data()))
	d.bw.WriteByte('\n')
}

func (d *Drawer) writeNode(node *radix.Node) {
	key := strconv.FormatUint(uint64(node.Key()), 10)

	var prefix string
	if d.nodes[node.Parent()] <= 0 {
		prefix = "└──"
	} else {
		prefix = "├──"
	}

	d.nodeTab[node] = d.leafTab[node.Parent()] + 2
	d.writeNodePrefix(node)

	d.bw.WriteString(prefix)
	d.bw.WriteString(key)
	d.bw.WriteString("\n")
}

func (d *Drawer) writeLeafPrefix(leaf *radix.Leaf) (n int) {
	gp := grandLeafParent(leaf)
	if gp != nil {
		n = d.writeLeafPrefix(gp)

		t := d.leafTab[gp]
		d.tab(t)
		n += t

		if d.nodes[gp] > 0 {
			d.bw.WriteString("│")
		}
	}

	//t := d.nodeTab[leaf.Parent()]
	//d.tab(t*0 + 1)
	//n += t

	return
}

func (d *Drawer) writeNodePrefix(node *radix.Node) (n int) {
	gp := grandNodeParent(node)
	if gp != nil {
		n = d.writeNodePrefix(gp)

		t := d.nodeTab[gp]
		//d.tab(t)
		n += t

		if d.leafs[gp] > 0 {
			d.bw.WriteString("│")
		}
	}

	t := d.leafTab[node.Parent()]
	d.tab(t)
	//n += t

	return
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
