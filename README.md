# CEL Json Templates
This library provides a way to create JSON from templates that embed CEL expressions within them.

Templates are themselves valid JSON files and the input data is provided using a `map[string]interface{}`.

## Basic Example

An example input of
```
 {
    "firstName": "Bob",
    "donuts": 5
}
```

is processed by a template
```
{
    "Name": "data.firstName",
    "Rating": "data.donuts > 4 ? 'Donut fan' : 'Donut eater'"
}
```
to produce the output:
```
{"Name":"Bob","Rating":"Donut fan"}
```

## Additional reference data
Additional reference data can be provided when compiling the template, enabling lookups of additional information beyond that present in the input data.

An input of:
```
{
    "firstName": "Bob",
    "type": "u"
}
```

With a reference object of:
```
{
    "categories": {
        "u": "User",
        "a": "Admin",
        "s": "Superuser"
    }
}
```

Can be used by a template of:
```
{
    "Name": "data.firstName",
    "Category": "ref.categories[data.type]"
}
```

To produce:

```
{"Name":"Bob","Category":"User"}
```

## Template Fragments
Template Fragments allow templates to reuse JSON objects, either once per list item or inline within the template.

An input of:
```
{
    "name": "Bob",
    "interests": [
        "eating",
        "sleeping"
    ]
}
```

A fragment called `interestFragment` with value:
```
{
    "Activity": "'Hobby'",
    "Kind": "args[0]"
}
```

A second fragment called `summary` with a value of:
```
{
    "TotalActivities": "args[0]"
}
```

Combine with a template of:
```
{
    "Person": "data.name",
    "Interests": "data.interests.fragment ('interestFragment')",
    "Other": "fragment ('summary', data.interests.size())"
}
```

To produce output of:
```
{
    "Person": "Bob",
    "Interests": [
        {
            "Activity": "Hobby",
            "Kind": "eating"
        },
        {
            "Activity": "Hobby",
            "Kind": "sleeping"
        }
    ],
    "Other": {
        "TotalActivities": 2
    }
}
```

## Getting Started
A basic way to start using the template library is:
```
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
```

## API Options

### WithRef
Use this function to provide additional reference data under the "ref" top level name in the CEL expressions.

For example: ```celjsontemplates.New(templateData, celjsontemplate.WithRef((map[string]interface{}{"Name": "Value"}))``` would make `ref.Name` equal to `Value`.

### WithMissingKeyErrors
Normally missing keys (e.g. `data.doesNotExist`) result in the JSON attribute being silently dropped. If you'd prefer to have an error instead pass `celjsontemplate.WithMissingKeyErrors()`.

### WithFragments
This function allows a map of fragment names to fragment template strings to be passed to the template: `celjsontemplate.WithFragments(map[string]string{"FragmentName": "{}"})`

### WithCelOptions
The CEL execution environment can be modified using `celjsontemplate.WithCelOptions` to pass a list of `cel.EnvOption` values. For example to add additional string functions:
```
celjsontemplate.WithCelOptions (ext.Strings())
```

## CEL Json Template - additional Functions
The CEL execution environment provides some additional functions for use with templates.

### remove_property
Calling remove_property() will remove a property from the Json output.

### fragment
`fragment ('name', ...)` will expand the fragment called 'name', passing any further arguments in the `args` top level CEL name.

This function is also available on lists: `[1,2].fragment ('name', ...)` will expand the fragment 'name' twice, with `args[0]` containing the list element (1 then 2) and args[1] onwards containing any other arguments.

## Limitations / known issues

### Built-in CEL macros.
Some of the CEL macros don't play well with the additional data types and functions in CEL Json Templates. For example the expression `[1,2].map (val, fragment ('name', val))` results in `{"test":[{},{}]}`. Use `[1,2].fragment ('name')` instead.

