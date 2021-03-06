package transform

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PaesslerAG/jsonpath"
)

type transformMethod int32

const (
	first transformMethod = iota
	last
	concatenate
)

// transformOperation defines the interface for operations that are implemented within the transform schema.
type transformOperation interface {
	init(args map[string]string) error
	transform(in interface{}) (interface{}, error)
}

type transformOperationJSON struct {
	Name string            `json:"type"`
	Args map[string]string `json:"args"`
}

// transformInstruction defines a JSONPath for a transform and an optional set of operations to be performed on the
// data from that path.
type transformInstruction struct {
	// For JSONPath format see http://goessner.net/articles/JsonPath/
	JsonPath   string               `json:"jsonPath"`
	Operations []transformOperation `json:"operations"`
}

type transformInstructionJSON struct {
	JsonPath   string                   `json:"jsonPath"`
	Operations []transformOperationJSON `json:"operations"`
}

// UnmarshalJSON implements the json.Unmarshaler interface, this function exists to properly map the transformOperation.
func (ti *transformInstruction) UnmarshalJSON(data []byte) error {
	var jti transformInstructionJSON

	if err := json.Unmarshal(data, &jti); err != nil {
		return fmt.Errorf("failed to extract transform from JSON: %v", err)
	}

	ti.JsonPath = jti.JsonPath
	ti.Operations = []transformOperation{}

	for _, toj := range jti.Operations {
		var op transformOperation
		switch toj.Name {
		case "changeCase":
			op = &changeCase{}
		case "duration":
			op = &duration{}
		case "inverse":
			op = &inverse{}
		case "max":
			op = &max{}
		case "replace":
			op = &replace{}
		case "split":
			op = &split{}
		default:
			return fmt.Errorf("unsupported operation %q", toj.Name)
		}

		if err := op.init(toj.Args); err != nil {
			return fmt.Errorf("failed initializing transform operation: %v", err)
		}
		ti.Operations = append(ti.Operations, op)
	}
	return nil
}

// transform runs the instructions in this object returning the new transformed value or an error if unable to.
// It handles the logic for finding the value to be transformed and chaining the Operations.
// It will not error if the value is not found, rather it returns nil for the value.
// If a conversion or operation fails an error is returned.
func (ti *transformInstruction) transform(in interface{}, fieldType string) (interface{}, error) {
	rawValue, err := jsonpath.Get(ti.JsonPath, in)
	if err != nil {
		return nil, nil
	}
	if rawValue == nil {
		return nil, nil
	}

	value, err := convert(rawValue, fieldType)
	if err != nil {
		// In some cases the conversion is helpful but in others like before a max operation it isn't
		value = rawValue
	}
	if value == nil {
		return nil, nil
	}

	for _, op := range ti.Operations {
		value, err = op.transform(value)
		if err != nil {
			return nil, fmt.Errorf("failed operation on value from JSONPath %q: %v", ti.JsonPath, err)
		}
	}
	return value, nil
}

// trransformInstructions defines a set of instructions and a method for combining their results.
// The default method is to take the first non-nil result.
type transformInstructions struct {
	From   []transformInstruction `json:"from"`
	Method transformMethod        `json:"method"`
}

type transformInstructionsJSON struct {
	From   []transformInstruction `json:"from"`
	Method string                 `json:"method"`
}

// UnmarshalJSON implements the json.Unmarshaler interface, this function exists to properly map the method.
func (tis *transformInstructions) UnmarshalJSON(data []byte) error {
	var jtis transformInstructionsJSON

	if err := json.Unmarshal(data, &jtis); err != nil {
		return fmt.Errorf("failed to extract transform from JSON: %v", err)
	}

	tis.From = jtis.From

	switch jtis.Method {
	case "":
		tis.Method = 0
	case "first":
		tis.Method = first
	case "last":
		tis.Method = last
	case "concatenate":
		tis.Method = concatenate
	default:
		return fmt.Errorf("unknown method %q", jtis.Method)
	}

	return nil
}

// transform runs the instructions in this object returning the new transformed value or nil if none is found.
// It handles the logic for concatenation, first or last methods.
func (tis *transformInstructions) transform(in interface{}, fieldType string) (interface{}, error) {
	var concatResult bool
	switch tis.Method {
	case last:
		var newFrom []transformInstruction
		for i := len(tis.From) - 1; i >= 0; i-- {
			newFrom = append(newFrom, tis.From[i])
		}
		tis.From = newFrom
	case concatenate:
		concatResult = true
	}

	var result interface{}

	for _, from := range tis.From {
		value, err := from.transform(in, fieldType)
		if err != nil {
			return nil, err
		}
		if concatResult {
			result, err = concat(result, value)
			if err != nil {
				return nil, fmt.Errorf("failed to concat values: %v", err)
			}
			continue
		}
		if value != nil {
			result = value
			break
		}
	}

	return result, nil
}

type transform map[string]transformInstructions

// replaceJSONPath will return a new transform after replacing the string old for new in any JSONPaths within the
// instructions.
func (t *transform) replaceJSONPath(old, new string) transform {
	newTR := transform{}
	for id, instructions := range *t {
		newInstructions := transformInstructions{Method: instructions.Method, From: []transformInstruction{}}
		for _, instruction := range instructions.From {
			newPath := strings.Replace(instruction.JsonPath, old, new, 1)
			newInstructions.From = append(newInstructions.From,
				transformInstruction{
					JsonPath:   newPath,
					Operations: instruction.Operations,
				})
		}
		newTR[id] = newInstructions
	}
	return newTR
}
