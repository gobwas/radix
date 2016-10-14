BENCH    ?= .
GTFLAGS  ?=

TEMPLATES = $(wildcard $(PWD)/*.go.h)
GENERATED = $(wildcard $(PWD)/*_gen.go $(PWD)/*_gen_test.go)
GRAPHICS = $(wildcard $(PWD)/*.png)

bin/viz: 
	go build -o bin/viz ./tools/viz/...

clean:
	for file in $(GENERATED); do [ -f $$file ] && rm $$file; done
	for file in $(GRAPHICS); do [ -f $$file ] && rm $$file; done

test:
	go test -v

bench: 
	go test -run=none -bench=$(BENCH) -benchmem $(GTFLAGS)

generate:
	for tmpl in $(TEMPLATES); do \
		name=`basename $$tmpl .h`; \
		base=`basename $$name .go` \
		output="$${base}_gen.go"; \
		if [[ "$$base" =~ "_test" ]]; then \
			base=`basename $$base _test`; \
			output="$${base}_gen_test.go"; \
		fi; \
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

