package main

import (
	"fmt"

	celjsontemplates "github.com/cms103/cel-json-templates"
)

func main() {
	t, _ := celjsontemplates.New(`{"Name": "data.firstName"}`)

	res, _ := t.Expand(map[string]interface{}{"firstName": "Bob"})

	fmt.Println(string(res))
}
