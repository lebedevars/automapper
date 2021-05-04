package automapper_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lebedevars/automapper"
)

type Simple1 struct {
	Int     int
	String  string
	Float64 float64
	Time    time.Time
}

type Simple2 struct {
	Int     int
	String  string
	Float64 float64
	Time    time.Time
}

type Embedded1 struct {
	Simple1 `mapper:"simple"`
}

type Embedded2 struct {
	Simple2 `mapper:"simple"`
}

type Structs1 struct {
	Field1 Simple1
	Field2 Simple1
	Field3 *Simple1
	Field4 *Simple1
}

type Structs2 struct {
	Field1 Simple2
	Field2 *Simple2
	Field3 Simple2
	Field4 *Simple2
}

type Slices1 struct {
	Field1 []Simple1
	Field2 []Simple1
	Field3 []*Simple1
	Field4 []*Simple1
}

type Slices2 struct {
	Field1 []Simple2
	Field2 []*Simple2
	Field3 []Simple2
	Field4 []*Simple2
}

type Arrays1 struct {
	Field1 [1]Simple1
	Field2 [1]Simple1
	Field3 [1]*Simple1
	Field4 [1]*Simple1
}

type Arrays2 struct {
	Field1 [1]Simple2
	Field2 [1]*Simple2
	Field3 [1]Simple2
	Field4 [1]*Simple2
}

type Converters1 struct {
	Field1 int
}

type Converters2 struct {
	Field1 string
}

func TestMapper_Map_Simple(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	simple1 := Simple1{
		Int:     1,
		String:  "string",
		Float64: 1,
		Time:    testTime,
	}
	simple2 := Simple2{}

	m := automapper.New()
	err := m.Map(&simple1, &simple2)

	assert.NoError(t, err)
	assert.EqualValues(t, simple1, simple2)
}

func TestMapper_Map_Structs(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	simple1 := Simple1{
		Int:     1,
		String:  "string",
		Float64: 1,
		Time:    testTime,
	}
	from := Structs1{
		Field1: simple1,
		Field2: simple1,
		Field3: &simple1,
		Field4: &simple1,
	}
	to := Structs2{}

	m := automapper.New()
	err := m.Map(&from, &to)

	assert.NoError(t, err)
	assert.EqualValues(t, from.Field1, to.Field1)
	assert.EqualValues(t, from.Field2, *to.Field2)
	assert.EqualValues(t, *from.Field3, to.Field3)
	assert.EqualValues(t, *from.Field4, *to.Field4)
}

func TestMapper_Map_Slices_Params(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	simple1 := Simple1{
		Int:     1,
		String:  "string",
		Float64: 1,
		Time:    testTime,
	}

	from := []Simple1{simple1}
	to := []Simple2{}

	m := automapper.New()
	err := m.Map(&from, &to)

	assert.NoError(t, err)
	assert.EqualValues(t, from[0], to[0])
}

func TestMapper_Map_Slices_Fields(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	simple1 := Simple1{
		Int:     1,
		String:  "string",
		Float64: 1,
		Time:    testTime,
	}
	from := Slices1{
		Field1: []Simple1{simple1},
		Field2: []Simple1{simple1},
		Field3: []*Simple1{&simple1},
		Field4: []*Simple1{&simple1},
	}
	to := Slices2{}

	m := automapper.New()
	err := m.Map(&from, &to)

	assert.NoError(t, err)
	assert.EqualValues(t, from.Field1[0], to.Field1[0])
	assert.EqualValues(t, from.Field2[0], *to.Field2[0])
	assert.EqualValues(t, *from.Field3[0], to.Field3[0])
	assert.EqualValues(t, *from.Field4[0], *to.Field4[0])
}

func TestMapper_Map_Array_Fields(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	simple1 := Simple1{
		Int:     1,
		String:  "string",
		Float64: 1,
		Time:    testTime,
	}
	from := Arrays1{
		Field1: [1]Simple1{simple1},
		Field2: [1]Simple1{simple1},
		Field3: [1]*Simple1{&simple1},
		Field4: [1]*Simple1{&simple1},
	}
	to := Arrays2{}

	m := automapper.New()
	err := m.Map(&from, &to)

	assert.NoError(t, err)
	assert.EqualValues(t, from.Field1[0], to.Field1[0])
	assert.EqualValues(t, from.Field2[0], *to.Field2[0])
	assert.EqualValues(t, *from.Field3[0], to.Field3[0])
	assert.EqualValues(t, *from.Field4[0], *to.Field4[0])
}

func TestMapper_Map_NilStructs(t *testing.T) {
	t.Parallel()
	from := Structs1{}
	to := Structs2{}

	m := automapper.New()
	err := m.Map(&from, &to)

	assert.NoError(t, err)
	assert.Nil(t, to.Field2)
	assert.Nil(t, to.Field4)
}

func TestMapper_MapEmbedded(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.Local)
	from := Embedded1{
		Simple1: Simple1{
			Int:     1,
			String:  "string",
			Float64: 1,
			Time:    testTime,
		},
	}
	to := Embedded2{}

	m := automapper.New()
	err := m.Map(&from, &to)

	assert.NoError(t, err)
	assert.EqualValues(t, from.Simple1, to.Simple2)
}

func TestMapper_Map_Converter_Err(t *testing.T) {
	t.Parallel()
	from := Converters1{1}
	to := Converters2{}
	m := automapper.New()

	err := m.Map(&from, &to)

	assert.ErrorIs(t, err, automapper.ErrMissingConverter)
}

func TestMapper_Map_Converter(t *testing.T) {
	t.Parallel()
	from := Converters1{1}
	to := Converters2{}
	m := automapper.New()
	err := m.Set(strconv.Itoa)
	assert.NoError(t, err)

	err = m.Map(&from, &to)

	assert.NoError(t, err)
	assert.EqualValues(t, "1", to.Field1)
}

func TestMapper_Map_Converter_WithError(t *testing.T) {
	t.Parallel()
	from := Converters2{"string"}
	to := Converters1{}
	m := automapper.New()
	err := m.Set(strconv.Atoi)
	assert.NoError(t, err)

	err = m.Map(&from, &to)

	assert.ErrorIs(t, err, automapper.ErrConverter)
}
