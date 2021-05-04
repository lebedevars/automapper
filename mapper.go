package automapper

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

var (
	ErrNotAPtr                   = errors.New("value is not a ptr")
	ErrNotAFn                    = errors.New("value is not a function")
	ErrMissingConverter          = errors.New("converter is missing for types")
	ErrConverter                 = errors.New("converter error")
	ErrConverterErrorUnknownType = errors.New("converter 2nd return value cannot be converted to error")
)

type converterInfo struct {
	from, to reflect.Type
}

type structMappingInfo struct {
	from, to reflect.Type
}

type fieldMappingInfo struct {
	fromIndex, toIndex int
	mapperFunc         mapperFunc
}

// Mapper maps struct values.
type Mapper struct {
	mu            sync.Mutex
	converters    map[converterInfo]reflect.Value
	strats        map[supportedType]mapperFunc
	knownMappings map[structMappingInfo][]fieldMappingInfo
}

type fieldInfo struct {
	index int
	val   reflect.Value
}

// New returns new Mapper.
func New() *Mapper {
	m := &Mapper{
		mu:            sync.Mutex{},
		converters:    make(map[converterInfo]reflect.Value),
		knownMappings: make(map[structMappingInfo][]fieldMappingInfo),
	}
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
	typeFrom := reflect.TypeOf(from)
	typeTo := reflect.TypeOf(to)
	valFrom := reflect.ValueOf(from)
	valTo := reflect.ValueOf(to)

	if (typeFrom.Kind() == reflect.Ptr && typeFrom.Elem().Kind() == reflect.Slice && isStructOrPtrToStruct(typeFrom.Elem().Elem())) &&
		(typeTo.Kind() == reflect.Ptr && typeTo.Elem().Kind() == reflect.Slice && isStructOrPtrToStruct(typeTo.Elem().Elem())) {
		return m.mapSlicesFunc(valFrom.Elem(), valTo.Elem())
	}

	if isStructOrPtrToStruct(typeFrom) && isStructOrPtrToStruct(typeTo) {
		return m.mapStructs(valFrom.Elem(), valTo.Elem())
	}

	return nil
}

// from, to must be struct values.
func (m *Mapper) mapStructs(from, to reflect.Value) error {
	if !from.IsValid() {
		return nil
	}

	m.mu.Lock()
	mappingInfo := structMappingInfo{from: from.Type(), to: to.Type()}
	if knownMapping, ok := m.knownMappings[mappingInfo]; ok {
		err := m.mapKnownStruct(knownMapping, from, to)
		if err != nil {
			return err
		}
	}

	m.knownMappings[mappingInfo] = make([]fieldMappingInfo, 0)
	m.mu.Unlock()

	fromFields, toFields := getFieldInfo(from, to)
	for name, fromVal := range fromFields {
		toVal, ok := toFields[name]
		if !ok {
			continue
		}

		mappingType := m.detectMappingType(fromVal, toVal)
		if mappingType != unsupported {
			err := m.strats[mappingType](fromVal.val, toVal.val)
			if err != nil {
				return err
			}

			m.mu.Lock()
			m.knownMappings[mappingInfo] = append(m.knownMappings[mappingInfo], fieldMappingInfo{
				fromIndex:  fromVal.index,
				toIndex:    toVal.index,
				mapperFunc: m.strats[mappingType],
			})
			m.mu.Unlock()

			continue
		}

		return fmt.Errorf("%w '%s -> %s'", ErrMissingConverter, fromVal.val.Type(), toVal.val.Type())
	}

	return nil
}

func getFieldInfo(from, to reflect.Value) (fromFields, toFields map[string]fieldInfo) {
	fromFields = make(map[string]fieldInfo)
	toFields = make(map[string]fieldInfo)
	for i := 0; i < from.NumField(); i++ {
		// skip zero or nil values
		fieldVal := from.Field(i)
		if fieldVal.IsZero() || (fieldVal.Kind() == reflect.Ptr && fieldVal.IsNil()) {
			continue
		}

		fieldType := from.Type().Field(i)
		name := fieldType.Name
		mapperTag, ok := fieldType.Tag.Lookup("mapper")
		if ok && mapperTag != "" {
			name = mapperTag
		}

		fromFields[name] = fieldInfo{
			index: i,
			val:   fieldVal,
		}
	}

	for i := 0; i < to.NumField(); i++ {
		fieldVal := to.Field(i)
		if !fieldVal.CanSet() {
			continue
		}

		fieldType := to.Type().Field(i)
		name := fieldType.Name
		mapperTag, ok := fieldType.Tag.Lookup("mapper")
		if ok && mapperTag != "" {
			name = mapperTag
		}

		toFields[name] = fieldInfo{
			index: i,
			val:   fieldVal,
		}
	}

	return fromFields, toFields
}

func (m *Mapper) mapKnownStruct(mappingInfo []fieldMappingInfo, from, to reflect.Value) error {
	for _, fieldInfo := range mappingInfo {
		err := fieldInfo.mapperFunc(from.Field(fieldInfo.fromIndex), to.Field(fieldInfo.toIndex))
		if err != nil {
			return err
		}
	}

	return nil
}
