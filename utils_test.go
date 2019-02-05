/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package contractapi

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
)

// ================================
// Helpers
// ================================
var anotherGoodStructPropertiesMap = map[string]spec.Schema{
	"StringProp": *stringTypeVar.getSchema(),
	"StructProp": *spec.RefSchema("#/components/schemas/GoodStruct"),
}

var expectedAnotherGoodStructMetadata = ObjectMetadata{
	ID:                   "AnotherGoodStruct",
	Properties:           anotherGoodStructPropertiesMap,
	Required:             []string{"StringProp", "StructProp"},
	AdditionalProperties: false,
}

type AnotherBadStruct struct {
	Prop1 BadStruct `json:"Prop1"`
}

func testConvertError(t *testing.T, bt basicType, toPass string, expectedType string) {
	t.Helper()

	val, err := bt.convert(toPass)
	assert.EqualError(t, err, fmt.Sprintf("Cannot convert passed value %s to %s", toPass, expectedType), "should return error for invalid value")
	assert.Equal(t, reflect.Value{}, val, "should have returned the blank value")
}

func testGetSchema(t *testing.T, typ reflect.Type, expectedSchema *spec.Schema) {
	var schema *spec.Schema
	var err error

	t.Helper()

	schema, err = getSchema(typ, nil)

	assert.Nil(t, err, "err should be nil when not erroring")
	assert.Equal(t, expectedSchema, schema, "should return expected schema for type")
}

type MyResultError struct {
	gojsonschema.ResultError
	message string
}

func (re MyResultError) String() string {
	return re.message
}

// ================================
// Tests
// ================================

func TestStringInSlice(t *testing.T) {
	slice := []string{"word", "another word"}

	// Should return true when string present in slice
	assert.True(t, stringInSlice("word", slice), "should have returned true when sling in slice")

	// Should return false when string no present in slice
	assert.False(t, stringInSlice("bad word", slice), "should have returned true when sling in slice")
}

func TestSliceAsCommaSentence(t *testing.T) {
	slice := []string{"one", "two", "three"}

	assert.Equal(t, "one, two and three", sliceAsCommaSentence(slice), "should have put commas between slice elements and join last element with and")
}

func TestReadLocalFile(t *testing.T) {
	// should return the file and error of reading given filepath
	file, err := readLocalFile("schema/schema.json")

	expectedFile, expectedErr := ioutil.ReadFile("./schema/schema.json")

	assert.Equal(t, expectedFile, file, "should return same file")
	assert.Equal(t, expectedErr, err, "should return same err")

	file, err = readLocalFile("i don't exist")

	expectedFile, expectedErr = ioutil.ReadFile("i don't exist")

	assert.Equal(t, expectedFile, file, "should return same file")
	assert.Contains(t, err.Error(), strings.Split(expectedErr.Error(), ":")[1], "should return same err")
}

