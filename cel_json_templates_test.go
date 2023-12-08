package celjsontemplates_test

import (
	"encoding/json"
	"strings"
	"testing"

	celjsontemplates "github.com/cms103/cel-json-templates"
)

const referenceTemplate = `{
	"test": "data.test",
	"sub1": " has (data.sub1) ?  data.sub1 : ref.sub1Default",
	"sub1.2": "data.doesnotexist",
	"sub2": {
		"name": "data.name",
		"age": 44
	},
	"sub3": [1,2,3, "data.age"],
	"sub4": [
		{"first": "data.age"},
		{"second": 3}
	],
	"stringtest": "'lit'",
	"fragtest": "ref.fragtest ? fragment('frag1') : ''"
}`

var referenceInputData = map[string]interface{}{
	"test":   "avalue",
	"name":   "a test name",
	"age":    40,
	"sub1":   88,
	"status": 2,
	"person": map[string]interface{}{
		"Name": "Bob",
		"Age":  22,
		"Address": map[string]interface{}{
			"Line1": "Here Street",
			"Line2": "There city",
		},
	},
	"list1": []interface{}{1, 2, 3, 4, 5, 6, 7, 8, 9},
}

func TestMissingKeyErrorEnabled(t *testing.T) {
	ourT, err := celjsontemplates.New(referenceTemplate, celjsontemplates.WithMissingKeyErrors())
	if err != nil {
		t.Error(err)
	}

	_, err = ourT.Expand(referenceInputData)
	if err == nil {
		t.Error("No error on missing key")
	}
}

func TestMissingKeyErrorDefault(t *testing.T) {
	ourT, err := celjsontemplates.New(referenceTemplate)
	if err != nil {
		t.Error(err)
	}

	_, err = ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}
}

func TestRemoveProperty(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"prop1": "data.name", "prop2": "remove_property()"}`, celjsontemplates.WithMissingKeyErrors())
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}

	if string(res) != `{"prop1":"a test name"}` {
		t.Errorf("Remove property did not result in expected output: %s", string(res))
	}

}

func TestRemovePropertyFromList(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"prop1": "data.name", "prop2": ["'retain'", "remove_property()", "'retain2'"]}`, celjsontemplates.WithMissingKeyErrors())
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Errorf("Error on missing key: %s", err.Error())
	}

	if string(res) != `{"prop1":"a test name","prop2":["retain","retain2"]}` {
		t.Errorf("Remove property did not result in expected output: %s", string(res))
	}

}

func TestNestedLists(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"l1": ["'l2'", "data.test", ["'l3'", "data.age", ["'l4'", "data.name"]]]}`)
	// ourT, err := celjsontemplates.New(`{"l1": ["name", "data.test"]}`)
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}

	if string(res) != `{"l1":["l2","avalue",["l3",40,["l4","a test name"]]]}` {
		t.Errorf("Unexpected output: %s", string(res))
	}
}

func TestEmptyList(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"l1": ["missing", "alsomissing"]}`)
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}

	if string(res) != `{"l1":[]}` {
		t.Errorf("Unexpected output: %s", string(res))
	}
}

func TestRefDataLookup(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"medal": "ref.status[data.status]"}`, celjsontemplates.WithRef(map[string]interface{}{
		"status": map[int]interface{}{
			1: "Bronze",
			2: "Silver",
			3: "Gold",
		},
	}))
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}

	if string(res) != `{"medal":"Silver"}` {
		t.Errorf("Unexpected output: %s", string(res))
	}
}

func TestDataObjectOutput(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"bob": "data.person"}`)
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}

	// Unmarshall the JSON output so we can test (order is not preserved)
	var output map[string]interface{}

	err = json.Unmarshal(res, &output)
	if err != nil {
		t.Errorf("Failed to produce valid JSON: %s\n", err)
	}

	if output["bob"].(map[string]interface{})["Address"].(map[string]interface{})["Line1"] != "Here Street" {
		t.Errorf("Failed to expand json object correctly! Got map %v\n", output)
	}
}

