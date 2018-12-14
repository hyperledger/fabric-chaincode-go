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

// Package contractapi provides the contract interface, a high level API for application developers to implement business logic for Hyperledger Fabric.
package contractapi

import (
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/go-openapi/spec"
)

type ioHlp interface {
	ReadFile(string) ([]byte, error)
}

type ioHlpStr struct{}

func (io ioHlpStr) ReadFile(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

var ioutilHelper ioHlp = ioHlpStr{}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func typeInSlice(a reflect.Type, list []reflect.Type) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func sliceAsCommaSentence(slice []string) string {
	return strings.Replace(strings.Join(slice, " and "), " and ", ", ", len(slice)-2)
}

func embedsStruct(sc interface{}, toEmbed string) bool {

	ifv := reflect.ValueOf(sc).Elem()

	ift := reflect.TypeOf(sc).Elem()

	for i := 0; i < ifv.NumField(); i++ {

		v := ifv.Field(i)

		t := ift.Field(i)

		kind := v.Kind()

		if kind == reflect.Struct && t.Type.String() == toEmbed {

			return true

		}
	}
	return false
}

func readLocalFile(localPath string) ([]byte, error) {
	_, filename, _, _ := runtime.Caller(1)

	schemaPath := path.Join(path.Dir(filename), localPath)

	file, err := ioutilHelper.ReadFile(schemaPath)

	return file, err
}

// Types
type basicType interface {
	convert(string) (reflect.Value, error)
	getSchema() *spec.Schema
}

type stringType struct{}

func (st *stringType) convert(value string) (reflect.Value, error) {
	return reflect.ValueOf(value), nil
}

func (st *stringType) getSchema() *spec.Schema {
	return spec.StringProperty()
}

type boolType struct{}

func (bt *boolType) convert(value string) (reflect.Value, error) {
	var boolVal bool
	var err error
	if value != "" {
		boolVal, err = strconv.ParseBool(value)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to bool", value)
		}
	}

	return reflect.ValueOf(boolVal), nil
}

func (bt *boolType) getSchema() *spec.Schema {
	return spec.BooleanProperty()
}

type intType struct{}

func (it *intType) convert(value string) (reflect.Value, error) {
	var intVal int
	var err error
	if value != "" {
		intVal, err = strconv.Atoi(value)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to int", value)
		}
	}

	return reflect.ValueOf(intVal), nil
}

func (it *intType) getSchema() *spec.Schema {
	return spec.Int64Property()
}

type int8Type struct{}

func (it *int8Type) convert(value string) (reflect.Value, error) {
	var intVal int8
	if value != "" {
		int64val, err := strconv.ParseInt(value, 10, 8)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to int8", value)
		}

		intVal = int8(int64val)
	}

	return reflect.ValueOf(intVal), nil
}

func (it *int8Type) getSchema() *spec.Schema {
	return spec.Int8Property()
}

type int16Type struct{}

func (it *int16Type) convert(value string) (reflect.Value, error) {
	var intVal int16
	if value != "" {
		int64val, err := strconv.ParseInt(value, 10, 16)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to int16", value)
		}

		intVal = int16(int64val)
	}

	return reflect.ValueOf(intVal), nil
}

func (it *int16Type) getSchema() *spec.Schema {
	return spec.Int16Property()
}

type int32Type struct{}

func (it *int32Type) convert(value string) (reflect.Value, error) {
	var intVal int32
	if value != "" {
		int64val, err := strconv.ParseInt(value, 10, 32)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to int32", value)
		}

		intVal = int32(int64val)
	}

	return reflect.ValueOf(intVal), nil
}

func (it *int32Type) getSchema() *spec.Schema {
	return spec.Int32Property()
}

type int64Type struct{}

func (it *int64Type) convert(value string) (reflect.Value, error) {
	var intVal int64
	var err error
	if value != "" {
		intVal, err = strconv.ParseInt(value, 10, 64)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to int64", value)
		}
	}

	return reflect.ValueOf(intVal), nil
}

func (it *int64Type) getSchema() *spec.Schema {
	return spec.Int64Property()
}

type uintType struct{}

func (ut *uintType) convert(value string) (reflect.Value, error) {
	var uintVal uint
	if value != "" {
		uint64Val, err := strconv.ParseUint(value, 10, 64)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to uint", value)
		}

		uintVal = uint(uint64Val)
	}

	return reflect.ValueOf(uintVal), nil
}

func (ut *uintType) getSchema() *spec.Schema {
	schema := spec.Float64Property()
	multOf := float64(1)
	schema.MultipleOf = &multOf
	minimum := float64(0)
	schema.Minimum = &minimum
	maximum := float64(18446744073709551615)
	schema.Maximum = &maximum
	return schema
}

type uint8Type struct{}

func (ut *uint8Type) convert(value string) (reflect.Value, error) {
	var uintVal uint8
	if value != "" {
		uint64Val, err := strconv.ParseUint(value, 10, 8)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to uint8", value)
		}

		uintVal = uint8(uint64Val)
	}

	return reflect.ValueOf(uintVal), nil
}

func (ut *uint8Type) getSchema() *spec.Schema {
	schema := spec.Int32Property()
	minimum := float64(0)
	schema.Minimum = &minimum
	maximum := float64(255)
	schema.Maximum = &maximum
	return schema
}

type uint16Type struct{}

