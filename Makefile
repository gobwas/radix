viz:
	cpp -DRADIX_DEBUG -P radix_graphviz.pgo tmp_radix_graphviz.go
	go build ./tools/viz/...
	rm -f tmp_radix_graphviz.go

clean:
	find . -name tmp_*.go | xargs rm

.PHONY: viz
