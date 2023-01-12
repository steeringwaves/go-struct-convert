package converter

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/fatih/structtag"
)

type TypescriptConverter struct {
	Namespace string
}

func (ts *TypescriptConverter) GetIdent(s string) string {
	switch s {
	case "bool":
		return "boolean"
	case "interface", "interface{}":
		return "any"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"complex64", "complex128":
		return "number"
	case "time.Time":
		// TODO not sure here
		return "number"
	case "decimal.Decimal":
		return "number"
	}

	return s
}

var TypescriptValidJSNameRegexp = regexp.MustCompile(`(?m)^[\pL_][\pL\pN_]*$`)

func (ts *TypescriptConverter) ValidName(n string) bool {
	return TypescriptValidJSNameRegexp.MatchString(n)
}

func (ts *TypescriptConverter) GetTypeFromTags(tags *structtag.Tags) (StructMemberType, bool) {
	memberType := StructMemberType{}

	tsTypeTag, err := tags.Get("tstype")
	if err == nil {
		idx := strings.Index(tsTypeTag.Name, "[")
		if idx > 0 {
			memberType.Value = tsTypeTag.Name[0:idx]
			memberType.Suffix = tsTypeTag.Name[idx:]
		} else {
			memberType.Value = tsTypeTag.Name
		}

		return memberType, true
	}

	return memberType, false
}

func TypescriptValidJSName(n string) bool {
	return TypescriptValidJSNameRegexp.MatchString(n)
}

func (ts *TypescriptConverter) FileExtension() string {
	return "ts"
}

func (ts *TypescriptConverter) Builder(w *strings.Builder, inspecter *Inspecter) error {
	for _, imports := range inspecter.Comments.TypescriptImports {
		w.WriteString(fmt.Sprintf("import %s;\n", imports))
	}

	if ts.Namespace != "" {
		w.WriteString(fmt.Sprintf("namespace %s {\n", ts.Namespace))
	}

	if len(inspecter.Comments.TypescriptImports) > 0 {
		w.WriteString("\n")
	}

	var interfaceIndent string
	var interfaceMemberIndent string

	if ts.Namespace != "" {
		interfaceIndent = inspecter.Indent
		interfaceMemberIndent = inspecter.Indent + inspecter.Indent
	} else {
		interfaceIndent = ""
		interfaceMemberIndent = inspecter.Indent
	}

	for _, newStruct := range inspecter.Structs {
		w.WriteString(interfaceIndent)

		if ts.Namespace != "" {
			w.WriteString("export interface ")
		} else {
			w.WriteString("declare interface ")
		}

		w.WriteString(newStruct.Name)
		w.WriteString(" {\n")

		for _, member := range newStruct.Members {
			w.WriteString(fmt.Sprintf("%s%s", interfaceMemberIndent, member.Name))
			if member.Type.IsPointer {
				w.WriteString("?")
			}

			if member.Type.IsMap {
				w.WriteString(fmt.Sprintf(": { [key: %s%s%s", member.Type.MapKey.Prefix, member.Type.MapKey.Value, member.Type.MapKey.Suffix))

				w.WriteString(fmt.Sprintf("]: %s%s%s", member.Type.MapVal.Prefix, member.Type.MapVal.Value, member.Type.MapVal.Suffix))

				w.WriteByte('}')
				continue
			} else {

				w.WriteString(fmt.Sprintf(": %s%s%s", member.Type.Prefix, member.Type.Value, member.Type.Suffix))

				if member.Type.IsArray {
					w.WriteString("[]")
				}
			}

			w.WriteByte(';')

			if member.Comment != "" {
				w.WriteString(fmt.Sprintf("%s// %s", inspecter.Indent, member.Comment))
			}

			w.WriteString("\n")
		}

		w.WriteString(fmt.Sprintf("%s}\n\n", interfaceIndent))
	}

	if ts.Namespace != "" {
		w.WriteString("\n}\n")
		w.WriteString(fmt.Sprintf("export default %s;\n", ts.Namespace))
	} else {
		w.WriteString("\n")
	}

	return nil
}