func (ut *uint16Type) convert(value string) (reflect.Value, error) {
	var uintVal uint16
	if value != "" {
		uint64Val, err := strconv.ParseUint(value, 10, 16)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to uint16", value)
		}

		uintVal = uint16(uint64Val)
	}

	return reflect.ValueOf(uintVal), nil
}

func (ut *uint16Type) getSchema() *spec.Schema {
	schema := spec.Int64Property()
	minimum := float64(0)
	schema.Minimum = &minimum
	maximum := float64(65535)
	schema.Maximum = &maximum
	return schema
}

type uint32Type struct{}

func (ut *uint32Type) convert(value string) (reflect.Value, error) {
	var uintVal uint32
	if value != "" {
		uint64Val, err := strconv.ParseUint(value, 10, 32)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to uint32", value)
		}

		uintVal = uint32(uint64Val)
	}

	return reflect.ValueOf(uintVal), nil
}

func (ut *uint32Type) getSchema() *spec.Schema {
	schema := spec.Int64Property()
	minimum := float64(0)
	schema.Minimum = &minimum
	maximum := float64(4294967295)
	schema.Maximum = &maximum
	return schema
}

type uint64Type struct{}

func (ut *uint64Type) convert(value string) (reflect.Value, error) {
	var uintVal uint64
	var err error
	if value != "" {
		uintVal, err = strconv.ParseUint(value, 10, 64)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to uint64", value)
		}
	}

	return reflect.ValueOf(uintVal), nil
}

func (ut *uint64Type) getSchema() *spec.Schema {
	schema := spec.Float64Property()
	multOf := float64(1)
	schema.MultipleOf = &multOf
	minimum := float64(0)
	schema.Minimum = &minimum
	maximum := float64(18446744073709551615)
	schema.Maximum = &maximum
	return schema
}

type float32Type struct{}

func (ft *float32Type) convert(value string) (reflect.Value, error) {
	var floatVal float32
	if value != "" {
		float64Val, err := strconv.ParseFloat(value, 32)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to float32", value)
		}

		floatVal = float32(float64Val)
	}

	return reflect.ValueOf(floatVal), nil
}

func (ft *float32Type) getSchema() *spec.Schema {
	return spec.Float32Property()
}

type float64Type struct{}

func (ft *float64Type) convert(value string) (reflect.Value, error) {
	var floatVal float64
	var err error
	if value != "" {
		floatVal, err = strconv.ParseFloat(value, 64)

		if err != nil {
			return reflect.Value{}, fmt.Errorf("Cannot convert passed value %s to float64", value)
		}
	}

	return reflect.ValueOf(floatVal), nil
}

func (ft *float64Type) getSchema() *spec.Schema {
	return spec.Float64Property()
}

var basicTypes = map[reflect.Kind]basicType{
	reflect.Bool:    new(boolType),
	reflect.Float32: new(float32Type),
	reflect.Float64: new(float64Type),
	reflect.Int:     new(intType),
	reflect.Int8:    new(int8Type),
	reflect.Int16:   new(int16Type),
	reflect.Int32:   new(int32Type),
	reflect.Int64:   new(int64Type),
	reflect.String:  new(stringType),
	reflect.Uint:    new(uintType),
	reflect.Uint8:   new(uint8Type),
	reflect.Uint16:  new(uint16Type),
	reflect.Uint32:  new(uint32Type),
	reflect.Uint64:  new(uint64Type),
}

func listBasicTypes() string {
	types := []string{}

	for el := range basicTypes {
		types = append(types, el.String())
	}
	sort.Strings(types)

	return sliceAsCommaSentence(types)
}

func buildArraySchema(array reflect.Value) (*spec.Schema, error) {
	if array.Len() < 1 {
		return nil, fmt.Errorf("Arrays must have length greater than 0")
	}

	return buildArrayOrSliceSchema(array)
}

func buildSliceSchema(slice reflect.Value) (*spec.Schema, error) {
	if slice.Len() < 1 {
		slice = reflect.MakeSlice(slice.Type(), 1, 10)
	}

	return buildArrayOrSliceSchema(slice)
}

func buildArrayOrSliceSchema(obj reflect.Value) (*spec.Schema, error) {
	var lowerSchema *spec.Schema
	var err error

	if obj.Index(0).Kind() == reflect.Array {
		lowerSchema, err = buildArraySchema(obj.Index(0))

		if err != nil {
			return nil, err
		}
	} else if obj.Index(0).Kind() == reflect.Slice {
		lowerSchema, err = buildSliceSchema(obj.Index(0))

		if err != nil {
			return nil, err
		}
	} else if _, ok := basicTypes[obj.Index(0).Kind()]; !ok {
		return nil, fmt.Errorf("Slices/Arrays can only have base types %s. Slice/Array has basic type %s", listBasicTypes(), obj.Index(0).Kind().String())
	} else {
		lowerSchema = basicTypes[obj.Index(0).Kind()].getSchema()
	}

	return spec.ArrayProperty(lowerSchema), nil
}

func getSchema(field reflect.Type) (*spec.Schema, error) {
	if bt, ok := basicTypes[field.Kind()]; !ok {
		if field.Kind() == reflect.Array {
			return buildArraySchema(reflect.New(field).Elem())
		} else if field.Kind() == reflect.Slice {
			return buildSliceSchema(reflect.MakeSlice(field, 1, 1))
		} else {
			return nil, fmt.Errorf("%s was not a valid basic type", field.String())
		}
	} else {
		return bt.getSchema(), nil
	}
}
