package main

import (
	"encoding/json"
	"fmt"

	celjsontemplates "github.com/cms103/cel-json-templates"
)

func main() {

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
