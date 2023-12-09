package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"

	celjsontemplates "github.com/cms103/cel-json-templates"
)

func main() {

	var exampleDir = flag.String("dir", "examples", "Directory holding the examples")
	flag.Parse()

	if flag.Lookup("help") != nil || flag.Lookup("h") != nil {
		flag.PrintDefaults()
		return
	}

	// Check we have a name
	exampleName := flag.Arg(0)
	if exampleName == "" {
		fmt.Printf("cel-json-demo exampleName (e.g. basic)\n")
		return
	}

	// Try and load an example
	e, err := LoadExample(*exampleDir, exampleName)
	if err != nil {
		return
	}

	// Expand the example
	res, err := e.template.Expand(e.input)
	if err != nil {
		fmt.Printf("Error expanding template: %v\n", err)
		return
	}

	fmt.Println(string(res))

}

type Example struct {
	Name      string
	template  celjsontemplates.Template
	input     map[string]interface{}
	reference map[string]interface{}
	fragments map[string]string
}

func LoadExample(directory string, name string) (*Example, error) {
	e := &Example{
		Name:      name,
		input:     make(map[string]interface{}),
		reference: make(map[string]interface{}),
		fragments: make(map[string]string),
	}

	var options []celjsontemplates.TemplateConfigFunc

	// See if we can load the template
	tLocation := path.Join(directory, name, "template.json")
	tContent, err := os.ReadFile(tLocation)

	if err != nil {
		fmt.Printf("Unable to load template file from %s\n", tLocation)
		return nil, err
	}

	// Load the input
	iLocation := path.Join(directory, name, "input.json")
	iContent, err := os.ReadFile(iLocation)

	if err != nil {
		fmt.Printf("Unable to load input file from %s\n", iLocation)
		return nil, err
	}

	err = json.Unmarshal(iContent, &e.input)
	if err != nil {
		fmt.Printf("Error processing input JSON from %s: %v\n", iLocation, err)
		return nil, err
	}

	// See if we also have reference data - if not that's OK
	refLocation := path.Join(directory, name, "reference.json")
	refContent, err := os.ReadFile(refLocation)

	if err == nil {
		// We have reference data to use.

		err = json.Unmarshal(refContent, &e.reference)
		if err != nil {
			fmt.Printf("Error processing input JSON from %s: %v\n", refLocation, err)
			return nil, err
		}

		options = append(options, celjsontemplates.WithRef(e.reference))

	}

	// See if we also have any fragment data - if not that's OK
	err = loadFragments(directory, name, e)
	if err != nil {
		fmt.Printf("Error processing fragments file: %v\n", err)
		return nil, err
	}

	if len(e.fragments) > 0 {
		options = append(options, celjsontemplates.WithFragments(e.fragments))
	}

	e.template, err = celjsontemplates.New(string(tContent), options...)

	if err != nil {
		fmt.Printf("Error compiling the template %s: %v\n", tLocation, err)
		return nil, err
	}
	return e, nil
}

func loadFragments(directory, name string, example *Example) error {
	fragLocation := path.Join(directory, name, "fragments.json")
	fragContent, err := os.ReadFile(fragLocation)

	if err == nil {
		// We have fragment data to use.
		var fragData map[string]interface{} = make(map[string]interface{})

		err = json.Unmarshal(fragContent, &fragData)
		if err != nil {
			fmt.Printf("Error processing input fragment data from %s: %v\n", fragLocation, err)
			return err
		}

		// Now re-encode each fragment to text so we can use our API for it.
		for key, data := range fragData {
			framData, err := json.Marshal(data)
			if err != nil {
				panic("Round trip of json failed!")
			}
			example.fragments[key] = string(framData)
		}

	}
	return nil
}

func oldExample() {

	someConfig := map[string]interface{}{
		"sub1Default": 99,
		"gameresults": map[int]string{
			1: "Lost",
			2: "Lost",
			3: "Draw",
			4: "Nearly won",
			5: "Winner!",
			6: "Best in class",
		},
	}

	simpleFragment := `{
		"testFrag": "'simple'",
		"testList": [1,2,3]
	}`

	paramFragment := `{
		"paramFram": "'With parameter'",
		"Param": "args[0]"
	}`

	gameFragment := `{
		"GameRoll": "args[0]",
		"GameResult": "ref.gameresults[args[0]]"
	}`

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
		"fragTest": "fragment('simpleFrag')",
		"paramFrag": "fragment('paramFrag', data.name).Param",
		"game1": "[1,2].map(roll, roll * 2)",
		"game2": "data.dicerolls",
		"gameresults": "data.dicerolls.map(roll, roll * 2)",
		"anothergametest": "fragment('game', data.dicerolls[2])",
		"gamewinloss": "data.dicerolls.fragment('game')",
		"mapresult": "data.dicerolls.map (roll, fragment ('game', roll))",
		"stringtest": "'lit'"
	}`, celjsontemplates.WithRef(someConfig), celjsontemplates.WithFragments(map[string]string{
		"simpleFrag": simpleFragment, "paramFrag": paramFragment, "game": gameFragment,
	}))

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
		},
		"dicerolls": [1,3,5,2,6],
		"anothervalue": "one"
	}`

	// res, err := ourT.ExpandJsonData(inputJson)

	var jdata map[string]interface{}

	json.Unmarshal([]byte(inputJson), &jdata)

	if err != nil {
		fmt.Printf("Error parsing json: %s\n", err)
		return
	}

	//jdata["dicerolls"] = types.DefaultTypeAdapter.NativeToValue([]int{1, 3, 5, 2, 6})

	res, err := ourT.Expand(jdata)

	if err != nil {
		fmt.Printf("Error expanding template: %s\n", err)
		return
	}

	fmt.Println(string(res))
}
