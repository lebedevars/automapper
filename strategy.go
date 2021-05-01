// Package automapper contains Mapper used to map two structs or slices automatically.
package automapper

import (
	"fmt"
	"reflect"
)

type supportedType int

const (
	unsupported supportedType = iota
	structs
	slices
	arrays
	sameTypes
)

type mapperFunc func(from, to fieldInfo) error

func (m *Mapper) initStrategies() map[supportedType]mapperFunc {
	strats := make(map[supportedType]mapperFunc)
	strats[structs] = m.mapStructsFunc
	strats[slices] = m.mapSlicesFunc
	strats[arrays] = m.mapArraysFunc
	strats[sameTypes] = m.mapSameTypesFunc
	return strats
}

func isStructOrPtrToStruct(tp reflect.Type) bool {
	return tp.Kind() == reflect.Struct || (tp.Kind() == reflect.Ptr && tp.Elem().Kind() == reflect.Struct)
}

func detectMappingType(fromVal, toVal fieldInfo) supportedType {
	if toVal.val.Type() == fromVal.val.Type() {
		return sameTypes
	}

	if isStructOrPtrToStruct(fromVal.tp) && isStructOrPtrToStruct(toVal.tp) {
		return structs
	}

	if (fromVal.tp.Kind() == reflect.Slice && isStructOrPtrToStruct(fromVal.tp.Elem())) &&
		(toVal.tp.Kind() == reflect.Slice && isStructOrPtrToStruct(toVal.tp.Elem())) {
		return slices
	}

	if (fromVal.tp.Kind() == reflect.Array && isStructOrPtrToStruct(fromVal.tp.Elem())) &&
		(toVal.tp.Kind() == reflect.Array && isStructOrPtrToStruct(toVal.tp.Elem())) {
		return arrays
	}

	return unsupported
}

func (m *Mapper) mapStructsFunc(fromVal, toVal fieldInfo) error {
	// if from val is ptr - take Elem
	if fromVal.val.Kind() == reflect.Ptr {
		fromVal.val = fromVal.val.Elem()
	}

	var err error
	// if to val is ptr - set ptr to zero value
	// and pass Elem to mapper
	if toVal.val.Kind() == reflect.Ptr {
		toVal.val.Set(reflect.New(toVal.tp.Elem()))
		err = m.mapStructs(fromVal.val, toVal.val.Elem())
	} else {
		err = m.mapStructs(fromVal.val, toVal.val)
	}

	if err != nil {
		return err
	}

	return nil
}

func (m *Mapper) mapSlicesFunc(fromVal, toVal fieldInfo) error {
	slice := reflect.MakeSlice(toVal.tp, fromVal.val.Len(), fromVal.val.Len())
	err := m.setArrayValue(fromVal, toVal, slice)
	if err != nil {
		return fmt.Errorf("error in setArrayValue: %w", err)
	}

	toVal.val.Set(slice)
	return nil
}

func (m *Mapper) mapArraysFunc(fromVal, toVal fieldInfo) error {
	array := reflect.New(reflect.ArrayOf(fromVal.val.Len(), toVal.tp.Elem())).Elem()
	err := m.setArrayValue(fromVal, toVal, array)
	if err != nil {
		return fmt.Errorf("error in setArrayValue: %w", err)
	}

	toVal.val.Set(array)
	return nil
}

func (m *Mapper) setArrayValue(fromVal, toVal fieldInfo, array reflect.Value) error {
	for i := 0; i < fromVal.val.Len(); i++ {
		var arrayElem reflect.Value
		// if target array's element kind is pointer - take Elem of it to get struct type
		if toVal.tp.Elem().Kind() == reflect.Ptr {
			arrayElem = reflect.New(toVal.tp.Elem().Elem())
		} else {
			// already a struct type
			arrayElem = reflect.New(toVal.tp.Elem())
		}

		var err error
		// if from array's element kind is struct - take it
		if fromVal.tp.Elem().Kind() == reflect.Struct {
			// take Elem of arrayElem because it's a pointer
			err = m.mapStructs(fromVal.val.Index(i), arrayElem.Elem())
		}
		// if from array's element kind is pointer - take Elem of it to get struct
		if fromVal.tp.Elem().Kind() == reflect.Ptr {
			// take Elem of arrayElem because it's a pointer
			err = m.mapStructs(fromVal.val.Index(i).Elem(), arrayElem.Elem())
		}

		if err != nil {
			return err
		}

		// if target array's element type is pointer - append pointer value
		if toVal.tp.Elem().Kind() == reflect.Ptr {
			array.Index(i).Set(arrayElem)
		} else {
			// append struct value
			array.Index(i).Set(arrayElem.Elem())
		}
	}

	return nil
}

func (m *Mapper) mapSameTypesFunc(fromVal, toVal fieldInfo) error {
	toVal.val.Set(fromVal.val)
	return nil
}
