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
	converterFunc
)

type mapperFunc func(from, to reflect.Value) error

func (m *Mapper) initStrategies() map[supportedType]mapperFunc {
	strats := make(map[supportedType]mapperFunc)
	strats[structs] = m.mapStructsFunc
	strats[slices] = m.mapSlicesFunc
	strats[arrays] = m.mapArraysFunc
	strats[sameTypes] = m.mapSameTypesFunc
	strats[sameTypes] = m.mapSameTypesFunc
	strats[converterFunc] = m.mapConverterFunc
	return strats
}

func isStructOrPtrToStruct(tp reflect.Type) bool {
	return tp.Kind() == reflect.Struct || (tp.Kind() == reflect.Ptr && tp.Elem().Kind() == reflect.Struct)
}

func (m *Mapper) detectMappingType(fromVal, toVal fieldInfo) supportedType {
	fromType := fromVal.val.Type()
	toType := toVal.val.Type()
	if _, ok := m.converters[converterInfo{from: fromVal.val.Type(), to: toVal.val.Type()}]; ok {
		return converterFunc
	}

	if toType == fromType {
		return sameTypes
	}

	if isStructOrPtrToStruct(fromType) && isStructOrPtrToStruct(toType) {
		return structs
	}

	if (fromType.Kind() == reflect.Slice && isStructOrPtrToStruct(fromType.Elem())) &&
		(toType.Kind() == reflect.Slice && isStructOrPtrToStruct(toType.Elem())) {
		return slices
	}

	if (fromType.Kind() == reflect.Array && isStructOrPtrToStruct(fromType.Elem())) &&
		(toType.Kind() == reflect.Array && isStructOrPtrToStruct(toType.Elem())) {
		return arrays
	}

	return unsupported
}

func (m *Mapper) mapStructsFunc(fromVal, toVal reflect.Value) error {
	// if from val is ptr - take Elem
	if fromVal.Kind() == reflect.Ptr {
		fromVal = fromVal.Elem()
	}

	var err error
	// if to val is ptr - set ptr to zero value
	// and pass Elem to mapper
	if toVal.Kind() == reflect.Ptr {
		toVal.Set(reflect.New(toVal.Type().Elem()))
		err = m.mapStructs(fromVal, toVal.Elem())
	} else {
		err = m.mapStructs(fromVal, toVal)
	}

	if err != nil {
		return err
	}

	return nil
}

func (m *Mapper) mapSlicesFunc(fromVal, toVal reflect.Value) error {
	slice := reflect.MakeSlice(toVal.Type(), fromVal.Len(), fromVal.Len())
	err := m.setArrayValue(fromVal, toVal, slice)
	if err != nil {
		return fmt.Errorf("error in setArrayValue: %w", err)
	}

	toVal.Set(slice)
	return nil
}

func (m *Mapper) mapArraysFunc(fromVal, toVal reflect.Value) error {
	array := reflect.New(reflect.ArrayOf(fromVal.Len(), toVal.Type().Elem())).Elem()
	err := m.setArrayValue(fromVal, toVal, array)
	if err != nil {
		return fmt.Errorf("error in setArrayValue: %w", err)
	}

	toVal.Set(array)
	return nil
}

func (m *Mapper) setArrayValue(fromVal, toVal, array reflect.Value) error {
	for i := 0; i < fromVal.Len(); i++ {
		var arrayElem reflect.Value
		// if target array's element kind is pointer - take Elem of it to get struct type
		toElemType := toVal.Type().Elem()
		if toElemType.Kind() == reflect.Ptr {
			arrayElem = reflect.New(toElemType.Elem())
		} else {
			// already a struct type
			arrayElem = reflect.New(toElemType)
		}

		fromElemType := fromVal.Type().Elem()
		var err error
		// if from array's element kind is struct - take it
		if fromElemType.Kind() == reflect.Struct {
			// take Elem of arrayElem because it's a pointer
			err = m.mapStructs(fromVal.Index(i), arrayElem.Elem())
		}
		// if from array's element kind is pointer - take Elem of it to get struct
		if fromElemType.Kind() == reflect.Ptr {
			// take Elem of arrayElem because it's a pointer
			err = m.mapStructs(fromVal.Index(i).Elem(), arrayElem.Elem())
		}

		if err != nil {
			return err
		}

		// if target array's element type is pointer - append pointer value
		if toElemType.Kind() == reflect.Ptr {
			array.Index(i).Set(arrayElem)
		} else {
			// append struct value
			array.Index(i).Set(arrayElem.Elem())
		}
	}

	return nil
}

func (m *Mapper) mapSameTypesFunc(fromVal, toVal reflect.Value) error {
	toVal.Set(fromVal)
	return nil
}

func (m *Mapper) mapConverterFunc(fromVal, toVal reflect.Value) error {
	converter, ok := m.converters[converterInfo{from: fromVal.Type(), to: toVal.Type()}]
	if !ok {
		return ErrMissingConverter
	}

	outArgs := converter.Call([]reflect.Value{fromVal})
	toVal.Set(outArgs[0])
	if len(outArgs) == 1 {
		return nil
	}

	err, ok := outArgs[1].Interface().(error)
	if !ok {
		return ErrConverterErrorUnknownType
	}

	if err != nil {
		return fmt.Errorf("%w: %v", ErrConverter, err)
	}

	return nil
}
