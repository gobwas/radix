package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"strconv"

	"github.com/gobwas/radix"
)

var action = flag.Bool("sift", "", "which key to sift up in the node")

func scanPath(r io.Reader) (path Pairs, val int, err error) {
	ws := bufio.NewScanner(r, bufio.ScanWords)
	var path radix.Pairs
	var i int
	var number string
	var key uint64
	for ws.Scan() {
		i++
		if i%2 == 0 {
			key, err = strconv.ParseUint(number, 10, 64)
			if err != nil {
				return
			}
			path = append(path, Pair{uint(key), ws.Text()})
		} else {
			number = ws.Text()
		}
	}
	if i%2 != 1 {
		return nil, fmt.Errorf("even path pairs")
	}
	v, err := strconv.ParseInt(ws.Text(), 10, 64)
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
	t := radix.New()
	ls := bufio.NewScanner(os.Stdin, bufio.ScanLines)
	for ls.Scan() {
		path, val, err := scanPath(bytes.NewReader(ls.Bytes()))
		if err != nil {
			fatal(err)
		}
		t.Insert(path, val)
	}

}
