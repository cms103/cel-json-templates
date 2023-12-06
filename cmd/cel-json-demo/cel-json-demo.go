package main

import (
	"encoding/json"
	"fmt"

	celjsontemplates "github.com/cms103/cel-json-templates"
)

func main() {

	someConfig := map[string]interface{}{
		"sub1Default": 99,
	}

	ourT, err := celjsontemplates.New(`{
		"test": "data.test",
		"sub1": " has (data.sub1) ?  data.sub1 : ref.sub1Default",
		"sub1.2": "data.doesnotexist",
		"sub1.3": "remove_property()",
		"sub2": {
			"name": "data.name",
			"age": 44
		},
		"sub3": [1,2,3, "data.age"],
		"simpleperson": "data.person",
		"sub4": [
			{"first": "data.age"},
			{"second": 3}
		],
		"stringtest": "'lit'"
	}`, celjsontemplates.WithRef(someConfig))

	if err != nil {
		fmt.Printf("Error compiling template: %s \n", err)
		return
	}

	inputJson := `{
		"test": "avalue",
		"name": "a test name",
		"age":  40,
		"sub1": 88,
		"person": {
			"Name": "Bob",
			"Address": {
				"line1": "Härvägen 4",
				"line2": "Storstad"
			},
			"Age": 18
		}
	}`

	// res, err := ourT.ExpandJsonData(inputJson)

	var jdata map[string]interface{}

	json.Unmarshal([]byte(inputJson), &jdata)

	if err != nil {
		fmt.Printf("Error parsing json: %s\n", err)
		return
	}

	res, err := ourT.Expand(jdata)

	if err != nil {
		fmt.Printf("Error expanding template: %s\n", err)
		return
	}

	fmt.Println(string(res))

}
