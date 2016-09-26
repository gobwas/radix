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

var sift = flag.Uint("sift", 0, "which key to sift up in the node")
var label = flag.String("label", "radix.Trie", "label to draw in graphviz diagram")

func scanPath(r io.Reader) (path radix.Pairs, val int, err error) {
	ws := bufio.NewScanner(r)
	ws.Split(bufio.ScanWords)
	var i int
	var number string
	var key uint64
	for ws.Scan() {
		i++
		txt := strings.TrimSpace(ws.Text())
		if i%2 == 0 {
			key, err = strconv.ParseUint(number, 10, 64)
			if err != nil {
				return
			}
			path = append(path, radix.Pair{uint(key), txt})
		} else {
			number = txt
		}
	}
	if i%2 != 1 {
		return nil, 0, fmt.Errorf("even path pairs")
	}
	v, err := strconv.ParseInt(number, 10, 64)
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
		path, val, err := scanPath(bytes.NewReader(ls.Bytes()))
		if err != nil {
			fatal(err)
		}
		t.Insert(path, val)
	}

	radix.Graphviz(os.Stdout, *label, t)
	if *sift != 0 {
		radix.SiftUp(t, *sift)
		fmt.Fprint(os.Stdout, "\n")
		radix.Graphviz(os.Stdout, *label, t)
	}
}
