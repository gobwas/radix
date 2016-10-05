BENCH   ?= .
GTFLAGS ?=

viz: graphviz _viz clean
_viz:
	go build ./tools/viz/...

graphviz:
	cpp -DRADIX_DEBUG -P radix_graphviz.pgo tmp_radix_graphviz.go

clean:
	find . -name tmp_*.go | xargs rm

test: 
	go test -v

bench:
	go test -run=none -bench=$(BENCH) -benchmem $(GTFLAGS)

.IGNORE: _test _viz
.PHONY: viz
