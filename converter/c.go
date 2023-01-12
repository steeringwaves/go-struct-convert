package converter

import (
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fatih/structtag"
)

type CConverter struct {
}

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

func (c *CConverter) GetTypeFromTags(tags *structtag.Tags) (StructMemberType, bool) {
	memberType := StructMemberType{}

	cTypeTag, err := tags.Get("ctype")
	if err == nil {
		idx := strings.Index(cTypeTag.Name, "[")
		if idx > 0 {
			memberType.Value = cTypeTag.Name[0:idx]
			memberType.Suffix = cTypeTag.Name[idx:]
		} else {
			memberType.Value = cTypeTag.Name
		}

		return memberType, true
	}

	return memberType, false
}

var CValidCNameRegexp = regexp.MustCompile(`(?m)^[\pL_][\pL\pN_]*$`)

func (c *CConverter) ValidName(n string) bool {
	return CValidCNameRegexp.MatchString(n)
}

func (c *CConverter) FileExtension() string {
	return "h"
}

func (c *CConverter) Builder(w *strings.Builder, inspecter *Inspecter) error {
	var err error
	w.WriteString("#pragma once\n\n")

	for _, include := range inspecter.Comments.CIncludes {
		w.WriteString(fmt.Sprintf("#include %s\n", include))
	}

	if len(inspecter.Comments.CIncludes) > 0 {
		w.WriteString("\n")
	}

	w.WriteString("\n")

	for _, cStruct := range inspecter.Structs {
		w.WriteString("typedef struct {\n")

		for _, member := range cStruct.Members {
			if member.Type.IsMap {
				fmt.Fprintf(os.Stderr, "WARNING! maps are unsupported, skipping %s\n", member.Name)
				continue
			}

			w.WriteString(fmt.Sprintf("%s%s%s ", inspecter.Indent, member.Type.Prefix, member.Type.Value))
			if member.Type.IsPointer {
				w.WriteString("*")
			}

			w.WriteString(fmt.Sprintf("%s%s;", member.Name, member.Type.Suffix))

			if member.Comment != "" {
				w.WriteString(fmt.Sprintf("%s// %s", inspecter.Indent, member.Comment))
			}

			w.WriteString("\n")
		}

		w.WriteString(fmt.Sprintf("} %s%s%s;\n\n", inspecter.Prefix, cStruct.Name, inspecter.Suffix))
	}

	return err
}
