package converter

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"go/ast"

	"github.com/fatih/structtag"
)

type CMember struct {
	Name       string
	Type       string
	IsPointer  bool
	TypeSuffix string
	Comment    string
}

type CStruct struct {
	Name    string
	Members []CMember
	Comment string
}

type CConverter struct {
	Prefix      string
	Suffix      string
	MappedTypes map[string]string
	Comments    Comments

	Structs []CStruct
}

var CIndent = "    "

func (c *CConverter) GetIdent(s string) string {
	switch s {
	case "byte":
		return "char"
	case "[]byte":
		return "char *"
	case "string":
		return "char *"
	case "bool":
		return "bool_t"
	case "int":
		return "int"
	case "uint":
		return "unsigned int"
	case "float32":
		return "float"
	case "float64":
		return "double"
	// case "complex64", "complex128":
	// 	return "int64_t"
	case "int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64":
		return fmt.Sprintf("%s_t", s)
	case "time.Time":
		// TODO shouldn't this just be an int64?
		return "int64_t"
	case "decimal.Decimal":
		return "double"
	case "interface", "interface{}":
		return "void *"
	}

	return s
}

var CValidCNameRegexp = regexp.MustCompile(`(?m)^[\pL_][\pL\pN_]*$`)

func (c *CConverter) ValidName(n string) bool {
	return CValidCNameRegexp.MatchString(n)
}

func (c *CConverter) FileExtension() string {
	return "h"
}

func (c *CConverter) InspectType(t ast.Expr, depth int, parent *CStruct) (string, error) {
	switch t := t.(type) {
	case *ast.ArrayType: // TODO
		if v, ok := t.Elt.(*ast.Ident); ok && v.String() == "byte" {
			return c.GetIdent("[]byte"), nil
		}
		res, err := c.InspectType(t.Elt, depth, parent)
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
		return c.GetIdent(t.String()), nil
	case *ast.SelectorExpr:
		longType := fmt.Sprintf("%s.%s", t.X, t.Sel)
		return c.GetIdent(longType), nil
	case *ast.InterfaceType:
		return c.GetIdent("interface"), nil
	default:
		return "", fmt.Errorf("unhandled: %s, %T", t, t)
	}

	return "", nil
}

func (c *CConverter) InspectFields(fields []*ast.Field, depth int, parent *CStruct) error {
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

		member := CMember{
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
				cStruct := CStruct{
					Name: name,
				}

				err := c.InspectFields(t.Fields.List, 0, &cStruct)
				if err != nil {
					return err
				}

				c.Structs = append(c.Structs, cStruct)

				newName := c.Prefix + name + c.Suffix
				c.MappedTypes[name] = newName

				member.Type = name
			default:
				res, err := c.InspectType(f.Type, depth, parent)
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

func (c *CConverter) InspectNodes(asts []ast.Node) error {
	var err error
	var name string

	for _, f := range asts {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.File:
				err = HandleFileComments(x.Comments, &c.Comments)
			case *ast.Ident:
				name = x.Name
			case *ast.StructType:
				cStruct := CStruct{
					Name: name,
				}

				err = c.InspectFields(x.Fields.List, 0, &cStruct)
				if err != nil {
					return false
				}

				c.Structs = append(c.Structs, cStruct)

				newName := c.Prefix + name + c.Suffix
				c.MappedTypes[name] = newName

				return false
			}
			return true
		})
	}

	return err
}

func (c *CConverter) String(mappedTypes map[string]string, comments Comments, structs []Struct) error {
	var err error
	w := new(strings.Builder)
	w.WriteString("#pragma once\n\n")

	for _, include := range comments.CIncludes {
		w.WriteString(fmt.Sprintf("#include %s\n", include))
	}

	if len(comments.CIncludes) > 0 {
		w.WriteString("\n")
	}

	w.WriteString("\n")

	for _, cStruct := range structs {
		w.WriteString("typedef struct {\n")

		for _, member := range cStruct.Members {

			renamed, ok := mappedTypes[member.Type]
			if !ok {
				renamed = member.Type
			}

			w.WriteString(fmt.Sprintf("%s%s ", CIndent, renamed))
			if member.IsPointer {
				w.WriteString("*")
			}

			w.WriteString(fmt.Sprintf("%s%s;", member.Name, member.TypeSuffix))

			if member.Comment != "" {
				w.WriteString(fmt.Sprintf("%s// %s", CIndent, member.Comment))
			}

			w.WriteString("\n")
		}

		w.WriteString(fmt.Sprintf("} %s%s%s;\n\n", c.Prefix, cStruct.Name, c.Suffix))
	}

	return err
}