func TestConvert(t *testing.T) {

	var val reflect.Value
	var err error

	// Should convert successfully for valid values
	val, err = stringTypeVar.convert("some string")
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, "some string", val.Interface().(string), "should have returned the same string")

	val, err = boolTypeVar.convert("true")
	assert.Nil(t, err, "should not return error for valid value")
	assert.True(t, val.Interface().(bool), "should have returned the boolean true")

	val, err = boolTypeVar.convert("false")
	assert.Nil(t, err, "should not return error for valid value")
	assert.False(t, val.Interface().(bool), "should have returned the boolean true")

	val, err = intTypeVar.convert("123")
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, 123, val.Interface().(int), "should have returned the int value")

	val, err = int8TypeVar.convert(strconv.Itoa(math.MaxInt8))
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, int8(math.MaxInt8), val.Interface().(int8), "should have returned the int8 value")

	val, err = int16TypeVar.convert(strconv.Itoa(math.MaxInt16))
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, int16(math.MaxInt16), val.Interface().(int16), "should have returned the int16 value")

	val, err = int32TypeVar.convert(strconv.Itoa(math.MaxInt32))
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, int32(math.MaxInt32), val.Interface().(int32), "should have returned the int32 value")

	val, err = int64TypeVar.convert(strconv.Itoa(math.MaxInt64))
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, int64(math.MaxInt64), val.Interface().(int64), "should have returned the int64 value")

	val, err = uintTypeVar.convert("123")
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, uint(123), val.Interface().(uint), "should have returned the uint value")

	val, err = uint8TypeVar.convert(fmt.Sprint(math.MaxUint8))
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, uint8(math.MaxUint8), val.Interface().(uint8), "should have returned the uint8 value")

	val, err = uint16TypeVar.convert(fmt.Sprint(math.MaxUint16))
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, uint16(math.MaxUint16), val.Interface().(uint16), "should have returned the uint16 value")

	val, err = uint32TypeVar.convert(fmt.Sprint(math.MaxUint32))
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, uint32(math.MaxUint32), val.Interface().(uint32), "should have returned the uint32 value")

	val, err = uint64TypeVar.convert(fmt.Sprint(math.MaxInt64))
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, uint64(math.MaxInt64), val.Interface().(uint64), "should have returned the uint64 value")

	val, err = float32TypeVar.convert(fmt.Sprint(math.MaxFloat32))
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, float32(math.MaxFloat32), val.Interface().(float32), "should have returned the float32 value")

	val, err = float64TypeVar.convert(fmt.Sprint(math.MaxFloat64))
	assert.Nil(t, err, "should not return error for valid value")
	assert.Equal(t, float64(math.MaxFloat64), val.Interface().(float64), "should have returned the float64 value")

	val, err = interfaceTypeVar.convert("hello world")
	assert.Nil(t, err, "should never return error for interface")
	assert.Equal(t, "hello world", val.Interface().(string), "should return string that went in")

	// Should error when bad value passed
	testConvertError(t, boolTypeVar, "gibberish", "bool")
	testConvertError(t, intTypeVar, "gibberish", "int")
	testConvertError(t, int8TypeVar, "gibberish", "int8")
	testConvertError(t, int8TypeVar, strconv.Itoa(math.MaxInt8+1), "int8")
	testConvertError(t, int16TypeVar, "gibberish", "int16")
	testConvertError(t, int16TypeVar, strconv.Itoa(math.MaxInt16+1), "int16")
	testConvertError(t, int32TypeVar, "gibberish", "int32")
	testConvertError(t, int32TypeVar, strconv.Itoa(math.MaxInt32+1), "int32")
	testConvertError(t, int64TypeVar, "gibberish", "int64")
	testConvertError(t, uintTypeVar, "gibberish", "uint")
	testConvertError(t, uint8TypeVar, "gibberish", "uint8")
	testConvertError(t, uint8TypeVar, fmt.Sprint(math.MaxUint8+1), "uint8")
	testConvertError(t, uint16TypeVar, "gibberish", "uint16")
	testConvertError(t, uint16TypeVar, fmt.Sprint(math.MaxUint16+1), "uint16")
	testConvertError(t, uint32TypeVar, "gibberish", "uint32")
	testConvertError(t, uint32TypeVar, fmt.Sprint(math.MaxUint32+1), "uint32")
	testConvertError(t, uint64TypeVar, "gibberish", "uint64")
	testConvertError(t, float32TypeVar, "gibberish", "float32")
	testConvertError(t, float32TypeVar, fmt.Sprint(math.MaxFloat64), "float32")
	testConvertError(t, float64TypeVar, "gibberish", "float64")
}

