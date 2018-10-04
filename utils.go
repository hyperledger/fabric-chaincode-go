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
	"reflect"
	"strconv"
	"strings"
)

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

// Types
type basicType interface {
	convert(string) (reflect.Value, error)
	getType() reflect.Type
}

type stringType struct{}

func (st *stringType) convert(value string) (reflect.Value, error) {
	return reflect.ValueOf(value), nil
}

func (st *stringType) getType() reflect.Type {
	return reflect.TypeOf("")
}

type boolType struct{}

func (st *boolType) convert(value string) (reflect.Value, error) {
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

func (st *boolType) getType() reflect.Type {
	return reflect.TypeOf(true)
}

type intType struct{}

func (st *intType) convert(value string) (reflect.Value, error) {
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

func (st *intType) getType() reflect.Type {
	return reflect.TypeOf(1)
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

func (it *int8Type) getType() reflect.Type {
	return reflect.TypeOf(int8(1))
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

func (it *int16Type) getType() reflect.Type {
	return reflect.TypeOf(int16(1))
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

func (it *int32Type) getType() reflect.Type {
	return reflect.TypeOf(int32(1))
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

func (it *int64Type) getType() reflect.Type {
	return reflect.TypeOf(int64(1))
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

func (ut *uintType) getType() reflect.Type {
	return reflect.TypeOf(uint(1))
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

func (ut *uint8Type) getType() reflect.Type {
	return reflect.TypeOf(uint8(1))
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

func (ut *uint16Type) getType() reflect.Type {
	return reflect.TypeOf(uint16(1))
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

func (ut *uint32Type) getType() reflect.Type {
	return reflect.TypeOf(uint32(1))
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

func (ut *uint64Type) getType() reflect.Type {
	return reflect.TypeOf(uint64(1))
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

func (ft *float32Type) getType() reflect.Type {
	return reflect.TypeOf(float32(1))
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

func (ft *float64Type) getType() reflect.Type {
	return reflect.TypeOf(float64(1))
}
