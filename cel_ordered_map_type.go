package celjsontemplates

import (
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// Wraps an OrderedMap in an OrderedCelMap so that it can be used inside CEL
func wrapOrderedCelMap(ourMap *orderedmap.OrderedMap[string, interface{}]) *orderedCelMap {
	// mapper := types.NewDynamicMap(orderedCelMapCustomTypeAdapter{}, ourMap)

	return &orderedCelMap{m: ourMap}
}

type orderedCelMap struct {
	// ref.TypeAdapter
	m *orderedmap.OrderedMap[string, interface{}]
	// traits.Mapper
	// traits.Receiver
}

var (
	orderedCelMapType = cel.ObjectType("OrderedCelMap",
		traits.ContainerType,
		traits.IndexerType,
		traits.IterableType,
		traits.SizerType,
		traits.ReceiverType)

	orderedCelMapAdapter = orderedCelMapCustomTypeAdapter{}
)

func (u *orderedCelMap) TypeName() string {
	return orderedCelMapType.TypeName()
}

func (u orderedCelMap) Type() ref.Type {
	return orderedCelMapType
}

func (u *orderedCelMap) HasTrait(trait int) bool {
	return orderedCelMapType.HasTrait(trait)
}

func (u orderedCelMap) Receive(function string, overload string, args []ref.Val) ref.Val {
	// if function == "Add" {
	// 	return types.Int(t.Add())
	// } else if function == "Subtract" {
	// 	return types.Int(t.Subtract())
	// }
	return types.ValOrErr(orderedCelMapType, "no such function - %s", function)
}

// ConvertToNative implements ref.Val.ConvertToNative.
func (u orderedCelMap) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return nil, fmt.Errorf("type conversion not supported for 'type'")
}

// ConvertToType implements ref.Val.ConvertToType.
func (u orderedCelMap) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case orderedCelMapType:
		return orderedCelMapType
	case types.StringType:
		return types.String(u.TypeName())
	}
	return types.NewErr("type conversion error from '%s' to '%s'", orderedCelMapType, typeVal)
}

func (u orderedCelMap) Equal(other ref.Val) ref.Val {
	o, ok := other.Value().(orderedCelMap)
	if ok {
		if o == u {
			return types.Bool(true)
		} else {
			return types.Bool(false)
		}
	} else {
		return types.ValOrErr(other, "%v is not of type Test", other)
	}
}

func (u *orderedCelMap) Find(key ref.Val) (ref.Val, bool) {
	ourval, present := u.m.Get(key.Value().(string))

	return u.NativeToValue(ourval), present
}

func (u orderedCelMap) Value() interface{} {
	return u.m
}

// Get implements the traits.Indexer interface method.
func (u *orderedCelMap) Get(key ref.Val) ref.Val {
	// fmt.Printf("Looking for key %v\n", key.Value())
	v, found := u.Find(key)
	if !found {
		// fmt.Printf("Found value %v for key %v\n", v.Value(), key.Value())
		return types.ValOrErr(v, "no such key: %v", key)
	}
	return v
}

func (u *orderedCelMap) Contains(value ref.Val) ref.Val {
	res, found := u.m.Get(value.Value().(string))
	if found {
		return u.NativeToValue(res)
	}
	return u.NativeToValue(nil)
}

func (u *orderedCelMap) Iterator() traits.Iterator {

	//for pair := node.Oldest(); pair != nil; pair = pair.Next() {

	return &mapIterator{
		Adapter: types.DefaultTypeAdapter,
		mapKeys: u.m.Oldest(),
	}
}

func (u *orderedCelMap) NativeToValue(value interface{}) ref.Val {
	val, ok := value.(orderedCelMap)
	if ok {
		return val
	}
	mapRef, ok := value.(*orderedmap.OrderedMap[string, any])
	if ok {
		return wrapOrderedCelMap(mapRef)
	}
	mapval, ok := value.(orderedmap.OrderedMap[string, any])
	if ok {
		return wrapOrderedCelMap(&mapval)
	}

	//let the default adapter handle other cases
	return types.DefaultTypeAdapter.NativeToValue(value)

}

type mapIterator struct {
	*baseIterator
	types.Adapter
	mapKeys *orderedmap.Pair[string, interface{}]
}

// HasNext implements the traits.Iterator interface method.
func (it *mapIterator) HasNext() ref.Val {
	return types.Bool(it.mapKeys != nil)
}

// Next implements the traits.Iterator interface method.
func (it *mapIterator) Next() ref.Val {
	toReturn := it.mapKeys
	it.mapKeys = it.mapKeys.Next()
	return it.NativeToValue(toReturn.Key)

}

// baseIterator is the basis for list, map, and object iterators.
//
// An iterator in and of itself should not be a valid value for comparison, but must implement the
// `ref.Val` methods in order to be well-supported within instruction arguments processed by the
// interpreter.
type baseIterator struct{}

func (*baseIterator) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return nil, fmt.Errorf("type conversion on iterators not supported")
}

func (*baseIterator) ConvertToType(typeVal ref.Type) ref.Val {
	return types.NewErr("no such overload")
}

func (*baseIterator) Equal(other ref.Val) ref.Val {
	return types.NewErr("no such overload")
}

func (*baseIterator) Type() ref.Type {
	return types.IteratorType
}

func (*baseIterator) Value() any {
	return nil
}

type orderedCelMapCustomTypeAdapter struct{}

func (o orderedCelMapCustomTypeAdapter) NativeToValue(value interface{}) ref.Val {
	val, ok := value.(orderedCelMap)
	if ok {
		return val
	}
	mapRef, ok := value.(*orderedmap.OrderedMap[string, any])
	if ok {
		return wrapOrderedCelMap(mapRef)
	}
	mapval, ok := value.(orderedmap.OrderedMap[string, any])
	if ok {
		return wrapOrderedCelMap(&mapval)
	}

	//let the default adapter handle other cases
	return types.DefaultTypeAdapter.NativeToValue(value)

}
