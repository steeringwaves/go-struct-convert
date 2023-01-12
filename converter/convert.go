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
	ValidName(n string) bool
	GetIdent(s string) string
	Builder(w *strings.Builder, inspecter *Inspecter) error
	GetTypeFromTags(tags *structtag.Tags) (StructMemberType, bool)
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
	Indent      string

	Structs []Struct
}

type StructMemberType struct {
	Value     string
	Prefix    string
	Suffix    string
	IsArray   bool
	IsPointer bool
	IsMap     bool
	MapKey    *StructMemberType
	MapVal    *StructMemberType
}

type StructMember struct {
	Name    string
	Type    StructMemberType
	Comment string
}

type Struct struct {
	Name    string
	Members []StructMember
	Comment string
}

func (inspecter *Inspecter) FileExtension() string {
	return inspecter.Converter.FileExtension()
}

func (inspecter *Inspecter) inspectTypes(t ast.Expr, depth int, parent *Struct) (StructMemberType, error) {
	structType := StructMemberType{}
	switch t := t.(type) {
	case *ast.ArrayType: // TODO
		if v, ok := t.Elt.(*ast.Ident); ok && v.String() == "byte" {
			structType.Value = inspecter.Converter.GetIdent("string")
			return structType, nil
		}
		res, err := inspecter.inspectTypes(t.Elt, depth, parent)
		if err != nil {
			return structType, err
		}
		res.IsArray = true
		return res, nil
	case *ast.Ident:
		structType.Value = inspecter.Converter.GetIdent(t.String())
		return structType, nil
	case *ast.SelectorExpr:
		longType := fmt.Sprintf("%s.%s", t.X, t.Sel)
		structType.Value = inspecter.Converter.GetIdent(longType)

		return structType, nil
	case *ast.InterfaceType:
		structType.Value = inspecter.Converter.GetIdent("interface")
		return structType, nil
	case *ast.MapType:
		res := StructMemberType{IsMap: true}

		mapKeyType, err := inspecter.inspectTypes(t.Key, depth, parent)
		if err != nil {
			return res, err
		}
		res.MapKey = &mapKeyType

		mapKeyVal, err := inspecter.inspectTypes(t.Value, depth, parent)
		if err != nil {
			return res, err
		}
		res.MapVal = &mapKeyVal
		return res, nil
	default:
		return structType, fmt.Errorf("unhandled: %s, %T", t, t)
	}

	return structType, nil
}

func (inspecter *Inspecter) inspectFields(fields []*ast.Field, depth int, parent *Struct) error {
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
		var typeFromTag StructMemberType
		var typeFromTagExists bool
		// var validator Validator
		// usingValidator := false
		if f.Tag != nil {
			tags, err := structtag.Parse(f.Tag.Value[1 : len(f.Tag.Value)-1])
			if err != nil {
				return err
			}

			typeFromTag, typeFromTagExists = inspecter.Converter.GetTypeFromTags(tags)

			// jsonTag, err := tags.Get("json")
			// if err == nil {
			// 	name = jsonTag.Name
			// 	if name == "-" {
			// 		continue
			// 	}

			// 	// optional = jsonTag.HasOption("omitempty")
			// }

			// get validator tag
			// validatorTag, err := tags.Get("validator")
			// if err == nil {
			// 	usingValidator, validator = getValidatorFromTag(validatorTag.String())
			// }
		}

		if len(name) == 0 {
			name = fieldName
		}

		if !inspecter.Converter.ValidName(name) {
			// TODO can we be smart about remove bad characters?
			fmt.Fprintf(os.Stderr, "WARNING! name is not valid %s\n", name)
			continue
		}

		isPointer := false

		switch t := f.Type.(type) {
		case *ast.StarExpr:
			f.Type = t.X
			isPointer = true
		}

		member := StructMember{
			Name:    name,
			Comment: comment,
			Type: StructMemberType{
				IsPointer: isPointer,
			},
		}

		if typeFromTagExists {
			member.Type = typeFromTag
		} else {
			switch t := f.Type.(type) {
			case *ast.StructType:
				// Nested struct, deal with it
				newStruct := Struct{
					Name: name,
				}

				err := inspecter.inspectFields(t.Fields.List, 0, &newStruct)
				if err != nil {
					return err
				}

				inspecter.Structs = append(inspecter.Structs, newStruct)

				newName := inspecter.Prefix + name + inspecter.Suffix
				inspecter.MappedTypes[name] = newName

				member.Type.Value = name
			default:
				res, err := inspecter.inspectTypes(f.Type, depth, parent)
				if err != nil {
					return err
				}

				member.Type = res
			}
		}

		parent.Members = append(parent.Members, member)

	}

	return nil
}

func (inspecter *Inspecter) inspectNodes(asts []ast.Node) error {
	var err error
	var name string

	for _, f := range asts {
		ast.Inspect(f, func(n ast.Node) bool {
			if err != nil {
				return false
			}

			switch x := n.(type) {
			case *ast.File:
				err = HandleFileComments(x.Comments, &inspecter.Comments)
			case *ast.Ident:
				name = x.Name
			case *ast.StructType:
				newStruct := Struct{
					Name: name,
				}

				err = inspecter.inspectFields(x.Fields.List, 0, &newStruct)
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

func (inspecter *Inspecter) convert(w *strings.Builder, asts []ast.Node) error {
	var err error
	inspecter.MappedTypes = make(map[string]string)

	// clean user includes for them
	for i := range inspecter.Comments.CIncludes {
		inspecter.Comments.CIncludes[i] = CleanCInclude(inspecter.Comments.CIncludes[i])
	}

	err = inspecter.inspectNodes(asts)
	if err != nil {
		return err
	}

	for i := range inspecter.Structs {
		renamed, ok := inspecter.MappedTypes[inspecter.Structs[i].Name]
		if ok {
			inspecter.Structs[i].Name = renamed
		}

		for j := range inspecter.Structs[i].Members {
			renamed, ok := inspecter.MappedTypes[inspecter.Structs[i].Members[j].Type.Value]
			if ok {
				inspecter.Structs[i].Members[j].Type.Value = renamed
			}
		}
	}

	return inspecter.Converter.Builder(w, inspecter)
}

func (inspecter *Inspecter) ConvertFiles(inputs []string) (*strings.Builder, error) {
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

		s := strings.TrimSpace(string(contents))
		if len(s) == 0 {
			return nil, errors.New("nothing to parse")
		}

		var f ast.Node
		f, err = parser.ParseExprFrom(token.NewFileSet(), filename, s, parser.AllErrors|parser.ParseComments)
		if err != nil {
			f, err = parser.ParseFile(token.NewFileSet(), filename, s, parser.AllErrors|parser.ParseComments)
			if err != nil {
				return nil, err
			}
		}

		asts = append(asts, f)
	}

	builder := new(strings.Builder)
	err := inspecter.convert(builder, asts)
	if err != nil {
		return nil, err
	}

	return builder, nil
}
