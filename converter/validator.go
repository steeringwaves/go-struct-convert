package converter

import (
	"fmt"
	"strings"
)

// TODO unused currently

// NumberValidator performs numerical value validation.
// Its limited to int type for simplicity.
type Validator struct {
	Type string
	Min  int
	Max  int
}

// Returns validator struct corresponding to validation type
func getValidatorFromTag(tag string) (bool, Validator) {
	args := strings.Split(tag, ",")

	switch args[0] {
	case "number":
		validator := Validator{Type: "number"}
		fmt.Sscanf(strings.Join(args[1:], ","), "min=%d,max=%d", &validator.Min, &validator.Max)
		return true, validator
	case "string":
		validator := Validator{Type: "string"}
		fmt.Sscanf(strings.Join(args[1:], ","), "min=%d,max=%d", &validator.Min, &validator.Max)
		return true, validator
	case "email":
		return true, Validator{Type: "email"}
	}

	return false, Validator{}
}