func TestTypesGetSchema(t *testing.T) {
	var expectedSchema *spec.Schema
	var actualSchema *spec.Schema

	var minimum float64
	var maximum float64

	// string
	expectedSchema = spec.StringProperty()
	actualSchema = stringTypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back string schema")

	// bool
	expectedSchema = spec.BooleanProperty()
	actualSchema = boolTypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back bool schema")

	// ints
	expectedSchema = spec.Int64Property()
	actualSchema = intTypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back int schema")

	expectedSchema = spec.Int8Property()
	actualSchema = int8TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back int8 schema")

	expectedSchema = spec.Int16Property()
	actualSchema = int16TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back int16 schema")

	expectedSchema = spec.Int32Property()
	actualSchema = int32TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back int32 schema")

	expectedSchema = spec.Int64Property()
	actualSchema = int64TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back int64 schema")

	// uints
	expectedSchema = spec.Float64Property()
	multOf := float64(1)
	expectedSchema.MultipleOf = &multOf
	minimum = float64(0)
	expectedSchema.Minimum = &minimum
	maximum = float64(18446744073709551615)
	expectedSchema.Maximum = &maximum
	actualSchema = uintTypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back uint schema")

	expectedSchema = spec.Int32Property()
	minimum = float64(0)
	expectedSchema.Minimum = &minimum
	maximum = float64(255)
	expectedSchema.Maximum = &maximum
	actualSchema = uint8TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back uint8 schema")

	expectedSchema = spec.Int64Property()
	minimum = float64(0)
	expectedSchema.Minimum = &minimum
	maximum = float64(65535)
	expectedSchema.Maximum = &maximum
	actualSchema = uint16TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back uint16 schema")

	expectedSchema = spec.Int64Property()
	minimum = float64(0)
	expectedSchema.Minimum = &minimum
	maximum = float64(4294967295)
	expectedSchema.Maximum = &maximum
	actualSchema = uint32TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back uint32 schema")

	expectedSchema = spec.Float64Property()
	expectedSchema.MultipleOf = &multOf
	minimum = float64(0)
	expectedSchema.Minimum = &minimum
	maximum = float64(18446744073709551615)
	expectedSchema.Maximum = &maximum
	actualSchema = uint64TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back uint64 schema")

	// floats
	expectedSchema = spec.Float32Property()
	actualSchema = float32TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back float32 schema")

	expectedSchema = spec.Float64Property()
	actualSchema = float64TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back float64 schema")

	// interface
	expectedSchema = new(spec.Schema)
	actualSchema = interfaceTypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back interface schema")
}

func TestBuildArraySchema(t *testing.T) {
	var schema *spec.Schema
	var err error

	// Should return an error when array is passed with a length of zero
	zeroArr := [0]int{}
	schema, err = buildArraySchema(reflect.ValueOf(zeroArr), nil)

	assert.Equal(t, errors.New("Arrays must have length greater than 0"), err, "should throw error when 0 length array passed")
	assert.Nil(t, schema, "should not have returned a schema for zero array")

	// Should return error when getSchema would
	schema, err = buildArraySchema(reflect.ValueOf([1]complex128{}), nil)
	_, expectedErr := getSchema(reflect.TypeOf(complex128(1)), nil)

	assert.Nil(t, schema, "spec should be nil when getSchema fails from buildArraySchema")
	assert.Equal(t, expectedErr, err, "should have same error as getSchema")
}

func TestBuildSliceSchema(t *testing.T) {
	var schema *spec.Schema
	var err error

	// Should handle adding to the length of the slice if currently 0
	assert.NotPanics(t, func() { buildSliceSchema(reflect.ValueOf([]string{}), nil) }, "shouldn't have panicked when slice sent was empty")

	// Should return error when getSchema would
	schema, err = buildSliceSchema(reflect.ValueOf([]complex128{}), nil)
	_, expectedErr := getSchema(reflect.TypeOf(complex128(1)), nil)

	assert.Nil(t, schema, "spec should be nil when buildArrayOrSliceSchema fails from buildSliceSchema")
	assert.Equal(t, expectedErr, err, "should have same error as buildArrayOrSliceSchema")
}

