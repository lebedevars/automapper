package automapper

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrNotAPtr          = errors.New("value is not a ptr")
	ErrNotAFn           = errors.New("value is not a function")
	ErrMissingConverter = errors.New("converter is missing for types")
	ErrConverter        = errors.New("converter error")
)

type converterInfo struct {
	from, to reflect.Type
}

// Mapper maps struct values.
type Mapper struct {
	converters map[converterInfo]reflect.Value
	strats     map[supportedType]mapperFunc
}

type fieldInfo struct {
	tp  reflect.Type
	val reflect.Value
}

// New returns new Mapper.
func New() *Mapper {
	m := &Mapper{converters: make(map[converterInfo]reflect.Value)}
	m.strats = m.initStrategies()
	return m
}

// Set sets converter function.
// Converter function must be in one of two forms:
//  func(in int) string
//  func(in string) (int, error)
// Set will make the Mapper use the converter function to map in-type to out-type
// every time the Mapper comes across one.
func (m *Mapper) Set(converter interface{}) error {
	fn := reflect.TypeOf(converter)
	if fn.Kind() != reflect.Func {
		return ErrNotAFn
	}

	m.converters[converterInfo{from: fn.In(0), to: fn.Out(0)}] = reflect.ValueOf(converter)
	return nil
}

// Map maps two structs or two slices of structs.
func (m *Mapper) Map(from, to interface{}) error {
	valFrom := reflect.ValueOf(from)
	valTo := reflect.ValueOf(to)

	if valFrom.Kind() != reflect.Ptr || valTo.Kind() != reflect.Ptr {
		return ErrNotAPtr
	}

	return m.mapStructs(valFrom.Elem(), valTo.Elem())
}

// from, to must be struct values.
func (m *Mapper) mapStructs(from, to reflect.Value) error {
	if !from.IsValid() {
		return nil
	}

	fromFields := make(map[string]fieldInfo)
	for i := 0; i < from.NumField(); i++ {
		// skip zero or nil values
		if from.Field(i).IsZero() || (from.Field(i).Kind() == reflect.Ptr && from.Field(i).IsNil()) {
			continue
		}

		field := from.Field(i)
		t := from.Type().Field(i).Type
		fromFields[from.Type().Field(i).Name] = fieldInfo{
			tp:  t,
			val: field,
		}
	}

	toFields := make(map[string]fieldInfo)
	for i := 0; i < to.NumField(); i++ {
		if !to.Field(i).CanSet() {
			continue
		}

		field := to.Field(i)
		t := to.Type().Field(i).Type
		toFields[to.Type().Field(i).Name] = fieldInfo{
			tp:  t,
			val: field,
		}
	}

	for name, fromVal := range fromFields {
		toVal, ok := toFields[name]
		if !ok {
			continue
		}

		converter, ok := m.converters[converterInfo{from: fromVal.tp, to: toVal.val.Type()}]
		if ok {
			outArgs := converter.Call([]reflect.Value{fromVal.val})
			if len(outArgs) == 1 {
				toVal.val.Set(outArgs[0])
				return nil
			}

			err := outArgs[1].Interface().(error)
			if err != nil {
				return fmt.Errorf("%w, field '%s': %v", ErrConverter, name, err)
			}
		}

		mappingType := detectMappingType(fromVal, toVal)
		if mappingType != unsupported {
			err := m.strats[mappingType](fromVal, toVal)
			if err != nil {
				return err
			}

			continue
		}

		return fmt.Errorf("%w '%s -> %s'", ErrMissingConverter, fromVal.val.Type(), toVal.val.Type())
	}

	return nil
}
