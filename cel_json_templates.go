package celjsontemplates

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// Used to communicate that this attribute should be removed from the template output
var removeAttributeFromOutput = errors.New("remove attribute")

// Template represents a CEL JSON Template
type Template interface {
	// Expand runs the CEL expressions in the template against the provided data and returns the result
	Expand(data map[string]interface{}) ([]byte, error)
}

// The structure that implements Template
type celTemplate struct {
	// ref holds the reference data
	ref map[string]interface{}
	// celOptions is the list of env options we'll use in the CEL environment
	celOptions []cel.EnvOption
	// compiledTemplate holds the compiled CEL expressions
	compiledTemplate *orderedmap.OrderedMap[string, interface{}]
	// errorOnMissingKeys flag controls whether to error if a key is not found
	errorOnMissingKeys bool
	// fragments holds the list of fragments that are available to this template
	fragments map[string]string
	// compiledFragments holds the CEL compiled fragments
	compiledFragments map[string]*orderedmap.OrderedMap[string, interface{}]
}

func (t *celTemplate) Expand(data map[string]interface{}) ([]byte, error) {
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

// ExpandJsonData will be used in the future to allow direct expansion of data
func (t *celTemplate) ExpandJsonData(data string) ([]byte, error) {
	inputJsonAsData, err := UnmarshallJson([]byte(data))
	if err != nil {
		return nil, err
	}
	//fmt.Printf("data object: %v\n", inputJsonAsData)

	input := map[string]interface{}{
		"data": inputJsonAsData,
	}

	if t.ref != nil {
		input["ref"] = t.ref
		// input.m.Set("ref", t.ref)
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

func (t *celTemplate) expandNode(input map[string]any, node *orderedmap.OrderedMap[string, interface{}]) (*orderedmap.OrderedMap[string, interface{}], error) {
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

func (t *celTemplate) expandNodeList(input map[string]interface{}, nodeList []interface{}) ([]interface{}, error) {
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

func (t *celTemplate) getFragmentsFunction() cel.EnvOption {
	ourBinding := cel.FunctionBinding(func(args ...ref.Val) ref.Val {
		// Search for the fragment name in the first argument, add additional arguments to the env then execute.
		if len(args) == 0 {
			return types.WrapErr(errors.New("fragment takes at least one argument - the name of the fragment to use"))
		}

		ct, ok := t.compiledFragments[args[0].Value().(string)]
		if !ok {
			return types.WrapErr(errors.New("fragment not found"))
		}

		var passedArgs []interface{}
		for _, ourArg := range args[1:] {
			passedArgs = append(passedArgs, ourArg.Value())
		}

		input := map[string]interface{}{
			"args": passedArgs,
		}

		if t.ref != nil {
			input["ref"] = t.ref
		}

		outputData, err := t.expandNode(input, ct)

		if err != nil {
			types.WrapErr(err)
		}

		return wrapOrderedCelMap(outputData)
		// return types.DefaultTypeAdapter.NativeToValue(outputData)
		// return orderedCelMapCustomTypeAdapter{}.NativeToValue(outputData)

	})

	listBasedBinding := cel.FunctionBinding(func(args ...ref.Val) ref.Val {
		// Search for the fragment name in the first argument, add additional arguments to the env then execute.
		if len(args) < 2 {
			return types.WrapErr(errors.New("fragment takes at least one argument - the name of the fragment to use"))
		}

		ct, ok := t.compiledFragments[args[1].Value().(string)]
		if !ok {
			return types.WrapErr(errors.New("fragment not found"))
		}

		var passedArgs []interface{}
		// Placeholder in position 0 of the passed args - will be the item
		passedArgs = append(passedArgs, nil)
		for _, ourArg := range args[2:] {
			passedArgs = append(passedArgs, ourArg.Value())
		}

		input := map[string]interface{}{
			"args": passedArgs,
		}

		if t.ref != nil {
			input["ref"] = t.ref
		}

		var resultList []interface{}
		for _, value := range args[0].Value().([]interface{}) {
			passedArgs[0] = value
			outputData, err := t.expandNode(input, ct)

			if err != nil {
				types.WrapErr(err)
			}

			resultList = append(resultList, outputData)

			// resultList = append(resultList, WrapOrderedCelMap(outputData))
		}

		return orderedCelMapAdapter.NativeToValue(resultList)
		// return types.DefaultTypeAdapter.NativeToValue(resultList)

	})

	return cel.Function("fragment",
		cel.Overload("fragment_string_dyn", []*cel.Type{cel.StringType}, cel.DynType,
			ourBinding,
		),
		cel.Overload("fragment_string_dyn_dyn", []*cel.Type{cel.StringType, cel.DynType}, cel.DynType,
			ourBinding,
		),
		cel.Overload("fragment_string_dyn_dyn_dyn", []*cel.Type{cel.StringType, cel.DynType, cel.DynType}, cel.DynType,
			ourBinding,
		),
		cel.Overload("fragment_string_dyn_dyn_dyn_dyn", []*cel.Type{cel.StringType, cel.DynType, cel.DynType, cel.DynType}, cel.DynType,
			ourBinding,
		),
		cel.Overload("fragment_string_dyn_dyn_dyn_dyn_dyn", []*cel.Type{cel.StringType, cel.DynType, cel.DynType, cel.DynType, cel.DynType}, cel.DynType,
			ourBinding,
		),
		cel.Overload("fragment_string_dyn_dyn_dyn_dyn_dyn_dyn", []*cel.Type{cel.StringType, cel.DynType, cel.DynType, cel.DynType, cel.DynType, cel.DynType}, cel.DynType,
			ourBinding,
		),
		// Now the list based overloads
		cel.MemberOverload("dyn_fragment_string_dyn", []*cel.Type{cel.DynType, cel.StringType}, cel.DynType,
			listBasedBinding,
		),
		cel.MemberOverload("dyn_fragment_string_dyn_dyn", []*cel.Type{cel.DynType, cel.StringType, cel.DynType}, cel.DynType,
			listBasedBinding,
		),
		cel.MemberOverload("dyn_fragment_string_dyn_dyn_dyn", []*cel.Type{cel.DynType, cel.StringType, cel.DynType, cel.DynType}, cel.DynType,
			listBasedBinding,
		),
		cel.MemberOverload("dyn_fragment_string_dyn_dyn_dyn_dyn", []*cel.Type{cel.DynType, cel.StringType, cel.DynType, cel.DynType, cel.DynType}, cel.DynType,
			listBasedBinding,
		),
		cel.MemberOverload("dyn_fragment_string_dyn_dyn_dyn_dyn_dyn", []*cel.Type{cel.DynType, cel.StringType, cel.DynType, cel.DynType, cel.DynType, cel.DynType}, cel.DynType,
			listBasedBinding,
		),
		cel.MemberOverload("dyn_fragment_string_dyn_dyn_dyn_dyn_dyn_dyn", []*cel.Type{cel.DynType, cel.StringType, cel.DynType, cel.DynType, cel.DynType, cel.DynType, cel.DynType}, cel.DynType,
			listBasedBinding,
		),
	)
}

// WithXXX functions provide configuration options by returning TemplateConfigFunc
type TemplateConfigFunc func(t *celTemplate)

// WithRef provides a "ref" object in the CEL environment
// This can be used to pass reference data used in the template
// For example to map values across data models
func WithRef(ref map[string]interface{}) TemplateConfigFunc {
	return func(t *celTemplate) {
		t.ref = ref

	}
}

// WithMissingKeyErrors will trigger errors when a template CEL expression refers to a missing key.
// By default such errors are suppressed
func WithMissingKeyErrors() TemplateConfigFunc {
	return func(t *celTemplate) {
		t.errorOnMissingKeys = true
	}
}

// WithCelOptions allows additional CEL EnvOptions to be used in the template.
// This can be used to add custom functions and other CEL behaviour modifications
func WithCelOptions(moreOptions []cel.EnvOption) TemplateConfigFunc {
	return func(t *celTemplate) {
		t.celOptions = append(t.celOptions, moreOptions...)
	}
}

// WithFragments registers a map of templates that can be used within this template.
func WithFragments(templates map[string]string) TemplateConfigFunc {
	return func(t *celTemplate) {
		t.fragments = templates
	}
}

// Creates a new Template using the provided input and options
func New(template string, config ...TemplateConfigFunc) (Template, error) {
	t := &celTemplate{
		ref:               make(map[string]interface{}),
		fragments:         make(map[string]string),
		compiledFragments: make(map[string]*orderedmap.OrderedMap[string, interface{}]),
	}
	for _, cfg := range config {
		cfg(t)
	}

	// Build the CEL compilation environment.
	var templateOptions []cel.EnvOption
	templateOptions = append(templateOptions, t.celOptions...)

	templateOptions = append(templateOptions, cel.Variable("ref", cel.MapType(cel.StringType, cel.DynType)))
	templateOptions = append(templateOptions, cel.Variable("data", cel.MapType(cel.StringType, cel.DynType)))
	templateOptions = append(templateOptions, getRemoveFunction())
	templateOptions = append(templateOptions, t.getFragmentsFunction())
	templateOptions = append(templateOptions, cel.CustomTypeAdapter(orderedCelMapCustomTypeAdapter{}))
	templateOptions = append(templateOptions, cel.Types(orderedCelMapType))

	env, err := cel.NewEnv(templateOptions...)

	if err != nil {
		return nil, err
	}

	// Parse the template from JSON
	t.compiledTemplate, err = parseTemplate(env, []byte(template))

	if err != nil {
		return nil, err
	}

	// Compile any fragments now
	var fragmentOptions []cel.EnvOption
	fragmentOptions = append(fragmentOptions, t.celOptions...)

	fragmentOptions = append(fragmentOptions, cel.Variable("ref", cel.MapType(cel.StringType, cel.DynType)))
	fragmentOptions = append(fragmentOptions, cel.Variable("args", cel.ListType(cel.DynType)))
	fragmentOptions = append(fragmentOptions, getRemoveFunction())
	fragmentOptions = append(fragmentOptions, cel.CustomTypeAdapter(orderedCelMapCustomTypeAdapter{}))
	fragmentOptions = append(fragmentOptions, cel.Types(orderedCelMapType))

	fragEnv, err := cel.NewEnv(fragmentOptions...)

	if err != nil {
		return nil, err
	}

	for name, fragment := range t.fragments {
		// Parse the template from JSON
		compiledFragment, err := parseTemplate(fragEnv, []byte(fragment))

		if err != nil {
			return nil, err
		}

		t.compiledFragments[name] = compiledFragment
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
				return types.WrapErr(removeAttributeFromOutput)
			}),
		),
	)
}
