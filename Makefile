BENCH    ?= .
GTFLAGS  ?=
GRAPHVIZ ?= 0

TEMPLATES = $(wildcard $(PWD)/*.go.h)

enable_graphviz:
	$(eval GRAPHVIZ:=1)

disable_graphviz: 
	$(eval GRAPHVIZ:=0)

bin/viz: enable_graphviz generate0 _viz disable_graphviz generate1
_viz:
	go build -o bin/viz ./tools/viz/...

clean:
	find . -name *_gen.go | xargs rm

test: enable_graphviz generate0 _test disable_graphviz generate1
_test:
	go test -v

bench: enable_graphviz generate0 _bench disable_graphviz generate1
_bench: 
	go test -run=none -bench=$(BENCH) -benchmem $(GTFLAGS)

generate: generate0
generate%:
	for tmpl in $(TEMPLATES); do \
		name=`basename $$tmpl .h`; \
		base=`basename $$name .go` \
		output="$${base}_gen.go"; \
		tmp="$${output}.tmp"; \
	   	cc -Iinclude -DGRAPHVIZ=$(GRAPHVIZ) -E -P $$tmpl \
			| sed -E -e 's/>>>/\/\//g' \
			| sed -e $$'s/;;/\\\n/g' \
		   	> $$tmp; \
		gofmt $$tmp > $$output; \
		rm -f $$tmp; \
	done;

.IGNORE: _test _bench _viz 
.PHONY: generate enable_graphviz disable_graphviz

