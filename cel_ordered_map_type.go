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

func WrapOrderedCelMap(ourMap *orderedmap.OrderedMap[string, interface{}]) *OrderedCelMap {
	// mapper := types.NewDynamicMap(orderedCelMapCustomTypeAdapter{}, ourMap)

	return &OrderedCelMap{m: ourMap}
}

type OrderedCelMap struct {
	// ref.TypeAdapter
	m *orderedmap.OrderedMap[string, interface{}]
	// traits.Mapper
	// traits.Receiver
}

var (
	OrderedCelMapType = cel.ObjectType("OrderedCelMap",
		traits.ContainerType,
		traits.IndexerType,
		traits.IterableType,
		traits.SizerType,
		traits.ReceiverType)
)

func (u *OrderedCelMap) TypeName() string {
	return OrderedCelMapType.TypeName()
}

func (u OrderedCelMap) Type() ref.Type {
	return OrderedCelMapType
}

func (u *OrderedCelMap) HasTrait(trait int) bool {
	return OrderedCelMapType.HasTrait(trait)
}

func (u OrderedCelMap) Receive(function string, overload string, args []ref.Val) ref.Val {
	// if function == "Add" {
	// 	return types.Int(t.Add())
	// } else if function == "Subtract" {
	// 	return types.Int(t.Subtract())
	// }
	return types.ValOrErr(OrderedCelMapType, "no such function - %s", function)
}

// ConvertToNative implements ref.Val.ConvertToNative.
func (u OrderedCelMap) ConvertToNative(typeDesc reflect.Type) (any, error) {
	return nil, fmt.Errorf("type conversion not supported for 'type'")
}

// ConvertToType implements ref.Val.ConvertToType.
func (u OrderedCelMap) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case OrderedCelMapType:
		return OrderedCelMapType
	case types.StringType:
		return types.String(u.TypeName())
	}
	return types.NewErr("type conversion error from '%s' to '%s'", OrderedCelMapType, typeVal)
}

func (u OrderedCelMap) Equal(other ref.Val) ref.Val {
	o, ok := other.Value().(OrderedCelMap)
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

func (u *OrderedCelMap) Find(key ref.Val) (ref.Val, bool) {
	ourval, present := u.m.Get(key.Value().(string))

	return u.NativeToValue(ourval), present
}

func (u OrderedCelMap) Value() interface{} {
	return u.m
}

// Get implements the traits.Indexer interface method.
func (u *OrderedCelMap) Get(key ref.Val) ref.Val {
	v, found := u.Find(key)
	if !found {
		return types.ValOrErr(v, "no such key: %v", key)
	}
	return v
}

func (u *OrderedCelMap) Contains(value ref.Val) ref.Val {
	res, found := u.m.Get(value.Value().(string))
	if found {
		return u.NativeToValue(res)
	}
	return u.NativeToValue(nil)
}

func (u *OrderedCelMap) Iterator() traits.Iterator {

	//for pair := node.Oldest(); pair != nil; pair = pair.Next() {

	return &mapIterator{
		Adapter: types.DefaultTypeAdapter,
		mapKeys: u.m.Oldest(),
	}
}

func (u *OrderedCelMap) NativeToValue(value interface{}) ref.Val {
	val, ok := value.(OrderedCelMap)
	if ok {
		return val
	} else {
		//let the default adapter handle other cases
		return types.DefaultTypeAdapter.NativeToValue(value)
	}
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
