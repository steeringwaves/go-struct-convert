package converter

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"go/ast"

	"github.com/fatih/structtag"
)

type TypescriptConverter struct {
	Namespace   string
	Prefix      string
	Suffix      string
	MappedTypes map[string]string

	Imports []string
}

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

func (ts *TypescriptConverter) TypescriptWriteType(s *strings.Builder, t ast.Expr, depth int, optionalParens bool) error {
	switch t := t.(type) {
	case *ast.StarExpr:
		if optionalParens {
			s.WriteByte('(')
		}
		err := ts.TypescriptWriteType(s, t.X, depth, false)
		if err != nil {
			return err
		}
		s.WriteString(" | undefined")
		if optionalParens {
			s.WriteByte(')')
		}
	case *ast.ArrayType:
		if v, ok := t.Elt.(*ast.Ident); ok && v.String() == "byte" {
			s.WriteString("string")
			break
		}
		err := ts.TypescriptWriteType(s, t.Elt, depth, true)
		if err != nil {
			return err
		}
		s.WriteString("[]")
	case *ast.StructType:
		s.WriteString("{\n")
		ts.TypescriptWriteFields(s, t.Fields.List, depth+1)

		for i := 0; i < depth+1; i++ {
			s.WriteString(TypescriptIndent)
		}
		s.WriteByte('}')
	case *ast.Ident:
		renamed, ok := ts.MappedTypes[t.String()]
		if ok {
			s.WriteString(TypescriptGetIdent(renamed))
		} else {
			s.WriteString(TypescriptGetIdent(t.String()))
		}
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
		err := ts.TypescriptWriteType(s, t.Key, depth, false)
		if err != nil {
			return err
		}
		s.WriteString("]: ")
		err = ts.TypescriptWriteType(s, t.Value, depth, false)
		if err != nil {
			return err
		}
		s.WriteByte('}')
	case *ast.InterfaceType:
		s.WriteString("any")
	default:
		return fmt.Errorf("unhandled: %s, %T", t, t)
	}

	return nil
}

var TypescriptValidJSNameRegexp = regexp.MustCompile(`(?m)^[\pL_][\pL\pN_]*$`)

func TypescriptValidJSName(n string) bool {
	return TypescriptValidJSNameRegexp.MatchString(n)
}

func (ts *TypescriptConverter) TypescriptWriteFields(s *strings.Builder, fields []*ast.Field, depth int) error {
	for _, f := range fields {
		optional := false

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
		if f.Tag != nil {
			tags, err := structtag.Parse(f.Tag.Value[1 : len(f.Tag.Value)-1])
			if err != nil {
				return err
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

		err := ts.TypescriptWriteType(s, f.Type, depth, false)
		if err != nil {
			return err
		}

		s.WriteString(";")

		if comment != "" {
			s.WriteString(fmt.Sprintf("	// %s", comment))
		}

		s.WriteString("\n")
	}

	return nil
}

func (ts *TypescriptConverter) FileExtension() string {
	return "ts"
}

var tsImportTag string = "ts.import"

func (ts *TypescriptConverter) HandleFileComments(comments []*ast.CommentGroup) error {
	for _, comment := range comments {
		lines := strings.Split(comment.Text(), "\n")
		for _, line := range lines {
			idx := strings.LastIndex(line, tsImportTag)
			if idx >= 0 {
				imports := strings.TrimSpace(line[idx+len(tsImportTag):])
				ts.Imports = append(ts.Imports, imports)
			}
		}
	}

	return nil
}

func (ts *TypescriptConverter) Convert(w *strings.Builder, f ast.Node) error {
	var err error
	name := "MyInterface"

	ts.MappedTypes = make(map[string]string)

	builder := new(strings.Builder)

	first := true

	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.File:
			err = ts.HandleFileComments(x.Comments)
		case *ast.Ident:
			name = x.Name
		case *ast.StructType:
			if !first {
				builder.WriteString("\n\n")
			}

			depth := 0
			if ts.Namespace != "" {
				depth++
			}

			for i := 0; i < depth; i++ {
				builder.WriteString(TypescriptIndent)
			}

			if ts.Namespace != "" {
				builder.WriteString("export interface ")
			} else {
				builder.WriteString("declare interface ")
			}

			newName := ts.Prefix + name + ts.Suffix
			ts.MappedTypes[name] = newName
			builder.WriteString(newName)
			builder.WriteString(" {\n")

			err = ts.TypescriptWriteFields(builder, x.Fields.List, depth)
			if err != nil {
				return false
			}

			for i := 0; i < depth; i++ {
				builder.WriteString(TypescriptIndent)
			}
			builder.WriteByte('}')

			first = false

			// TODO: allow multiple structs
			return false
		}
		return true
	})

	for _, imports := range ts.Imports {
		w.WriteString(fmt.Sprintf("%s\n", imports))
	}

	if ts.Namespace != "" {
		w.WriteString(fmt.Sprintf("namespace %s {\n", ts.Namespace))
	}

	if len(ts.Imports) > 0 {
		w.WriteString("\n")
	}

	w.WriteString(builder.String())

	if ts.Namespace != "" {
		w.WriteString("\n}\n")
		w.WriteString(fmt.Sprintf("export default %s;\n", ts.Namespace))
	} else {
		w.WriteString("\n")
	}

	return nil
}
