package converter

import (
	_ "embed"
	"errors"
	"os"
	"strings"

	"go/ast"
	"go/parser"
	"go/token"
)

type Converter interface {
	Convert(w *strings.Builder, f ast.Node) error
}

func ConvertFile(filename string, converter Converter) (*strings.Builder, error) {
	stat, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		return nil, errors.New("cannot load a directory")
	}

	contents, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	s := strings.TrimSpace(string(contents))
	if len(s) == 0 {
		return nil, errors.New("nothing to parse")
	}

	var f ast.Node
	f, err = parser.ParseExprFrom(token.NewFileSet(), "editor.go", s, parser.SpuriousErrors)
	if err != nil {
		// 		s = fmt.Sprintf(`package main

		// func main() {
		// 	%s
		// }`, s)

		f, err = parser.ParseFile(token.NewFileSet(), "editor.go", s, parser.SpuriousErrors)
		if err != nil {
			return nil, err
		}
	}

	builder := new(strings.Builder)
	err = converter.Convert(builder, f)
	if err != nil {
		return nil, err
	}

	return builder, nil

}
