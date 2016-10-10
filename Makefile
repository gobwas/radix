BENCH    ?= .
GTFLAGS  ?=
GRAPHVIZ ?= 0

TEMPLATES = $(wildcard $(PWD)/*.go.h)

enable_graphviz: 
	$(eval GRAPHVIZ:=1)

bin/viz: enable_graphviz generate
	go build ./tools/viz/...

clean:
	find . -name *_gen.go | xargs rm

test: enable_graphviz generate
	go test -v

bench: enable_graphviz generate
	go test -run=none -bench=$(BENCH) -benchmem $(GTFLAGS)

generate:
	for tmpl in $(TEMPLATES); do \
		name=`basename $$tmpl .h`; \
		base=`basename $$name .go` \
		output="$${base}_gen.go"; \
		tmp="$${output}.tmp"; \
	   	cc -Iinclude -DGRAPHVIZ=$(GRAPHVIZ) -E -P $$tmpl | sed -e $$'s/;;/\\\n/g' > $$tmp; \
		gofmt $$tmp > $$output; \
		rm -f $$tmp; \
	done;

.IGNORE: _test _bench _viz
.PHONY: viz

