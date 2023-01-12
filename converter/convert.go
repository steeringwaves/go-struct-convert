package converter

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"go/ast"
	"go/parser"
	"go/token"

	"github.com/fatih/structtag"
	"github.com/samber/lo"
)

type Converter interface {
	FileExtension() string
	Convert(w *strings.Builder, f []ast.Node) error

	////
	ValidName(n string) bool
	GetIdent(s string) string
	String(mappedTypes map[string]string, comments Comments, structs []Struct) error
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

// /////////////
type Inspecter struct {
	Converter   Converter
	Prefix      string
	Suffix      string
	MappedTypes map[string]string
	Comments    Comments

	Structs []Struct
}
type StructMember struct {
	Name       string
	Type       string
	IsPointer  bool
	TypeSuffix string
	Comment    string
}

type Struct struct {
	Name    string
	Members []StructMember
	Comment string
}

func (inspecter *Inspecter) InspectType(t ast.Expr, depth int, parent *Struct) (string, error) {
	switch t := t.(type) {
	case *ast.ArrayType: // TODO
		if v, ok := t.Elt.(*ast.Ident); ok && v.String() == "byte" {
			return inspecter.Converter.GetIdent("[]byte"), nil
		}
		res, err := inspecter.InspectType(t.Elt, depth, parent)
		if err != nil {
			return "", err
		}
		return res, nil
	// case *ast.StructType:
	// 	// TODO this is wrong
	// 	_, err := c.InspectType(t.Fields.List, depth+1, parent)
	// 	if err != nil {
	// 		return "", err
	// 	}

	case *ast.Ident:
		return inspecter.Converter.GetIdent(t.String()), nil
	case *ast.SelectorExpr:
		longType := fmt.Sprintf("%s.%s", t.X, t.Sel)
		return inspecter.Converter.GetIdent(longType), nil
	case *ast.InterfaceType:
		return inspecter.Converter.GetIdent("interface"), nil
	default:
		return "", fmt.Errorf("unhandled: %s, %T", t, t)
	}

	return "", nil
}

func (inspecter *Inspecter) InspectFields(fields []*ast.Field, depth int, parent *Struct) error {
	for _, f := range fields {
		var fieldName string
		if len(f.Names) != 0 && f.Names[0] != nil && len(f.Names[0].Name) != 0 {
			fieldName = f.Names[0].Name
		}
		if len(fieldName) == 0 || 'A' > fieldName[0] || fieldName[0] > 'Z' {
			continue
		}

		comment := f.Comment.Text()
		if comment != "" {
			// TODO parse special comments
		}

		var name string
		var cType string
		var cTypeSuffix string
		// var validator Validator
		// usingValidator := false
		if f.Tag != nil {
			tags, err := structtag.Parse(f.Tag.Value[1 : len(f.Tag.Value)-1])
			if err != nil {
				return err
			}

			// get ctype tag
			cTypeTag, err := tags.Get("ctype")
			if err == nil {
				idx := strings.Index(cTypeTag.Name, "[")
				if idx > 0 {
					cType = cTypeTag.Name[0:idx]
					cTypeSuffix = cTypeTag.Name[idx:]
				} else {
					cType = cTypeTag.Name
				}
			}

			jsonTag, err := tags.Get("json")
			if err == nil {
				name = jsonTag.Name
				if name == "-" {
					continue
				}

				// optional = jsonTag.HasOption("omitempty")
			}

			// get validator tag
			// validatorTag, err := tags.Get("validator")
			// if err == nil {
			// 	usingValidator, validator = getValidatorFromTag(validatorTag.String())
			// }
		}

		if len(name) == 0 {
			name = fieldName
		}

		// quoted := !CValidCName(name)
		isPointer := false

		switch t := f.Type.(type) {
		case *ast.StarExpr:
			f.Type = t.X
			isPointer = true
		}

		member := StructMember{
			Name:       name,
			IsPointer:  isPointer,
			TypeSuffix: cTypeSuffix,
			Comment:    comment,
			// TODO what is quoted??
		}

		if cType != "" {
			member.Type = cType
		} else {
			switch t := f.Type.(type) {
			case *ast.StructType:
				// Nested struct, deal with it
				newStruct := Struct{
					Name: name,
				}

				err := inspecter.InspectFields(t.Fields.List, 0, &newStruct)
				if err != nil {
					return err
				}

				inspecter.Structs = append(inspecter.Structs, newStruct)

				newName := inspecter.Prefix + name + inspecter.Suffix
				inspecter.MappedTypes[name] = newName

				member.Type = name
			default:
				res, err := inspecter.InspectType(f.Type, depth, parent)
				if err != nil {
					return err
				}

				// TODO what if it's another struct??????
				member.Type = res
			}
		}

		parent.Members = append(parent.Members, member)

		// if quoted {
		// 	s.WriteByte('\'')
		// }
	}

	return nil
}

func (inspecter *Inspecter) InspectNodes(asts []ast.Node) error {
	var err error
	var name string

	for _, f := range asts {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.File:
				err = HandleFileComments(x.Comments, &inspecter.Comments)
			case *ast.Ident:
				name = x.Name
			case *ast.StructType:
				newStruct := Struct{
					Name: name,
				}

				err = inspecter.InspectFields(x.Fields.List, 0, &newStruct)
				if err != nil {
					return false
				}

				inspecter.Structs = append(inspecter.Structs, newStruct)

				newName := inspecter.Prefix + name + inspecter.Suffix
				inspecter.MappedTypes[name] = newName

				return false
			}
			return true
		})
	}

	return err
}

func (inspecter *Inspecter) Convert(asts []ast.Node) (string, error) {
	var err error
	inspecter.MappedTypes = make(map[string]string)

	// clean user includes for them
	for i := range inspecter.Comments.CIncludes {
		inspecter.Comments.CIncludes[i] = CleanCInclude(inspecter.Comments.CIncludes[i])
	}

	err = inspecter.InspectNodes(asts)
	if err != nil {
		return "", err
	}

	return inspecter.Converter.String(), nil

	return "", err
}
