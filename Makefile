BENCH   ?= .
GTFLAGS ?=

viz: graphviz _viz clean
_viz:
	go build ./tools/viz/...

graphviz:
	cpp -DRADIX_DEBUG -P radix_graphviz.pgo tmp_radix_graphviz.go

clean:
	find . -name tmp_*.go | xargs rm

test: graphviz _test clean

_test:
	go test -v

bench: graphviz _bench clean
_bench:
	go test -run=none -bench=$(BENCH) -benchmem $(GTFLAGS)

.IGNORE: _test _bench _viz
.PHONY: viz