func TestDataListOutput(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"alist": "data.list1.map(e, 'value' + string(e))", "secondlist": "data.list1"}`)
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}

	resStr := string(res)

	if !strings.Contains(resStr, "value4") {
		t.Errorf("Missing value 4 in output: %s\n", string(res))
	}
}

func TestNoArgFragmentOutput(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"alist": "fragment('frag1')"}`, celjsontemplates.WithFragments(map[string]string{
		"frag1": `{
			"Name": "'Test Name'",
			"Age": "20+22",
		}`}))
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}

	resStr := string(res)

	if !strings.Contains(resStr, `"Age":42`) {
		t.Errorf("Missing value 4 in output: %s\n", string(res))
	}
}

func TestOneArgFragmentOutput(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"alist": "fragment('frag1', 'blue')"}`, celjsontemplates.WithFragments(map[string]string{
		"frag1": `{
			"Name": "'Test Name'",
			"Age": "20+22",
			"EyeColour": "args[0]"
		}`}))
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}

	resStr := string(res)

	if !strings.Contains(resStr, `"Age":42`) {
		t.Errorf("Missing value 4 in output: %s\n", string(res))
	}

	if !strings.Contains(resStr, `"EyeColour":"blue"`) {
		t.Errorf("Missing value 4 in output: %s\n", string(res))
	}
}

func TestTwoArgFragmentOutput(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"alist": "fragment('frag1', 'blue', data.person.Name)"}`, celjsontemplates.WithFragments(map[string]string{
		"frag1": `{
			"Name": "'Test Name'",
			"Age": "20+22",
			"EyeColour": "args[0]",
			"AName": "args[1]"
		}`}))
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}

	resStr := string(res)

	if !strings.Contains(resStr, `"Age":42`) {
		t.Errorf("Missing value 4 in output: %s\n", string(res))
	}

	if !strings.Contains(resStr, `"EyeColour":"blue"`) {
		t.Errorf("Missing value 4 in output: %s\n", string(res))
	}

	if !strings.Contains(resStr, `"AName":"Bob"`) {
		t.Errorf("Missing value 4 in output: %s\n", string(res))
	}
}

func TestListNoArgFragmentOutput(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"ourresult": "data.list1.fragment('frag1')", "list": "data.list1"}`, celjsontemplates.WithFragments(map[string]string{
		"frag1": `{
			"Name": "'Test Name'",
			"Age": "args[0]",
		}`}))
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}

	resStr := string(res)

	if !strings.Contains(resStr, `"Age":1`) {
		t.Errorf("Missing value 1 in output: %s\n", string(res))
	}
}

func TestListOneArgFragmentOutput(t *testing.T) {
	ourT, err := celjsontemplates.New(`{"ourresult": "data.list1.fragment('frag1', data.person.Name)", "list": "data.list1"}`, celjsontemplates.WithFragments(map[string]string{
		"frag1": `{
			"Name": "args[1]",
			"Age": "args[0]",
		}`}))
	if err != nil {
		t.Error(err)
	}

	res, err := ourT.Expand(referenceInputData)
	if err != nil {
		t.Error("Error on missing key")
	}

	resStr := string(res)

	if !strings.Contains(resStr, `"Age":1`) {
		t.Errorf("Missing value 1 in output: %s\n", string(res))
	}
}

func BenchmarkSimpleTemplate(b *testing.B) {
	ourT, err := celjsontemplates.New(referenceTemplate)
	if err != nil {
		b.Error(err)
	}

	for i := 0; i < b.N; i++ {
		_, err = ourT.Expand(referenceInputData)
		if err != nil {
			b.Errorf("Error during benchmark: %v\n", err)
		}
	}

}

func BenchmarkFragmentTemplate(b *testing.B) {
	ourT, err := celjsontemplates.New(referenceTemplate, celjsontemplates.WithRef(map[string]interface{}{
		"fragtest": true,
	}),
		celjsontemplates.WithFragments(map[string]string{
			"frag1": `
		{
			"NestedOne": "'Value goes here'",
			"NestedList": [1,2,3,4,5]
		}`,
		}))
	if err != nil {
		b.Error(err)
	}

	for i := 0; i < b.N; i++ {
		_, err = ourT.Expand(referenceInputData)
		if err != nil {
			b.Errorf("Error during benchmark: %v\n", err)
		}
	}

}
