package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/gobwas/radix"
)

var sift = flag.String("sift", "", "which key to sift up in the trie")
var del = flag.String("del", "", "which key to delete in the trie")
var siftN = flag.Int("sift_times", 1, "how much to sift up found node")
var label = flag.String("label", "radix.Trie", "label to draw in graphviz diagram")

func scanPath(r io.Reader) (path radix.Path, number string, err error) {
	ws := bufio.NewScanner(r)
	ws.Split(bufio.ScanWords)
	var i int
	var key uint64
	for ws.Scan() {
		i++
		txt := strings.TrimSpace(ws.Text())
		if i%2 == 0 {
			key, err = strconv.ParseUint(number, 10, 64)
			if err != nil {
				return
			}
			path = path.With(uint(key), txt)
		} else {
			number = txt
		}
	}
	return
}

func scanPathVal(r io.Reader) (path radix.Path, val int, err error) {
	var rem string
	path, rem, err = scanPath(r)
	if err != nil {
		return
	}
	v, err := strconv.ParseInt(rem, 10, 64)
	if err != nil {
		return
	}
	val = int(v)
	return
}

func fatal(err error) {
	fmt.Printf("error: %s\n", err)
	flag.Usage()
	os.Exit(1)
}

func main() {
	flag.Parse()

	t := radix.New()
	ls := bufio.NewScanner(os.Stdin)
	ls.Split(bufio.ScanLines)
	for ls.Scan() {
		path, val, err := scanPathVal(bytes.NewReader(ls.Bytes()))
		if err != nil {
			fatal(err)
		}
		t.Insert(path, val)
	}

	// initial tree
	radix.Graphviz(os.Stdout, *label, t)

	if *sift != "" {
		path, _, err := scanPath(strings.NewReader(*sift))
		if err != nil {
			fatal(err)
		}
		n := radix.SearchNode(t, path)
		if n != nil {
			radix.MarkNode(n)
			fmt.Fprint(os.Stdout, "\n\n")
			radix.Graphviz(os.Stdout, *label, t)
			radix.UnmarkNode(n)
			for i := 0; i < *siftN; i++ {
				n = radix.SiftUp(n)
				fmt.Fprint(os.Stdout, "\n\n")
				radix.Graphviz(os.Stdout, *label, t)
			}
		}
	}

	if *del != "" {
		path, val, err := scanPathVal(strings.NewReader(*del))
		if err != nil {
			fatal(err)
		}
		if t.Delete(path, val) {
			fmt.Fprint(os.Stdout, "\n")
			radix.Graphviz(os.Stdout, *label, t)
		}
	}

}
