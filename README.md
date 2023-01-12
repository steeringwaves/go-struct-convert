# go-struct-convert
![workflow](https://github.com/steeringwaves/go-struct-convert/actions/workflows/test.yml/badge.svg)
[![Go Reference](https://pkg.go.dev/badge/github.com/steeringwaves/go-struct-convert.svg)](https://pkg.go.dev/github.com/steeringwaves/go-struct-convert)


## roadmap

- [x] Generate c header from go file
- [x] Generate typescript from go file
- [ ] Parse nested go structs
- [ ] Parse multiple go files

### go -> c

- [x] Generate c header from single go file and include declarations for all structs and struct members
- [ ] Generate c header from multiple go files and include declarations for all structs and struct members
- [x] Generate c header with an optional prefix that is applied to all struct types `--prefix <name>`
- [x] Generate c header with an optional suffix that is applied to all struct types `--suffix <name>`
- [x] Parse and apply c types from reflect tags `ctype:"char *"` or `ctype:"char[255]"`
- [x] Carry comments on struct members forward to c
- [x] Generate `#include` statements from cli flags `--include '#include <stdint.h>'`
- [ ] Generate `#include` statements from cli flags `--include '#include "myfile.h>"` (cobra does not like the quotes)
- [x] Generate `#include` statements from inline comments `// #c.include #include <stdint.h>` or `// #c.include <stdint.h>`

### go -> ts

- [x] Generate typescript from single go file and include interface declarations for all structs and struct members
- [ ] Generate typescript from multiple go files and include interface declarations for all structs and struct members
- [x] Generate typescript with an optional namespace `--namespace <name>` and nest all interface declartions underneath
- [x] Generate typescript with an optional prefix that is applied to all struct types `--prefix <name>`
- [x] Generate typescript with an optional suffix that is applied to all struct types `--suffix <name>`
- [ ] Parse and apply typescript types from reflect tags `tstype:"number"` or `tstype:"[]number"`
- [ ] Generate `import` statements from cli flags `--import 'import "lodash"'` (cobra does not like the quotes)
- [x] Generate `import` statements from inline comments `// #ts.import import "lodash"` or `// #ts.import import { uniq } from "lodash"`

### strech goals

- [ ] Generate code to parse json to struct
- [ ] Generate code to convert struct to json

## usage

```sh

go-struct-convert typescript example/example.go

# output file to a directory
go-struct-convert typescript example/example.go --output dist/

```

