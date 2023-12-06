package celjsontemplates

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// Used to communicate that this attribute should be removed from the template output
var removeAttributeFromOutput = errors.New("remove attribute")

type Template struct {
	ref                map[string]interface{}
	celOptions         []cel.EnvOption
	compiledTemplate   *orderedmap.OrderedMap[string, interface{}]
	errorOnMissingKeys bool
}

func (t *Template) Expand(data map[string]interface{}) ([]byte, error) {
	input := map[string]interface{}{
		"data": data,
	}

	if t.ref != nil {
		input["ref"] = t.ref
	}

	outputData, err := t.expandNode(input, t.compiledTemplate)

	if err != nil {
		return nil, err
	}

	// Encode as JSON
	jdata, err := json.Marshal(outputData)

	if err != nil {
		return nil, err
	}

	return jdata, nil
}

func (t *Template) ExpandJsonData(data string) ([]byte, error) {
	inputJsonAsData, err := UnmarshallJson([]byte(data))
	if err != nil {
		return nil, err
	}
	fmt.Printf("data object: %v\n", inputJsonAsData)
	input := map[string]interface{}{
		"data": inputJsonAsData,
	}

	if t.ref != nil {
		input["ref"] = t.ref
	}

	outputData, err := t.expandNode(input, t.compiledTemplate)

	if err != nil {
		return nil, err
	}

	// Encode as JSON
	jdata, err := json.Marshal(outputData)

	if err != nil {
		return nil, err
	}

	return jdata, nil

}

func (t *Template) expandNode(input map[string]interface{}, node *orderedmap.OrderedMap[string, interface{}]) (*orderedmap.OrderedMap[string, interface{}], error) {
	// Our output data
	outputData := orderedmap.New[string, interface{}]()

	for pair := node.Oldest(); pair != nil; pair = pair.Next() {
		switch val := pair.Value.(type) {
		case cel.Program:
			// Run the program
			out, _, err := val.Eval(input)
			if err != nil {
				// This is a signal to remove the attribute
				if err.Error() == removeAttributeFromOutput.Error() {
					continue
				}
				// If there's a key missing we normally just continue
				if !(strings.Contains(err.Error(), "no such key") && t.errorOnMissingKeys) {
					continue

				} else {
					return nil, err
				}
			}

			outputData.Set(pair.Key, out.Value())
		case *orderedmap.OrderedMap[string, interface{}]:
			// Sub object - expand it
			subOutput, err := t.expandNode(input, val)
			if err != nil {
				return outputData, err
			}

			outputData.Set(pair.Key, subOutput)
		case []interface{}:
			// Expand the node list
			listOutput, err := t.expandNodeList(input, val)
			if err != nil {
				return outputData, err
			}

			outputData.Set(pair.Key, listOutput)
		default:
			outputData.Set(pair.Key, pair.Value)
		}
	}
	return outputData, nil
}

func (t *Template) expandNodeList(input map[string]interface{}, nodeList []interface{}) ([]interface{}, error) {
	// Our output data
	var outputList []interface{} = make([]interface{}, 0)

	for _, value := range nodeList {
		switch val := value.(type) {
		case cel.Program:
			// Run the program
			out, _, err := val.Eval(input)
			if err != nil {
				if err.Error() == removeAttributeFromOutput.Error() {
					// Signal is to remove this item from the list
					continue
				}
				if !(strings.Contains(err.Error(), "no such key") && t.errorOnMissingKeys) {
					continue

				} else {
					return nil, err
				}

			}
			outputList = append(outputList, out.Value())
		case *orderedmap.OrderedMap[string, interface{}]:
			// Sub object - expand it
			subOutput, err := t.expandNode(input, val)
			if err != nil {
				return outputList, err
			}
			outputList = append(outputList, subOutput)

		case []interface{}:
			// Expand the node list
			listOutput, err := t.expandNodeList(input, val)
			if err != nil {
				return listOutput, err
			}

			outputList = append(outputList, listOutput)

		default:
			outputList = append(outputList, val)
		}
	}
	return outputList, nil
}

type templateConfigFunc func(t *Template)

// WithRef provides a "ref" object in the CEL environment
// This can be used to pass reference data used in the template
// For example to map values across data models
func WithRef(ref map[string]interface{}) templateConfigFunc {
	return func(t *Template) {
		t.ref = ref

	}
}

// WithMissingKeyErrors will trigger errors when a template CEL expression refers to a missing key.
// By default such errors are suppressed
func WithMissingKeyErrors() templateConfigFunc {
	return func(t *Template) {
		t.errorOnMissingKeys = true
	}
}

