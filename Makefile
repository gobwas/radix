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
	go generate ./...

.IGNORE: _test _bench _viz 
.PHONY: generate enable_graphviz disable_graphviz

