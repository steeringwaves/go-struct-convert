.DEFAULT_GOAL: all
.PHONY: all ts c

all: c ts

c:
	@../dist/go-struct-convert c ./example.go --output dist/ --name Example
	@../dist/go-struct-convert c ./example.go --output dist/ --name Example2 --prefix Example2 --suffix _t

ts:
	@../dist/go-struct-convert typescript ./example.go --output dist/ --name example --namespace Example 
	@../dist/go-struct-convert typescript ./example.go --output dist/ --name NoNamespace
	@../dist/go-struct-convert typescript ./example.go --output dist/ --name Prefixes --prefix Example --suffix _t

clean:
	-@rm -rf ./dist
