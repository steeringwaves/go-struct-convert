.DEFAULT_GOAL: all
.PHONY: all ts c

all: c ts

c:
	@../dist/go-struct-convert c ./example.go --output dist/

ts:
	@../dist/go-struct-convert typescript ./example.go --output dist/

clean:
	-@rm -rf ./dist
