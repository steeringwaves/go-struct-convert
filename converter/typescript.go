package converter

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"go/ast"

	"github.com/fatih/structtag"
)

type TypescriptConverter struct{}

var TypescriptIndent = "    "

func TypescriptGetIdent(s string) string {
	switch s {
	case "bool":
		return "boolean"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"complex64", "complex128":
		return "number"
	}

	return s
}

func TypescriptWriteType(s *strings.Builder, t ast.Expr, depth int, optionalParens bool) {
	switch t := t.(type) {
	case *ast.StarExpr:
		if optionalParens {
			s.WriteByte('(')
		}
		TypescriptWriteType(s, t.X, depth, false)
		s.WriteString(" | undefined")
		if optionalParens {
			s.WriteByte(')')
		}
	case *ast.ArrayType:
		if v, ok := t.Elt.(*ast.Ident); ok && v.String() == "byte" {
			s.WriteString("string")
			break
		}
		TypescriptWriteType(s, t.Elt, depth, true)
		s.WriteString("[]")
	case *ast.StructType:
		s.WriteString("{\n")
		TypescriptWriteFields(s, t.Fields.List, depth+1)

		for i := 0; i < depth+1; i++ {
			s.WriteString(TypescriptIndent)
		}
		s.WriteByte('}')
	case *ast.Ident:
		s.WriteString(TypescriptGetIdent(t.String()))
	case *ast.SelectorExpr:
		longType := fmt.Sprintf("%s.%s", t.X, t.Sel)
		switch longType {
		case "time.Time":
			s.WriteString("string")
		case "decimal.Decimal":
			s.WriteString("number")
		default:
			s.WriteString(longType)
		}
	case *ast.MapType:
		s.WriteString("{ [key: ")
		TypescriptWriteType(s, t.Key, depth, false)
		s.WriteString("]: ")
		TypescriptWriteType(s, t.Value, depth, false)
		s.WriteByte('}')
	case *ast.InterfaceType:
		s.WriteString("any")
	default:
		err := fmt.Errorf("unhandled: %s, %T", t, t)
		fmt.Println(err)
		panic(err)
	}
}

var TypescriptValidJSNameRegexp = regexp.MustCompile(`(?m)^[\pL_][\pL\pN_]*$`)

func TypescriptValidJSName(n string) bool {
	return TypescriptValidJSNameRegexp.MatchString(n)
}

func TypescriptWriteFields(s *strings.Builder, fields []*ast.Field, depth int) {
	for _, f := range fields {
		optional := false

		var fieldName string
		if len(f.Names) != 0 && f.Names[0] != nil && len(f.Names[0].Name) != 0 {
			fieldName = f.Names[0].Name
		}
		if len(fieldName) == 0 || 'A' > fieldName[0] || fieldName[0] > 'Z' {
			continue
		}

		var name string
		if f.Tag != nil {
			tags, err := structtag.Parse(f.Tag.Value[1 : len(f.Tag.Value)-1])
			if err != nil {
				panic(err)
			}

			jsonTag, err := tags.Get("json")
			if err == nil {
				name = jsonTag.Name
				if name == "-" {
					continue
				}

				optional = jsonTag.HasOption("omitempty")
			}
		}

		if len(name) == 0 {
			name = fieldName
		}

		for i := 0; i < depth+1; i++ {
			s.WriteString(TypescriptIndent)
		}

		quoted := !TypescriptValidJSName(name)

		if quoted {
			s.WriteByte('\'')
		}
		s.WriteString(name)
		if quoted {
			s.WriteByte('\'')
		}

		switch t := f.Type.(type) {
		case *ast.StarExpr:
			optional = true
			f.Type = t.X
		}

		if optional {
			s.WriteByte('?')
		}

		s.WriteString(": ")

		TypescriptWriteType(s, f.Type, depth, false)

		s.WriteString(";\n")
	}
}

func (ts *TypescriptConverter) Convert(w *strings.Builder, f ast.Node) error {
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

			w.WriteString("declare interface ")
			w.WriteString(name)
			w.WriteString(" {\n")

			TypescriptWriteFields(w, x.Fields.List, 0)

			w.WriteByte('}')

			first = false

			// TODO: allow multiple structs
			return false
		}
		return true
	})

	return nil
}
