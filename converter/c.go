package converter

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"go/ast"

	"github.com/fatih/structtag"
)

type CConverter struct{}

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

func CWriteType(s *strings.Builder, t ast.Expr, depth int, optionalParens bool) {
	switch t := t.(type) {
	case *ast.ArrayType: // TODO
		if v, ok := t.Elt.(*ast.Ident); ok && v.String() == "byte" {
			s.WriteString("char *")
			break
		}
		CWriteType(s, t.Elt, depth, true)
		s.WriteString("*")
	case *ast.StructType:
		s.WriteString("{\n")
		CWriteFields(s, t.Fields.List, depth+1)

		for i := 0; i < depth+1; i++ {
			s.WriteString(CIndent)
		}
		s.WriteByte('}')
	case *ast.Ident:
		s.WriteString(CGetIdent(t.String()))
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
		err := fmt.Errorf("unhandled: %s, %T", t, t)
		fmt.Println(err)
		panic(err)
	}
}

var CValidCNameRegexp = regexp.MustCompile(`(?m)^[\pL_][\pL\pN_]*$`)

func CValidCName(n string) bool {
	return CValidCNameRegexp.MatchString(n)
}

func CWriteFields(s *strings.Builder, fields []*ast.Field, depth int) {
	for _, f := range fields {
		var fieldName string
		if len(f.Names) != 0 && f.Names[0] != nil && len(f.Names[0].Name) != 0 {
			fieldName = f.Names[0].Name
		}
		if len(fieldName) == 0 || 'A' > fieldName[0] || fieldName[0] > 'Z' {
			continue
		}

		var name string
		var cType string
		// var validator Validator
		// usingValidator := false
		if f.Tag != nil {
			tags, err := structtag.Parse(f.Tag.Value[1 : len(f.Tag.Value)-1])
			if err != nil {
				panic(err)
			}

			// get ctype tag
			cTypeTag, err := tags.Get("ctype")
			if err == nil {
				cType = cTypeTag.Name
			}

			jsonTag, err := tags.Get("json")
			if err == nil {
				name = jsonTag.Name
				if name == "-" {
					continue
				}

				// optional = jsonTag.HasOption("omitempty")
			}

			// get ctype tag
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
			CWriteType(s, f.Type, depth, false)
		}

		if isPointer {
			s.WriteByte('*')
		}

		s.WriteByte(' ')

		if quoted {
			s.WriteByte('\'')
		}
		s.WriteString(name)
		if quoted {
			s.WriteByte('\'')
		}

		s.WriteString(";\n")
	}
}

func (c *CConverter) Convert(w *strings.Builder, f ast.Node) error {
	name := "MyInterface"

	first := true

	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			name = x.Name
		case *ast.StructType:
			if !first {
				w.WriteString("\n\n")
			}

			w.WriteString("typedef struct {\n")

			CWriteFields(w, x.Fields.List, 0)

			w.WriteString("} ")
			w.WriteString(name)
			w.WriteString(";\n")

			first = false

			// TODO: allow multiple structs
			return false
		}
		return true
	})

	return nil
}
