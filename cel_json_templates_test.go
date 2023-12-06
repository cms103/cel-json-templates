package celjsontemplates_test

import (
	"encoding/json"
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
	"stringtest": "'lit'"
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
