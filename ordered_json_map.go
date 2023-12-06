package celjsontemplates

import (
	"fmt"

	"github.com/buger/jsonparser"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func UnmarshallJson(jsonData []byte) (*orderedmap.OrderedMap[string, any], error) {
	return readJsonObject(jsonData)
}

func readJsonObject(jObj []byte) (*orderedmap.OrderedMap[string, any], error) {
	objectData := orderedmap.New[string, any]()

	err := jsonparser.ObjectEach(jObj,
		func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
			fmt.Printf("Key: '%s'\n Value: '%s'\n Type: %s\n", string(key), string(value), dataType)
			switch dataType {
			case jsonparser.Object:
				objVal, err := readJsonObject(value)
				if err != nil {
					return err
				}
				objectData.Set(string(key), objVal)
			case jsonparser.String:
				objectData.Set(string(key), string(value))
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
				listVal, err := readJsonList(value)
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

func readJsonList(jObj []byte) ([]interface{}, error) {
	var ourArray []interface{}
	var lastError error
	jsonparser.ArrayEach(jObj, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		switch dataType {
		case jsonparser.Object:
			objVal, err := readJsonObject(value)
			if err != nil {
				lastError = err
			}
			ourArray = append(ourArray, objVal)
		case jsonparser.String:
			ourArray = append(ourArray, string(value))
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
			listVal, err := readJsonList(value)
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
