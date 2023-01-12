.DEFAULT_GOAL: all
.PHONY: all build example

all: build example

build:
	@go build -o ./dist/go-struct-convert

example:
	@make -C example --no-print-directory all

clean:
	@make -C example --no-print-directory clean
	-@rm -rf ./dist