func TestAddComponentIfNotExists(t *testing.T) {
	var err error
	var components *ComponentMetadata

	// Should return nil when object with that name already in components map
	someObject := ObjectMetadata{}
	someObject.ID = "some ID"

	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)
	components.Schemas["GoodStruct"] = someObject

	err = addComponentIfNotExists(reflect.TypeOf(GoodStruct{}), components)

	assert.Nil(t, err, "should return nil when already exists")
	assert.Equal(t, len(components.Schemas), 1, "should not have added a new component")
	assert.Equal(t, components.Schemas["GoodStruct"].ID, "some ID", "should not overwrite existing component")

	// Should return nil when object with that name already in components map and object is pointer
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)
	components.Schemas["GoodStruct"] = someObject

	err = addComponentIfNotExists(reflect.TypeOf(new(GoodStruct)), components)

	assert.Nil(t, err, "should return nil when already exists")
	assert.Equal(t, len(components.Schemas), 1, "should not have added a new component")
	assert.Equal(t, components.Schemas["GoodStruct"].ID, "some ID", "should not overwrite existing component")

	// Should build up schema and to components
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	err = addComponentIfNotExists(reflect.TypeOf(GoodStruct{}), components)

	assert.Nil(t, err, "should return nil when valid object")
	assert.Equal(t, len(components.Schemas), 1, "should have added a new component")
	assert.Equal(t, components.Schemas["GoodStruct"], goodStructMetadata, "should have added correct metadata to components")

	// Should build up schema and to components when object is pointer
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	err = addComponentIfNotExists(reflect.TypeOf(new(GoodStruct)), components)

	assert.Nil(t, err, "should return nil when valid object")
	assert.Equal(t, len(components.Schemas), 1, "should have added a new component")
	assert.Equal(t, components.Schemas["GoodStruct"], goodStructMetadata, "should have added correct metadata to components")

	// should add component for sub structs of structs
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	err = addComponentIfNotExists(reflect.TypeOf(new(AnotherGoodStruct)), components)

	assert.Nil(t, err, "should return nil when valid object")
	assert.Equal(t, len(components.Schemas), 2, "should have added two new components")
	assert.Equal(t, components.Schemas["GoodStruct"], goodStructMetadata, "should have added correct metadata to components for sub struct")
	assert.Equal(t, components.Schemas["AnotherGoodStruct"], expectedAnotherGoodStructMetadata, "should have added correct metadata to components for main struct")

	// Should error for struct with bad property
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	err = addComponentIfNotExists(reflect.TypeOf(new(BadStruct)), components)

	assert.EqualError(t, err, "complex64 was not a valid type", "should return err when invalid object")
	assert.Equal(t, len(components.Schemas), 0, "should not have added new component")

	// Should error for struct with a bad struct
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	err = addComponentIfNotExists(reflect.TypeOf(new(AnotherBadStruct)), components)

	assert.EqualError(t, err, "complex64 was not a valid type", "should return err when invalid object")
	assert.Equal(t, len(components.Schemas), 0, "should not have added new component")
}

func TestBuildStructSchema(t *testing.T) {
	var schema *spec.Schema
	var err error

	components := new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	// Should return error when addComponentIfNotExists does
	schema, err = buildStructSchema(reflect.TypeOf(BadStruct{}), components)
	expectedErr := addComponentIfNotExists(reflect.TypeOf(BadStruct{}), components)

	assert.Nil(t, schema, "spec should be nil when buildArrayOrSliceSchema fails from buildSliceSchema")
	assert.NotNil(t, err, "error should not be nil")
	assert.Equal(t, expectedErr, err, "should have same error as buildArrayOrSliceSchema")

	// Should return a ref schema when adding component doesn't error
	schema, err = buildStructSchema(reflect.TypeOf(GoodStruct{}), components)
	assert.Nil(t, err, "should nto return error when struct is good")
	assert.Equal(t, schema, spec.RefSchema("#/components/schemas/GoodStruct"), "should make schema ref to component")

	_, ok := components.Schemas["GoodStruct"]

	assert.True(t, ok, "should have added component")

	// Should handle struct when pointer
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	schema, err = buildStructSchema(reflect.TypeOf(new(GoodStruct)), components)
	assert.Nil(t, err, "should nto return error when struct is good")
	assert.Equal(t, schema, spec.RefSchema("#/components/schemas/GoodStruct"), "should make schema ref to component")

	_, ok = components.Schemas["GoodStruct"]

	assert.True(t, ok, "should have added component")
}

