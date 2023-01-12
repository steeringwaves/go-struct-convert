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
	TypeSuffix string
}

type CStruct struct {
	Name    string
	Members []CMember
}

type CConverter struct {
	Prefix      string
	Suffix      string
	MappedTypes map[string]string
	Comments    Comments

	Structs []CStruct
}

var CIndent = "    "

func CGetIdent(s string) string {
	switch s {
	case "byte":
		return "char"
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
	}

	return s
}

func (c *CConverter) CWriteType(s *strings.Builder, t ast.Expr, depth int, optionalParens bool) error {
	switch t := t.(type) {
	case *ast.ArrayType: // TODO
		if v, ok := t.Elt.(*ast.Ident); ok && v.String() == "byte" {
			s.WriteString("char *")
			break
		}
		err := c.CWriteType(s, t.Elt, depth, true)
		if err != nil {
			return err
		}
		s.WriteString("*")
	case *ast.StructType:
		s.WriteString("{\n")
		err := c.CWriteFields(s, t.Fields.List, depth+1)
		if err != nil {
			return err
		}

		for i := 0; i < depth+1; i++ {
			s.WriteString(CIndent)
		}
		s.WriteByte('}')
	case *ast.Ident:
		renamed, ok := c.MappedTypes[t.String()]
		if ok {
			s.WriteString(CGetIdent(renamed))
		} else {
			s.WriteString(CGetIdent(t.String()))
		}
	case *ast.SelectorExpr:
		longType := fmt.Sprintf("%s.%s", t.X, t.Sel)
		switch longType {
		case "time.Time":
			s.WriteString("char *") // TODO
		case "decimal.Decimal":
			s.WriteString("double")
		default:
			s.WriteString(longType)
		}
	case *ast.InterfaceType:
		s.WriteString("void *")
	default:
		return fmt.Errorf("unhandled: %s, %T", t, t)
	}

	return nil
}

var CValidCNameRegexp = regexp.MustCompile(`(?m)^[\pL_][\pL\pN_]*$`)

func CValidCName(n string) bool {
	return CValidCNameRegexp.MatchString(n)
}

func (c *CConverter) CWriteFields(s *strings.Builder, fields []*ast.Field, depth int) error {
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

		for i := 0; i < depth+1; i++ {
			s.WriteString(CIndent)
		}

		quoted := !CValidCName(name)
		isPointer := false

		switch t := f.Type.(type) {
		case *ast.StarExpr:
			f.Type = t.X
			isPointer = true
		}

		if cType != "" {
			s.WriteString(cType)
		} else {
			err := c.CWriteType(s, f.Type, depth, false)
			if err != nil {
				return err
			}
		}

		if isPointer {
			s.WriteByte('*')
		}

		s.WriteByte(' ')

		if quoted {
			s.WriteByte('\'')
		}
		s.WriteString(name)

		if cTypeSuffix != "" {
			s.WriteString(cTypeSuffix)
		}
		if quoted {
			s.WriteByte('\'')
		}

		s.WriteString(";")

		if comment != "" {
			s.WriteString(fmt.Sprintf("	// %s", comment))
		}

		s.WriteString("\n")
	}

	return nil
}

func (c *CConverter) FileExtension() string {
	return "h"
}

func (c *CConverter) Convert(w *strings.Builder, asts []ast.Node) error {
	var err error
	name := "MyInterface"

	c.MappedTypes = make(map[string]string)

	first := true

	builder := new(strings.Builder)

	// clean user includes for them
	for i := range c.Comments.CIncludes {
		c.Comments.CIncludes[i] = CleanCInclude(c.Comments.CIncludes[i])
	}

	for _, f := range asts {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.File:
				err = HandleFileComments(x.Comments, &c.Comments)
			case *ast.Ident:
				name = x.Name
			case *ast.StructType:
				if !first {
					builder.WriteString("\n\n")
				}

				builder.WriteString("typedef struct {\n")

				err = c.CWriteFields(builder, x.Fields.List, 0)
				if err != nil {
					return false
				}

				builder.WriteString("} ")

				newName := c.Prefix + name + c.Suffix
				c.MappedTypes[name] = newName
				builder.WriteString(newName)

				builder.WriteString(";\n")

				first = false

				// TODO: allow multiple structs
				return false
			}
			return true
		})
	}

	w.WriteString("#pragma once\n\n")

	for _, include := range c.Comments.CIncludes {
		w.WriteString(fmt.Sprintf("#include %s\n", include))
	}

	if len(c.Comments.CIncludes) > 0 {
		w.WriteString("\n")
	}

	w.WriteString("\n")
	w.WriteString(builder.String())

	return err
}