// WithCelOptions allows additional CEL EnvOptions to be used in the template.
// This can be used to add custom functions and other CEL behaviour modifications
func WithCelOptions(moreOptions []cel.EnvOption) templateConfigFunc {
	return func(t *Template) {
		t.celOptions = append(t.celOptions, moreOptions...)
	}
}

func New(template string, config ...templateConfigFunc) (*Template, error) {
	t := &Template{
		ref: make(map[string]interface{}),
	}
	for _, cfg := range config {
		cfg(t)
	}

	// Build the CEL compilation environment.
	t.celOptions = append(t.celOptions, cel.Variable("ref", cel.MapType(cel.StringType, cel.DynType)))
	t.celOptions = append(t.celOptions, cel.Variable("data", cel.MapType(cel.StringType, cel.DynType)))
	t.celOptions = append(t.celOptions, getRemoveFunction())

	env, err := cel.NewEnv(t.celOptions...)

	if err != nil {
		return nil, err
	}

	// Parse the template from JSON
	t.compiledTemplate, err = parseTemplate(env, []byte(template))

	if err != nil {
		return nil, err
	}

	return t, nil
}

func parseTemplate(env *cel.Env, jsonTemplate []byte) (*orderedmap.OrderedMap[string, any], error) {
	return parseJsonObject(env, jsonTemplate)
}

func parseJsonObject(env *cel.Env, jObj []byte) (*orderedmap.OrderedMap[string, any], error) {
	objectData := orderedmap.New[string, any]()

	err := jsonparser.ObjectEach(jObj,
		func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			// fmt.Printf("Key: '%s'\n Value: '%s'\n Type: %s\n", string(key), string(value), dataType)
			switch dataType {
			case jsonparser.Object:
				objVal, err := parseJsonObject(env, value)
				if err != nil {
					return err
				}
				objectData.Set(string(key), objVal)
			case jsonparser.String:
				// We can attempt compilation
				fmt.Printf("Attempting to compile value %s\n", value)
				ast, issues := env.Compile(string(value))
				if issues != nil && issues.Err() != nil {
					return issues.Err()

				}

				prg, err := env.Program(ast)
				if err != nil {
					return err
				}

				objectData.Set(string(key), prg)
			case jsonparser.Boolean:
				bval, err := jsonparser.ParseBoolean(value)
				if err != nil {
					return err
				}
				objectData.Set(string(key), bval)
			case jsonparser.Number:
				bval, err := jsonparser.ParseFloat(value)
				if err != nil {
					return err
				}
				objectData.Set(string(key), bval)
			case jsonparser.Array:
				listVal, err := parseJsonList(env, value)
				if err != nil {
					return err
				}
				objectData.Set(string(key), listVal)

			default:
				objectData.Set(string(key), value)

			}
			return nil
		})

	if err != nil {
		return nil, err
	}
	return objectData, nil
}

func parseJsonList(env *cel.Env, jObj []byte) ([]interface{}, error) {
	var ourArray []interface{}
	var lastError error
	jsonparser.ArrayEach(jObj, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		switch dataType {
		case jsonparser.Object:
			objVal, err := parseJsonObject(env, value)
			if err != nil {
				lastError = err
			}
			ourArray = append(ourArray, objVal)
		case jsonparser.String:
			// We can attempt compilation
			ast, issues := env.Compile(string(value))
			if issues != nil && issues.Err() != nil {
				return

			}

			prg, err := env.Program(ast)
			if err != nil {
				return
			}

			ourArray = append(ourArray, prg)
		case jsonparser.Boolean:
			bval, err := jsonparser.ParseBoolean(value)
			if err != nil {
				return
			}
			ourArray = append(ourArray, bval)
		case jsonparser.Number:
			bval, err := jsonparser.ParseFloat(value)
			if err != nil {
				return
			}
			ourArray = append(ourArray, bval)
		case jsonparser.Array:
			listVal, err := parseJsonList(env, value)
			if err != nil {
				lastError = err
			}
			ourArray = append(ourArray, listVal)
		default:
			ourArray = append(ourArray, value)
		}
	})

	return ourArray, lastError
}

func getRemoveFunction() cel.EnvOption {
	return cel.Function("remove_property",
		cel.Overload("remove_property_dyn", []*cel.Type{}, cel.DynType,
			cel.FunctionBinding(func(args ...ref.Val) ref.Val {
				fmt.Printf("MADE IT HERE")
				return types.WrapErr(removeAttributeFromOutput)
			}),
		),
	)
}
