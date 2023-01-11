.DEFAULT_GOAL: all
.PHONY: all build

all: build

build:
	@go build -o ./dist/go-struct-convert

clean:
	-@rm -rf ./dist
