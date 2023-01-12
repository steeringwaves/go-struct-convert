.DEFAULT_GOAL: all
.PHONY: all ts c

all: c ts

c:
	@../dist/go-struct-convert c ./example.go ./another.go --output dist/ --name Example
	@../dist/go-struct-convert c ./example.go ./another.go --output dist/ --name Example2 --prefix Example2 --suffix _t
	@../dist/go-struct-convert c ./example.go ./another.go --output dist/ --name Example3 --suffix _t --include "#include <sys/stat.h>" --include=" #  include   <stdbool.h>"

ts:
	@../dist/go-struct-convert typescript ./example.go ./another.go --output dist/ --name example --namespace Example 
	@../dist/go-struct-convert typescript ./example.go ./another.go --output dist/ --name NoNamespace
	@../dist/go-struct-convert typescript ./example.go ./another.go --output dist/ --name Prefixes --prefix Example --suffix _t
	@../dist/go-struct-convert typescript ./example.go ./another.go --output dist/ --name Imports --import "import 'lodash'"

clean:
	-@rm -rf ./dist