func TestGetSchema(t *testing.T) {
	// should return error if passed type not in basic types
	schema, err := getSchema(reflect.TypeOf(complex128(1)), nil)

	assert.Nil(t, schema, "schema should be nil when erroring")
	assert.EqualError(t, err, "complex128 was not a valid type", "should have returned correct error for bad type")

	// should return schema for a basic type
	testGetSchema(t, stringRefType, stringTypeVar.getSchema())
	testGetSchema(t, boolRefType, boolTypeVar.getSchema())
	testGetSchema(t, intRefType, intTypeVar.getSchema())
	testGetSchema(t, int8RefType, int8TypeVar.getSchema())
	testGetSchema(t, int16RefType, int16TypeVar.getSchema())
	testGetSchema(t, int32RefType, int32TypeVar.getSchema())
	testGetSchema(t, int64RefType, int64TypeVar.getSchema())
	testGetSchema(t, uintRefType, uintTypeVar.getSchema())
	testGetSchema(t, uint8RefType, uint8TypeVar.getSchema())
	testGetSchema(t, uint16RefType, uint16TypeVar.getSchema())
	testGetSchema(t, uint32RefType, uint32TypeVar.getSchema())
	testGetSchema(t, uint64RefType, uint64TypeVar.getSchema())
	testGetSchema(t, float32RefType, float32TypeVar.getSchema())
	testGetSchema(t, float64RefType, float64TypeVar.getSchema())

	mc := myContract{}
	mcFuncType := reflect.TypeOf(mc.AfterTransactionWithInterface)

	testGetSchema(t, mcFuncType.In(1), interfaceTypeVar.getSchema())

	// Should return error when array is not one of the basic types
	badArr := [1]complex128{}
	schema, err = getSchema(reflect.TypeOf(badArr), nil)

	assert.EqualError(t, err, "complex128 was not a valid type", "should throw error when invalid type passed")
	assert.Nil(t, schema, "should not have returned a schema for an array of bad type")

	// Should return error when multidimensional array is not one of the basic types
	badMultArr := [1][1]complex128{}
	schema, err = getSchema(reflect.TypeOf(badMultArr), nil)

	assert.EqualError(t, err, "complex128 was not a valid type", "should throw error when invalid type passed")
	assert.Nil(t, schema, "should not have returned a schema for an array of bad type")

	// Should return an error when array is passed with sub array with a length of zero
	zeroSubArr := [1][0]int{}
	schema, err = getSchema(reflect.TypeOf(zeroSubArr), nil)

	assert.Equal(t, errors.New("Arrays must have length greater than 0"), err, "should throw error when 0 length array passed")
	assert.Nil(t, schema, "should not have returned a schema for zero array")

	// Should return schema for arrays made of each of the valid types
	stringArraySchema := spec.ArrayProperty(stringTypeVar.getSchema())
	boolArraySchema := spec.ArrayProperty(boolTypeVar.getSchema())
	intArraySchema := spec.ArrayProperty(intTypeVar.getSchema())
	int8ArraySchema := spec.ArrayProperty(int8TypeVar.getSchema())
	int16ArraySchema := spec.ArrayProperty(int16TypeVar.getSchema())
	int32ArraySchema := spec.ArrayProperty(int32TypeVar.getSchema())
	int64ArraySchema := spec.ArrayProperty(int64TypeVar.getSchema())
	uintArraySchema := spec.ArrayProperty(uintTypeVar.getSchema())
	uint8ArraySchema := spec.ArrayProperty(uint8TypeVar.getSchema())
	uint16ArraySchema := spec.ArrayProperty(uint16TypeVar.getSchema())
	uint32ArraySchema := spec.ArrayProperty(uint32TypeVar.getSchema())
	uint64ArraySchema := spec.ArrayProperty(uint64TypeVar.getSchema())
	float32ArraySchema := spec.ArrayProperty(float32TypeVar.getSchema())
	float64ArraySchema := spec.ArrayProperty(float64TypeVar.getSchema())

	testGetSchema(t, reflect.TypeOf([1]string{}), stringArraySchema)
	testGetSchema(t, reflect.TypeOf([1]bool{}), boolArraySchema)
	testGetSchema(t, reflect.TypeOf([1]int{}), intArraySchema)
	testGetSchema(t, reflect.TypeOf([1]int8{}), int8ArraySchema)
	testGetSchema(t, reflect.TypeOf([1]int16{}), int16ArraySchema)
	testGetSchema(t, reflect.TypeOf([1]int32{}), int32ArraySchema)
	testGetSchema(t, reflect.TypeOf([1]int64{}), int64ArraySchema)
	testGetSchema(t, reflect.TypeOf([1]uint{}), uintArraySchema)
	testGetSchema(t, reflect.TypeOf([1]uint8{}), uint8ArraySchema)
	testGetSchema(t, reflect.TypeOf([1]uint16{}), uint16ArraySchema)
	testGetSchema(t, reflect.TypeOf([1]uint32{}), uint32ArraySchema)
	testGetSchema(t, reflect.TypeOf([1]uint64{}), uint64ArraySchema)
	testGetSchema(t, reflect.TypeOf([1]float32{}), float32ArraySchema)
	testGetSchema(t, reflect.TypeOf([1]float64{}), float64ArraySchema)
	testGetSchema(t, reflect.TypeOf([1]byte{}), uint8ArraySchema)
	testGetSchema(t, reflect.TypeOf([1]rune{}), int32ArraySchema)

	// Should return schema for multidimensional arrays made of each of the basic types
	testGetSchema(t, reflect.TypeOf([1][1]string{}), spec.ArrayProperty(stringArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]bool{}), spec.ArrayProperty(boolArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]int{}), spec.ArrayProperty(intArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]int8{}), spec.ArrayProperty(int8ArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]int16{}), spec.ArrayProperty(int16ArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]int32{}), spec.ArrayProperty(int32ArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]int64{}), spec.ArrayProperty(int64ArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]uint{}), spec.ArrayProperty(uintArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]uint8{}), spec.ArrayProperty(uint8ArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]uint16{}), spec.ArrayProperty(uint16ArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]uint32{}), spec.ArrayProperty(uint32ArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]uint64{}), spec.ArrayProperty(uint64ArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]float32{}), spec.ArrayProperty(float32ArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]float64{}), spec.ArrayProperty(float64ArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]byte{}), spec.ArrayProperty(uint8ArraySchema))
	testGetSchema(t, reflect.TypeOf([1][1]rune{}), spec.ArrayProperty(int32ArraySchema))

	// Should build schema for a big multidimensional array
	testGetSchema(t, reflect.TypeOf([1][2][3][4][5][6][7][8]string{}), spec.ArrayProperty(spec.ArrayProperty(spec.ArrayProperty(spec.ArrayProperty(spec.ArrayProperty(spec.ArrayProperty(spec.ArrayProperty(stringArraySchema))))))))

	// Should return error when array is not one of the valid types
	badSlice := []complex128{}
	schema, err = getSchema(reflect.TypeOf(badSlice), nil)

	assert.EqualError(t, err, "complex128 was not a valid type", "should throw error when invalid type passed")
	assert.Nil(t, schema, "should not have returned a schema for an array of bad type")

	// Should return an error when array is passed with sub array with a length of zero
	zeroSubArrInSlice := [][0]int{}
	schema, err = getSchema(reflect.TypeOf(zeroSubArrInSlice), nil)

	assert.Equal(t, errors.New("Arrays must have length greater than 0"), err, "should throw error when 0 length array passed")
	assert.Nil(t, schema, "should not have returned a schema for zero array")

	// should build schema for slices of original types
	testGetSchema(t, reflect.TypeOf([]string{""}), stringArraySchema)
	testGetSchema(t, reflect.TypeOf([]bool{true}), boolArraySchema)
	testGetSchema(t, reflect.TypeOf([]int{1}), intArraySchema)
	testGetSchema(t, reflect.TypeOf([]int8{1}), int8ArraySchema)
	testGetSchema(t, reflect.TypeOf([]int16{1}), int16ArraySchema)
	testGetSchema(t, reflect.TypeOf([]int32{1}), int32ArraySchema)
	testGetSchema(t, reflect.TypeOf([]int64{1}), int64ArraySchema)
	testGetSchema(t, reflect.TypeOf([]uint{1}), uintArraySchema)
	testGetSchema(t, reflect.TypeOf([]uint8{1}), uint8ArraySchema)
	testGetSchema(t, reflect.TypeOf([]uint16{1}), uint16ArraySchema)
	testGetSchema(t, reflect.TypeOf([]uint32{1}), uint32ArraySchema)
	testGetSchema(t, reflect.TypeOf([]uint64{1}), uint64ArraySchema)
	testGetSchema(t, reflect.TypeOf([]float32{1}), float32ArraySchema)
	testGetSchema(t, reflect.TypeOf([]float64{1}), float64ArraySchema)
	testGetSchema(t, reflect.TypeOf([]byte{1}), uint8ArraySchema)
	testGetSchema(t, reflect.TypeOf([]rune{1}), int32ArraySchema)

	// Should return schema for multidimensional slices made of each of the basic types
	testGetSchema(t, reflect.TypeOf([][]bool{[]bool{}}), spec.ArrayProperty(boolArraySchema))
	testGetSchema(t, reflect.TypeOf([][]int{[]int{}}), spec.ArrayProperty(intArraySchema))
	testGetSchema(t, reflect.TypeOf([][]int8{[]int8{}}), spec.ArrayProperty(int8ArraySchema))
	testGetSchema(t, reflect.TypeOf([][]int16{[]int16{}}), spec.ArrayProperty(int16ArraySchema))
	testGetSchema(t, reflect.TypeOf([][]int32{[]int32{}}), spec.ArrayProperty(int32ArraySchema))
	testGetSchema(t, reflect.TypeOf([][]int64{[]int64{}}), spec.ArrayProperty(int64ArraySchema))
	testGetSchema(t, reflect.TypeOf([][]uint{[]uint{}}), spec.ArrayProperty(uintArraySchema))
	testGetSchema(t, reflect.TypeOf([][]uint8{[]uint8{}}), spec.ArrayProperty(uint8ArraySchema))
	testGetSchema(t, reflect.TypeOf([][]uint16{[]uint16{}}), spec.ArrayProperty(uint16ArraySchema))
	testGetSchema(t, reflect.TypeOf([][]uint32{[]uint32{}}), spec.ArrayProperty(uint32ArraySchema))
	testGetSchema(t, reflect.TypeOf([][]uint64{[]uint64{}}), spec.ArrayProperty(uint64ArraySchema))
	testGetSchema(t, reflect.TypeOf([][]float32{[]float32{}}), spec.ArrayProperty(float32ArraySchema))
	testGetSchema(t, reflect.TypeOf([][]float64{[]float64{}}), spec.ArrayProperty(float64ArraySchema))
	testGetSchema(t, reflect.TypeOf([][]byte{[]byte{}}), spec.ArrayProperty(uint8ArraySchema))
	testGetSchema(t, reflect.TypeOf([][]rune{[]rune{}}), spec.ArrayProperty(int32ArraySchema))

	// Should handle an array of slice
	testGetSchema(t, reflect.TypeOf([1][]string{}), spec.ArrayProperty(stringArraySchema))

	// Should handle a slice of array
	testGetSchema(t, reflect.TypeOf([][1]string{[1]string{}}), spec.ArrayProperty(stringArraySchema))

	// Should return error when multidimensional array/slice/array is bad
	badMixedArr := [1][][0]string{}
	schema, err = getSchema(reflect.TypeOf(badMixedArr), nil)

	assert.EqualError(t, err, "Arrays must have length greater than 0", "should throw error when 0 length array passed")
	assert.Nil(t, schema, "schema should be nil when sub array bad type")

	var components *ComponentMetadata

	// Should handle a valid struct and add to components
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	schema, err = getSchema(reflect.TypeOf(GoodStruct{}), components)

	assert.Nil(t, err, "should return nil when valid object")
	assert.Equal(t, len(components.Schemas), 1, "should have added a new component")
	assert.Equal(t, components.Schemas["GoodStruct"], goodStructMetadata, "should have added correct metadata to components")
	assert.Equal(t, schema, spec.RefSchema("#/components/schemas/GoodStruct"))

	// should handle pointer to struct
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	schema, err = getSchema(reflect.TypeOf(new(GoodStruct)), components)

	assert.Nil(t, err, "should return nil when valid object")
	assert.Equal(t, len(components.Schemas), 1, "should have added a new component")
	assert.Equal(t, components.Schemas["GoodStruct"], goodStructMetadata, "should have added correct metadata to components")
	assert.Equal(t, schema, spec.RefSchema("#/components/schemas/GoodStruct"))

	// Should handle an array of structs
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	schema, err = getSchema(reflect.TypeOf([1]GoodStruct{}), components)

	assert.Nil(t, err, "should return nil when valid object")
	assert.Equal(t, len(components.Schemas), 1, "should have added a new component")
	assert.Equal(t, components.Schemas["GoodStruct"], goodStructMetadata, "should have added correct metadata to components")
	assert.Equal(t, schema, spec.ArrayProperty(spec.RefSchema("#/components/schemas/GoodStruct")))

	// Should handle a slice of structs
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	schema, err = getSchema(reflect.TypeOf([]GoodStruct{}), components)

	assert.Nil(t, err, "should return nil when valid object")
	assert.Equal(t, len(components.Schemas), 1, "should have added a new component")
	assert.Equal(t, components.Schemas["GoodStruct"], goodStructMetadata, "should have added correct metadata to components")
	assert.Equal(t, schema, spec.ArrayProperty(spec.RefSchema("#/components/schemas/GoodStruct")))

	// Should handle a valid struct with struct property and add to components
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	schema, err = getSchema(reflect.TypeOf(new(AnotherGoodStruct)), components)

	assert.Nil(t, err, "should return nil when valid object")
	assert.Equal(t, len(components.Schemas), 2, "should have added two new components")
	assert.Equal(t, components.Schemas["GoodStruct"], goodStructMetadata, "should have added correct metadata to components for sub struct")
	assert.Equal(t, components.Schemas["AnotherGoodStruct"], expectedAnotherGoodStructMetadata, "should have added correct metadata to components for main struct")
	assert.Equal(t, schema, spec.RefSchema("#/components/schemas/AnotherGoodStruct"))

	// Should return an error for a bad struct
	components = new(ComponentMetadata)
	components.Schemas = make(map[string]ObjectMetadata)

	schema, err = getSchema(reflect.TypeOf(new(BadStruct)), components)

	assert.Nil(t, schema, "should not give back a schema when struct is bad")
	assert.EqualError(t, err, "complex64 was not a valid type", "should return err when invalid object")
	assert.Equal(t, len(components.Schemas), 0, "should not have added new component")
}

func TestValidateErrorsToString(t *testing.T) {
	// should join errors with a new line
	error1 := MyResultError{
		message: "some error message",
	}
	error2 := MyResultError{
		message: "another error message",
	}

	assert.Equal(t, "1. some error message", validateErrorsToString([]gojsonschema.ResultError{error1}), "should return nicely formatted single error")
	assert.Equal(t, "1. some error message\n2. another error message", validateErrorsToString([]gojsonschema.ResultError{error1, error2}), "should return nicely formatted multiple error")
}
