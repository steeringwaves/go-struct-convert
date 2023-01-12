package converter

import (
	_ "embed"
	"errors"
	"os"
	"regexp"
	"strings"

	"go/ast"
	"go/parser"
	"go/token"

	"github.com/samber/lo"
)

type Converter interface {
	FileExtension() string
	Convert(w *strings.Builder, f []ast.Node) error
}

func ConvertFile(inputs []string, converter Converter) (*strings.Builder, error) {
	var asts []ast.Node

	for _, filename := range inputs {
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

		// need to strip out any additional package declarations

		s := strings.TrimSpace(string(contents))
		if len(s) == 0 {
			return nil, errors.New("nothing to parse")
		}

		var f ast.Node
		f, err = parser.ParseExprFrom(token.NewFileSet(), filename, s, parser.AllErrors|parser.ParseComments)
		if err != nil {
			// 		s = fmt.Sprintf(`package main

			// func main() {
			// 	%s
			// }`, s)

			f, err = parser.ParseFile(token.NewFileSet(), filename, s, parser.AllErrors|parser.ParseComments)
			if err != nil {
				return nil, err
			}
		}

		asts = append(asts, f)
	}

	builder := new(strings.Builder)
	err := converter.Convert(builder, asts)
	if err != nil {
		return nil, err
	}

	return builder, nil

}

type Comments struct {
	CIncludes []string

	TypescriptImports []string
}

func CleanCInclude(str string) string {
	// be nice and clean up user includes
	var re = regexp.MustCompile(`(?m)([\s]*#[\s]*include[\s]*)`)

	cleaned := strings.TrimSpace(str)
	for _, match := range re.FindAllString(cleaned, -1) {
		cleaned = cleaned[len(match):]
		break
	}

	return cleaned
}

func CleanTypescriptImport(str string) string {
	// be nice and clean up user imports
	var re = regexp.MustCompile(`(?m)([\s]*import[\s]*)`)

	cleaned := strings.TrimSpace(str)
	for _, match := range re.FindAllString(cleaned, -1) {
		cleaned = cleaned[len(match):]
		break
	}

	// remove trailing semicolons
	var semiRe = regexp.MustCompile(`(?m)([;]+.*)`)
	matches := semiRe.FindAllString(cleaned, -1)
	if len(matches) == 1 {
		cleaned = cleaned[:len(cleaned)-len(matches[0])]
	}

	return cleaned
}

var cIncludeRe = regexp.MustCompile(`(?m)([\s]*#[\s]*c.include[\s]*)`)
var tsImportRe = regexp.MustCompile(`(?m)([\s]*#[\s]*ts.import[\s]*)`)

func HandleFileComments(comments []*ast.CommentGroup, result *Comments) error {
	for _, comment := range comments {
		lines := strings.Split(comment.Text(), "\n")
		for _, line := range lines {
			matches := cIncludeRe.FindAllString(line, -1)
			if len(matches) == 1 {
				cleaned := CleanCInclude(line[len(matches[0]):])

				_, exists := lo.Find[string](result.CIncludes, func(i string) bool {
					return i == cleaned
				})

				if !exists {
					result.CIncludes = append(result.CIncludes, cleaned)
				}

				continue
			}

			matches = tsImportRe.FindAllString(line, -1)
			if len(matches) == 1 {
				cleaned := CleanTypescriptImport(line[len(matches[0]):])

				_, exists := lo.Find[string](result.TypescriptImports, func(i string) bool {
					return i == cleaned
				})

				if !exists {
					result.TypescriptImports = append(result.TypescriptImports, cleaned)
				}

				continue
			}
		}
	}

	return nil
}
