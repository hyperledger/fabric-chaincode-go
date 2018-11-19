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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
)

const standardAssetID = "ABC123"
const standardValue = "100"
const standardTxID = "txID"
const standardMSPID = "SampleOrg"

const invokeType = "INVOKE"
const initType = "INIT"

var badType = reflect.TypeOf(complex64(1))
var badArrayType = reflect.TypeOf([1]complex64{})
var badSliceType = reflect.TypeOf([]complex64{})

var errorType = reflect.TypeOf((*error)(nil)).Elem()

var basicContextType = reflect.TypeOf(TransactionContext{})
var basicContextPtrType = reflect.TypeOf(new(TransactionContext))

var stringTypeVar = new(stringType)
var boolTypeVar = new(boolType)
var intTypeVar = new(intType)
var int8TypeVar = new(int8Type)
var int16TypeVar = new(int16Type)
var int32TypeVar = new(int32Type)
var int64TypeVar = new(int64Type)
var uintTypeVar = new(uintType)
var uint8TypeVar = new(uint8Type)
var uint16TypeVar = new(uint16Type)
var uint32TypeVar = new(uint32Type)
var uint64TypeVar = new(uint64Type)
var float32TypeVar = new(float32Type)
var float64TypeVar = new(float64Type)

var boolRefType = reflect.TypeOf(true)
var stringRefType = reflect.TypeOf("")
var intRefType = reflect.TypeOf(1)
var int8RefType = reflect.TypeOf(int8(1))
var int16RefType = reflect.TypeOf(int16(1))
var int32RefType = reflect.TypeOf(int32(1))
var int64RefType = reflect.TypeOf(int64(1))
var uintRefType = reflect.TypeOf(uint(1))
var uint8RefType = reflect.TypeOf(uint8(1))
var uint16RefType = reflect.TypeOf(uint16(1))
var uint32RefType = reflect.TypeOf(uint32(1))
var uint64RefType = reflect.TypeOf(uint64(1))
var float32RefType = reflect.TypeOf(float32(1.0))
var float64RefType = reflect.TypeOf(1.0)

var standardExtras = []string{"Extra1", "Extra2"}

type osExcTestStr struct{}

func (o osExcTestStr) Executable() (string, error) {
	return "", errors.New("some error")
}

func testConvertError(t *testing.T, bt basicType, toPass string, expectedType string) {
	t.Helper()

	val, err := bt.convert(toPass)
	assert.EqualError(t, err, fmt.Sprintf("Cannot convert passed value %s to %s", toPass, expectedType), "should return error for invalid value")
	assert.Equal(t, reflect.Value{}, val, "should have returned the blank value")
}

func testBuildArrayOrSliceSchema(t *testing.T, toTest interface{}, expectedSchema *Schema) {
	t.Helper()

	arr := reflect.ValueOf(toTest)

	schema, err := buildArrayOrSliceSchema(arr)

	assert.Nil(t, err, "should not return error for valid array")
	assert.Equal(t, expectedSchema, schema, "should have returned expected schema")
}

func testGetSchema(t *testing.T, typ reflect.Type, expectedSchema *Schema) {
	var schema *Schema
	var err error

	t.Helper()

	schema, err = getSchema(typ)

	assert.Nil(t, err, "err should be nil when not erroring")
	assert.Equal(t, expectedSchema, schema, "should return expected schema for type")
}

func testArrayOfValidTypeIsValid(t *testing.T, arr interface{}) {
	t.Helper()

	err := arrayOfValidType(reflect.ValueOf(arr))

	assert.Nil(t, err, "should not return error for basic type")
}

func testSliceOfValidTypeIsValid(t *testing.T, arr interface{}) {
	t.Helper()

	err := sliceOfValidType(reflect.ValueOf(arr))

	assert.Nil(t, err, "should not return error for basic type")
}

func generateMethodTypesAndValuesFromName(contract ContractInterface, methodName string) (reflect.Method, reflect.Value) {
	contractT := reflect.PtrTo(reflect.TypeOf(contract).Elem())
	contractV := reflect.ValueOf(contract).Elem().Addr()

	for i := 0; i < contractT.NumMethod(); i++ {
		if contractT.Method(i).Name == methodName {
			return contractT.Method(i), contractV.Method(i)
		}
	}

	panic(fmt.Sprintf("Function with name %s does not exist for contract interface passed", methodName))
}

func generateMethodTypesAndValuesFromFunc(fn interface{}) reflect.Method {
	fnType := reflect.TypeOf(fn)
	fnValue := reflect.ValueOf(fn)

	if fnType.Kind() != reflect.Func {
		panic(fmt.Sprintf("Cannot create new contract function from %s. Can only use func", fnType.Kind()))
	}

	myMethod := reflect.Method{}
	myMethod.Func = fnValue
	myMethod.Type = fnType

	return myMethod
}

func testMethod2ContractFunctionParams(t *testing.T, funcFromStruct bool) {
	t.Helper()

	var method reflect.Method
	var methodName string
	var params contractFunctionParams
	var err error
	expectedCFParams := contractFunctionParams{}
	bc := new(badContract)
	mc := new(myContract)
	sc := new(simpleTestContractWithCustomContext)

	customCtxType := reflect.ValueOf(new(customContext)).Type()

	genericMethodName := "Function"

	// Should return error when method takes in type not in validParams
	if funcFromStruct {
		methodName = "TakesBadType"
		method, _ = generateMethodTypesAndValuesFromName(bc, methodName)
	} else {
		method = generateMethodTypesAndValuesFromFunc(bc.TakesBadType)
		methodName = genericMethodName
	}
	params, err = method2ContractFunctionParams(method, basicContextPtrType)

	assert.Equal(t, contractFunctionParams{}, params, "should return a blank contractFunctionParams")
	assert.EqualError(t, err, fmt.Sprintf("%s contains invalid parameter type. %s", methodName, typeIsValid(badType, []reflect.Type{basicContextPtrType})), "should error when param found not in validParams")

	// Should return error when method takes in intype when using custom context
	if funcFromStruct {
		methodName = "CheckContextStub"
		method, _ = generateMethodTypesAndValuesFromName(mc, methodName)
	} else {
		method = generateMethodTypesAndValuesFromFunc(mc.CheckContextStub)
		methodName = genericMethodName
	}
	params, err = method2ContractFunctionParams(method, customCtxType)

	assert.Equal(t, contractFunctionParams{}, params, "should return a blank contractFunctionParams")
	assert.EqualError(t, err, fmt.Sprintf("%s contains invalid parameter type. %s", methodName, typeIsValid(reflect.TypeOf(new(TransactionContext)), []reflect.Type{customCtxType})), "should error when param found not in validParams")

	// Should return error when method uses context but not as first arg
	if funcFromStruct {
		methodName = "TakesContextBadly"
		method, _ = generateMethodTypesAndValuesFromName(bc, methodName)
	} else {
		method = generateMethodTypesAndValuesFromFunc(bc.TakesContextBadly)
		methodName = genericMethodName
	}
	params, err = method2ContractFunctionParams(method, basicContextPtrType)

	assert.Equal(t, contractFunctionParams{}, params, "should return a blank contractFunctionParams")
	assert.EqualError(t, err, fmt.Sprintf("Functions requiring the TransactionContext must require it as the first parameter. %s takes it in as parameter 1", methodName), "should error when context used but not first arg")

	// Should return contractFunctionParams for method with no params
	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(mc, "ReturnsString")
	} else {
		method = generateMethodTypesAndValuesFromFunc(mc.ReturnsString)
	}
	params, err = method2ContractFunctionParams(method, basicContextPtrType)

	expectedCFParams.context = nil
	expectedCFParams.fields = []reflect.Type{}

	assert.Nil(t, err, "should not return err for valid method")
	assert.Nil(t, params.context, "should have set correct context in contractFunctionParams for method with no params")
	assert.Equal(t, 0, len(params.fields), "should have set correct fields in contractFunctionParams for method with no params")

	// Should return contractFunctionParams for method with params but no context
	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(mc, "NotUsesContext")
	} else {
		method = generateMethodTypesAndValuesFromFunc(mc.NotUsesContext)
	}
	params, err = method2ContractFunctionParams(method, basicContextPtrType)

	expectedCFParams.context = nil
	expectedCFParams.fields = []reflect.Type{
		stringRefType,
		stringRefType,
	}

	assert.Nil(t, err, "should not return err for valid method")
	assert.Equal(t, expectedCFParams, params, "should have set correct contractFunctionParams for method with params but no context")

	// Should return contractFunctionParams for method with params and context
	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(mc, "UsesContext")
	} else {
		method = generateMethodTypesAndValuesFromFunc(mc.UsesContext)
	}
	params, err = method2ContractFunctionParams(method, basicContextPtrType)

	expectedCFParams.context = basicContextPtrType
	expectedCFParams.fields = []reflect.Type{
		stringRefType,
		stringRefType,
	}

	assert.Nil(t, err, "should not return err for valid method")
	assert.Equal(t, expectedCFParams, params, "should have set correct contractFunctionParams for method with context")

	// Should return contractFunctionParams for method with params that are of basic types
	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(mc, "UsesBasics")
	} else {
		method = generateMethodTypesAndValuesFromFunc(mc.UsesBasics)
	}
	params, err = method2ContractFunctionParams(method, basicContextPtrType)

	expectedCFParams.context = nil
	expectedCFParams.fields = []reflect.Type{
		stringRefType,
		boolRefType,
		intRefType,
		int8RefType,
		int16RefType,
		int32RefType,
		int64RefType,
		uintRefType,
		uint8RefType,
		uint16RefType,
		uint32RefType,
		uint64RefType,
		float32RefType,
		float64RefType,
		reflect.TypeOf(byte(1)),
		reflect.TypeOf(rune(1)),
	}

	assert.Nil(t, err, "should not return err for valid method")
	assert.Equal(t, expectedCFParams, params, "should have set correct contractFunctionParams for func withbasic types")

	// Should return error for a method that takes an array not of basic type
	if funcFromStruct {
		methodName = "TakesBadArray"
		method, _ = generateMethodTypesAndValuesFromName(bc, "TakesBadArray")
	} else {
		methodName = genericMethodName
		method = generateMethodTypesAndValuesFromFunc(bc.TakesBadArray)
	}
	params, err = method2ContractFunctionParams(method, basicContextPtrType)

	assert.Equal(t, fmt.Errorf("%s contains invalid parameter type. %s", methodName, typeIsValid(badArrayType, []reflect.Type{basicContextPtrType})), err, "should return err for invalid method with bad array")
	assert.Equal(t, contractFunctionParams{}, params, "should return a blank contractFunctionParams for func taking bad array")

	// Should return error for a method that takes a slice not of basic type
	if funcFromStruct {
		methodName = "TakesBadSlice"
		method, _ = generateMethodTypesAndValuesFromName(bc, "TakesBadSlice")
	} else {
		methodName = genericMethodName
		method = generateMethodTypesAndValuesFromFunc(bc.TakesBadSlice)
	}
	params, err = method2ContractFunctionParams(method, basicContextPtrType)

	assert.Equal(t, fmt.Errorf("%s contains invalid parameter type. %s", methodName, typeIsValid(badSliceType, []reflect.Type{basicContextPtrType})), err, "should return err for invalid method with bad slice")
	assert.Equal(t, contractFunctionParams{}, params, "should return a blank contractFunctionParams for func taking bad slice")

	// Should return contractFunctionParams for method with custom context
	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(sc, "CheckCustomContext")
	} else {
		method = generateMethodTypesAndValuesFromFunc(sc.CheckCustomContext)
	}
	params, err = method2ContractFunctionParams(method, customCtxType)

	expectedCFParams.context = customCtxType
	expectedCFParams.fields = []reflect.Type{}

	assert.Nil(t, err, "should not return err for valid method")
	assert.Equal(t, expectedCFParams.context, params.context, "should have set correct takesContext contractFunctionParams for method with custom context")
	assert.Equal(t, 0, len(params.fields), "should have set correct fields contractFunctionParams for method with custom context")
}

func testMethod2ContractFunctionReturnsSingleType(t *testing.T, funcFromStruct bool, testFunction reflect.Method, testFunctionName string, expectedSuccessType reflect.Type) {
	t.Helper()

	var method reflect.Method
	mc := new(myContract)

	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(mc, testFunctionName)
	} else {
		method = testFunction
	}

	expectedCFReturns := contractFunctionReturns{}
	expectedCFReturns.success = expectedSuccessType
	expectedCFReturns.error = false

	returns, err := method2ContractFunctionReturns(method)

	assert.Nil(t, err, "should not return error for valid return type")
	assert.Equal(t, expectedCFReturns, returns, fmt.Sprintf("should set success to %s type and error false for function that returns only %s", expectedSuccessType.String(), expectedSuccessType.String()))
}

func testMethod2ContractFunctionReturns(t *testing.T, funcFromStruct bool) {
	t.Helper()

	var method reflect.Method
	var methodName string
	var returns contractFunctionReturns
	var err error
	var expectedCFReturns contractFunctionReturns
	bc := new(badContract)
	mc := new(myContract)

	genericMethodName := "Function"

	// Should return error when returns a single value and it is not a valid type
	if funcFromStruct {
		methodName = "ReturnsBadType"
		method, _ = generateMethodTypesAndValuesFromName(bc, methodName)
	} else {
		method = generateMethodTypesAndValuesFromFunc(bc.ReturnsBadType)
		methodName = genericMethodName
	}

	returns, err = method2ContractFunctionReturns(method)

	assert.Equal(t, contractFunctionReturns{}, returns, "should return a blank contractFunctionReturns")
	assert.EqualError(t, err, fmt.Sprintf("%s contains invalid single return type. %s", methodName, typeIsValid(badType, []reflect.Type{errorType})), "should return expected error for using a bad type")

	// Should return error when returning two types and they are in the wrong order
	if funcFromStruct {
		methodName = "ReturnsWrongOrder"
		method, _ = generateMethodTypesAndValuesFromName(bc, methodName)
	} else {
		method = generateMethodTypesAndValuesFromFunc(bc.ReturnsWrongOrder)
		methodName = genericMethodName
	}

	returns, err = method2ContractFunctionReturns(method)

	assert.Equal(t, contractFunctionReturns{}, returns, "should return a blank contractFunctionParams")

	assert.EqualError(t, err, fmt.Sprintf("%s contains invalid first return type. Type error is not valid. Expected one of the basic types %s or an array/slice of these", methodName, listBasicTypes()), "should return expected error for bad first return type")

	// Should return error when returning two types and first return type is bad
	if funcFromStruct {
		methodName = "ReturnsBadTypeAndError"
		method, _ = generateMethodTypesAndValuesFromName(bc, methodName)
	} else {
		method = generateMethodTypesAndValuesFromFunc(bc.ReturnsBadTypeAndError)
		methodName = genericMethodName
	}

	returns, err = method2ContractFunctionReturns(method)

	assert.Equal(t, contractFunctionReturns{}, returns, "should return a blank contractFunctionParams")
	assert.EqualError(t, err, fmt.Sprintf("%s contains invalid first return type. %s", methodName, typeIsValid(badType, []reflect.Type{})), "should return expected error for bad first return type")

	// Should return error when returning two types and second return type is bad
	if funcFromStruct {
		methodName = "ReturnsStringAndInt"
		method, _ = generateMethodTypesAndValuesFromName(bc, methodName)
	} else {
		method = generateMethodTypesAndValuesFromFunc(bc.ReturnsStringAndInt)
		methodName = genericMethodName
	}

	returns, err = method2ContractFunctionReturns(method)

	assert.Equal(t, contractFunctionReturns{}, returns, "should return a blank contractFunctionParams")
	assert.EqualError(t, err, fmt.Sprintf("%s contains invalid second return type. Type int is not valid. Expected error", methodName))

	// Should return error when returning more than two types
	if funcFromStruct {
		methodName = "ReturnsStringErrorAndInt"
		method, _ = generateMethodTypesAndValuesFromName(bc, methodName)
	} else {
		method = generateMethodTypesAndValuesFromFunc(bc.ReturnsStringErrorAndInt)
		methodName = genericMethodName
	}

	returns, err = method2ContractFunctionReturns(method)

	assert.Equal(t, contractFunctionReturns{}, returns, "should return a blank contractFunctionParams")
	assert.EqualError(t, err, fmt.Sprintf("Functions may only return a maximum of two values. %s returns 3", methodName))

	// Should return contractFunctionReturns for no return types
	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(mc, "ReturnsNothing")
	} else {
		method = generateMethodTypesAndValuesFromFunc(mc.ReturnsNothing)
	}

	expectedCFReturns = contractFunctionReturns{}

	returns, err = method2ContractFunctionReturns(method)

	assert.Nil(t, err, "should not return error for valid return type")
	assert.Equal(t, expectedCFReturns, returns, "should set success to empty type and error false for function that returns no types")

	var funcToTest reflect.Method

	// Should return contractFunctionReturns for single value return of type string
	funcToTest = generateMethodTypesAndValuesFromFunc(mc.ReturnsString)
	testMethod2ContractFunctionReturnsSingleType(t, funcFromStruct, funcToTest, "ReturnsString", stringRefType)

	// Should return contractFunctionReturns for a basic type return value
	funcToTest = generateMethodTypesAndValuesFromFunc(mc.ReturnsInt)
	testMethod2ContractFunctionReturnsSingleType(t, funcFromStruct, funcToTest, "ReturnsInt", intRefType)

	// Should return contractFunctionReturns for a array of basic type return value
	funcToTest = generateMethodTypesAndValuesFromFunc(mc.ReturnsArray)
	testMethod2ContractFunctionReturnsSingleType(t, funcFromStruct, funcToTest, "ReturnsArray", reflect.TypeOf(mc.ReturnsArray()))

	// Should return contractFunctionReturns for a slice of basic type return value
	funcToTest = generateMethodTypesAndValuesFromFunc(mc.ReturnsSlice)
	testMethod2ContractFunctionReturnsSingleType(t, funcFromStruct, funcToTest, "ReturnsSlice", reflect.TypeOf(mc.ReturnsSlice()))

	// Should return contractFunctionReturns for single value return of type error
	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(mc, "ReturnsError")
	} else {
		method = generateMethodTypesAndValuesFromFunc(mc.ReturnsError)
	}

	expectedCFReturns = contractFunctionReturns{}
	expectedCFReturns.error = true

	returns, err = method2ContractFunctionReturns(method)

	assert.Nil(t, err, "should not return error for valid return type")
	assert.Equal(t, expectedCFReturns, returns, "should set string false and error true for function that returns only error")

	// Should return contractFunctionReturns for double value return of type string and error
	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(mc, "ReturnsStringAndError")
	} else {
		method = generateMethodTypesAndValuesFromFunc(mc.ReturnsStringAndError)
	}

	expectedCFReturns.success = stringRefType
	expectedCFReturns.error = true

	returns, err = method2ContractFunctionReturns(method)

	assert.Nil(t, err, "should not return error for valid return type")
	assert.Equal(t, expectedCFReturns, returns, "should set string true and error true for function that returns string and error")
}

func testParseMethod(t *testing.T, funcFromStruct bool) {
	t.Helper()

	var method reflect.Method
	var params contractFunctionParams
	var returns contractFunctionReturns
	var err error
	var expectedErr error
	var expectedCFParams contractFunctionParams
	var expectedCFReturns contractFunctionReturns
	bc := new(badContract)
	mc := new(myContract)

	// Should return error returned by method2ContractFunctionParams for invalid params
	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(bc, "TakesBadType")
	} else {
		method = generateMethodTypesAndValuesFromFunc(bc.TakesBadType)
	}
	_, expectedErr = method2ContractFunctionParams(method, basicContextPtrType)

	params, returns, err = parseMethod(method, basicContextPtrType)

	assert.Equal(t, contractFunctionParams{}, params, "should return a blank contractFunctionParams")
	assert.Equal(t, contractFunctionReturns{}, returns, "should return a blank contractFunctionReturns")
	assert.EqualError(t, err, expectedErr.Error(), "should return same error as method2ContractFunctionParams")

	// Should return error returned by method2ContractFunctionReturns for invalid return types
	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(bc, "ReturnsBadType")
	} else {
		method = generateMethodTypesAndValuesFromFunc(bc.ReturnsBadType)
	}
	_, expectedErr = method2ContractFunctionReturns(method)

	params, returns, err = parseMethod(method, basicContextPtrType)

	assert.Equal(t, contractFunctionParams{}, params, "should return a blank contractFunctionParams")
	assert.Equal(t, contractFunctionReturns{}, returns, "should return a blank contractFunctionReturns")
	assert.EqualError(t, err, expectedErr.Error(), "should return same error as method2ContractFunctionReturns")

	// Should return the parsed params and returns
	if funcFromStruct {
		method, _ = generateMethodTypesAndValuesFromName(mc, "ReturnsString")
	} else {
		method = generateMethodTypesAndValuesFromFunc(mc.ReturnsString)
	}
	expectedCFParams, _ = method2ContractFunctionParams(method, basicContextPtrType)
	expectedCFReturns, _ = method2ContractFunctionReturns(method)

	params, returns, err = parseMethod(method, basicContextPtrType)

	assert.Equal(t, expectedCFParams, params, "should return same contractFunctionParams as method2ContractFunctionParams")
	assert.Equal(t, expectedCFReturns, returns, "should return same contractFunctionReturns as method2ContractFunctionReturns")
	assert.Nil(t, err, "should return nil when valid params and return types")
}

func testContractFunctionUsingReturnsString(t *testing.T, mc *myContract, cf *contractFunction) {
	t.Helper()
	expectedResp := mc.ReturnsString()
	actualResp := cf.function.Call([]reflect.Value{})[0].Interface().(string)

	assert.Equal(t, expectedResp, actualResp, "should set reflect value of function as function")

	expectedCFParams := contractFunctionParams{}
	expectedCFParams.context = nil
	expectedCFParams.fields = []reflect.Type{}

	assert.Nil(t, cf.params.context, "should have set correct params takesContext")
	assert.Equal(t, 0, len(cf.params.fields), "should have set correct params fields")

	expectedCFReturns := contractFunctionReturns{}
	expectedCFReturns.success = reflect.TypeOf(expectedResp)
	expectedCFReturns.error = false

	assert.Equal(t, expectedCFReturns, cf.returns, "should have correct return")
}

func testCreateArrayOrSliceErrors(t *testing.T, json string, arrType reflect.Type) {
	t.Helper()

	val, err := createArrayOrSlice(json, arrType)

	assert.EqualError(t, err, fmt.Sprintf("Value %s was not passed in expected format %s", json, arrType.String()), "should error when invalid JSON")
	assert.Equal(t, reflect.Value{}, val, "should return an empty value when error found")
}

func setContractFunctionParams(cf *contractFunction, context reflect.Type, fields []reflect.Type) {
	cfp := contractFunctionParams{}

	cfp.context = context
	cfp.fields = fields
	cf.params = cfp
}

func setContractFunctionReturns(cf *contractFunction, successReturn reflect.Type, returnsError bool) {
	cfr := contractFunctionReturns{}
	cfr.success = successReturn
	cfr.error = returnsError

	cf.returns = cfr
}

func callGetArgsAndBasicTest(t *testing.T, cf contractFunction, ctx *TransactionContext, testParams []string) []reflect.Value {
	t.Helper()

	values, err := getArgs(cf, reflect.ValueOf(ctx), testParams)

	assert.Nil(t, err, "should not return an error for a valid cf")

	if cf.params.context != nil {
		assert.Equal(t, len(cf.params.fields)+1, len(values), "should return same length array list as number of fields plus 1 for context")
	} else {
		assert.Equal(t, len(cf.params.fields), len(values), "should return same length array list as number of fields")
	}

	return values
}

func testReflectValueEqualSlice(t *testing.T, values []reflect.Value, expectedValues interface{}) {
	t.Helper()

	s := reflect.ValueOf(expectedValues)
	expectedArr := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		expectedArr[i] = s.Index(i).Interface()
	}

	for index, value := range values {
		assert.Equal(t, fmt.Sprintf("%v", value), fmt.Sprintf("%v", expectedArr[index]), "should return params in order passed")
	}
}

func testGetArgsWithTypes(t *testing.T, types map[reflect.Kind]interface{}, params []string) {
	t.Helper()

	cf := contractFunction{}
	ctx := new(TransactionContext)

	for kind, expectedArgs := range types {
		var typ reflect.Type

		switch kind {
		case reflect.Bool:
			typ = boolRefType
		case reflect.String:
			typ = stringRefType
		case reflect.Int:
			typ = intRefType
		case reflect.Int8:
			typ = int8RefType
		case reflect.Int16:
			typ = int16RefType
		case reflect.Int32:
			typ = int32RefType
		case reflect.Int64:
			typ = int64RefType
		case reflect.Uint:
			typ = uintRefType
		case reflect.Uint8:
			typ = uint8RefType
		case reflect.Uint16:
			typ = uint16RefType
		case reflect.Uint32:
			typ = uint32RefType
		case reflect.Uint64:
			typ = uint64RefType
		case reflect.Float32:
			typ = float32RefType
		case reflect.Float64:
			typ = float64RefType
		}

		setContractFunctionParams(&cf, nil, []reflect.Type{
			typ,
		})

		values := callGetArgsAndBasicTest(t, cf, ctx, params)
		testReflectValueEqualSlice(t, values, expectedArgs)
	}
}

func compareJSON(t *testing.T, actual []byte, expected []byte) {
	t.Helper()

	actualMap := make(map[string]interface{})
	expectedMap := make(map[string]interface{})

	err := json.Unmarshal(actual, &actualMap)
	if err != nil {
		assert.Fail(t, err.Error())
	}
	err = json.Unmarshal(expected, &expectedMap)
	if err != nil {
		assert.Fail(t, err.Error())
	}

	assert.Equal(t, actualMap, expectedMap, "JSONs to compare should have been equal")
}

func createMetadataJSONFile(data []byte, permissions os.FileMode) string {
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)

	folderPath := filepath.Join(exPath, metadataFolder)
	filePath := filepath.Join(folderPath, metadataFile)

	os.Mkdir(folderPath, os.ModePerm)
	ioutil.WriteFile(filePath, data, permissions)

	return filePath
}

func cleanupMetadataJSONFile() {
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)

	folderPath := filepath.Join(exPath, metadataFolder)

	os.RemoveAll(folderPath)
}

func testMetadata(t *testing.T, metadata string, expectedMetadata ContractChaincodeMetadata) {
	t.Helper()

	contractChaincodeMetadata := ContractChaincodeMetadata{}

	err := json.Unmarshal([]byte(metadata), &contractChaincodeMetadata)

	assert.Nil(t, err, "Should be able to unmarshal metadata")
	assert.Equal(t, expectedMetadata, contractChaincodeMetadata, "Should match expected metadata")
}

func testContractChaincodeContractRepresentsContract(t *testing.T, ccns contractChaincodeContract, contract simpleTestContract) {
	t.Helper()

	assert.Equal(t, len(expectedSimpleContractFuncs), len(ccns.functions), "should only have one function as simpleTestContract")

	assert.Equal(t, ccns.functions["DoSomething"].params, contractFunctionParams{nil, nil}, "should set correct params for contract function")
	assert.Equal(t, ccns.functions["DoSomething"].returns, contractFunctionReturns{stringRefType, true}, "should set correct returns for contract function")

	transactionContextHandler := reflect.ValueOf(contract.GetTransactionContextHandler()).Elem().Type()
	transactionContextPtrHandler := reflect.ValueOf(contract.GetTransactionContextHandler()).Type()

	assert.Equal(t, ccns.transactionContextHandler, transactionContextHandler, "should have correct transaction context set")
	assert.Equal(t, ccns.transactionContextPtrHandler, transactionContextPtrHandler, "should have correct transaction context set")

	ut, err := contract.GetUnknownTransaction()

	if err != nil {
		assert.Nil(t, ccns.unknownTransaction, "should be nil when contract has no unknown transaction")
	} else {
		assert.Equal(t, ccns.unknownTransaction, newContractFunctionFromFunc(ut, transactionContextPtrHandler), "should have set correct unknown transaction when set")
	}

	bt, err := contract.GetBeforeTransaction()

	if err != nil {
		assert.Nil(t, ccns.beforeTransaction, "should be nil when contract has no before transaction")
	} else {
		assert.Equal(t, ccns.beforeTransaction, newContractFunctionFromFunc(bt, transactionContextPtrHandler), "should have set correct before transaction when set")
	}

	at, err := contract.GetAfterTransaction()

	if err != nil {
		assert.Nil(t, ccns.afterTransaction, "should be nil when contract has no after transaction")
	} else {
		assert.Equal(t, ccns.afterTransaction, newContractFunctionFromFunc(at, transactionContextPtrHandler), "should have set correct after transaction when set")
	}
}

func testConvertCC(t *testing.T, testData []simpleTestContract) {
	t.Helper()

	contractInterfaces := []ContractInterface{}

	for i := 0; i < len(testData); i++ {
		contractInterfaces = append(contractInterfaces, &testData[i])
	}

	cc := convertC2CC(contractInterfaces...)

	// Plus 1 as system contract
	assert.Equal(t, len(testData)+1, len(cc.contracts), "Didn't map correct number of smart contracts")

	expectedSysMetadata := ContractChaincodeMetadata{}

	successSchema := Schema{}
	successSchema.Type = []string{"string"}

	errorSchema := Schema{}
	errorSchema.Type = []string{"object"}
	errorSchema.Format = "error"

	successMetadata := ParameterMetadata{}
	successMetadata.Name = "success"
	successMetadata.Schema = successSchema

	errorMetadata := ParameterMetadata{}
	errorMetadata.Name = "error"
	errorMetadata.Schema = errorSchema

	simpleContractFunctionMetadata := TransactionMetadata{}
	simpleContractFunctionMetadata.Name = "DoSomething"
	simpleContractFunctionMetadata.Returns = []ParameterMetadata{successMetadata, errorMetadata}

	// Test that the data set for each contract in chaincode is correct e.g. unknown fn set etc
	for i := 0; i < len(testData); i++ {
		contract := testData[i]
		ns := contract.GetName()

		nsContract, ok := cc.contracts[ns]

		contractMetadata := ContractMetadata{}
		contractMetadata.Name = ns
		contractMetadata.Transactions = []TransactionMetadata{
			simpleContractFunctionMetadata,
		}

		expectedSysMetadata.Contracts = append(expectedSysMetadata.Contracts, contractMetadata)

		assert.True(t, ok, "should have name in map of contracts")

		// simpleTestContract should only have 1 function DoSomething
		assert.Equal(t, 1, len(nsContract.functions), "should have same number of functions as a simpleTestContract")

		testContractChaincodeContractRepresentsContract(t, nsContract, contract)
	}

	// should have system contract
	sysContract, ok := cc.contracts[SystemContractName]

	assert.True(t, ok, "should have added a system contract with other contracts")

	fn, ok := sysContract.functions["GetMetadata"]

	assert.True(t, ok, "should have GetMetadata for system contract")

	systemContractFunctionMetadata := TransactionMetadata{}
	systemContractFunctionMetadata.Name = "GetMetadata"
	systemContractFunctionMetadata.Returns = []ParameterMetadata{
		successMetadata,
	}

	systemContractMetadata := ContractMetadata{}
	systemContractMetadata.Name = SystemContractName
	systemContractMetadata.Transactions = []TransactionMetadata{
		systemContractFunctionMetadata,
	}

	expectedSysMetadata.Contracts = append(expectedSysMetadata.Contracts, systemContractMetadata)

	metadata, _ := fn.call(reflect.Value{})

	testMetadata(t, metadata, expectedSysMetadata)
}

func callContractFunctionAndCheckError(t *testing.T, cc *ContractChaincode, arguments []string, callType string, expectedMessage string) {
	t.Helper()

	callContractFunctionAndCheckResponse(t, cc, arguments, callType, expectedMessage, "error")
}

func callContractFunctionAndCheckSuccess(t *testing.T, cc *ContractChaincode, arguments []string, callType string, expectedMessage string) {
	t.Helper()

	callContractFunctionAndCheckResponse(t, cc, arguments, callType, expectedMessage, "success")
}

func callContractFunctionAndCheckResponse(t *testing.T, cc *ContractChaincode, arguments []string, callType string, expectedMessage string, expectedType string) {
	t.Helper()

	args := [][]byte{}
	for _, str := range arguments {
		arg := []byte(str)
		args = append(args, arg)
	}

	mockStub := shim.NewMockStub("smartContractTest", cc)

	var response peer.Response

	if callType == initType {
		response = mockStub.MockInit(standardTxID, args)
	} else if callType == invokeType {
		response = mockStub.MockInvoke(standardTxID, args)
	} else {
		panic(fmt.Sprintf("Call type passed should be %s or %s. Value passed was %s", initType, invokeType, callType))
	}

	expectedResponse := shim.Success([]byte(expectedMessage))

	if expectedType == "error" {
		expectedResponse = shim.Error(expectedMessage)
	}

	assert.Equal(t, expectedResponse, response)
}

func testCallingContractFunctions(t *testing.T, callType string) {
	t.Helper()

	cc := convertC2CC()

	// Should error when blank name not found
	callContractFunctionAndCheckError(t, cc, []string{"somebadfunctionname"}, callType, "No contract found without name")

	mc := myContract{}
	cc = convertC2CC(&mc)

	// Should error when name not known
	callContractFunctionAndCheckError(t, cc, []string{"somebadname:somebadfunctionname"}, callType, "Name not found somebadname")

	// should return error when function not known and no unknown transaction specified
	callContractFunctionAndCheckError(t, cc, []string{"somebadfunctionname"}, callType, "Function somebadfunctionname not found for contract with no name")

	// should return error when function not known and no unknown transaction specified for custom name
	mc.SetName("customname")
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckError(t, cc, []string{"customname:somebadfunctionname"}, callType, "Function somebadfunctionname not found in name customname")
	mc = myContract{}
	cc = convertC2CC(&mc)

	// Should return success when function returns nothing
	callContractFunctionAndCheckSuccess(t, cc, []string{"ReturnsNothing"}, callType, "")

	// should return success when function returns no error
	callContractFunctionAndCheckSuccess(t, cc, []string{"ReturnsString"}, callType, mc.ReturnsString())

	// Should return error when function returns error
	callContractFunctionAndCheckError(t, cc, []string{"ReturnsError"}, callType, mc.ReturnsError().Error())

	// Should return error when function unknown and set unknown function returns error
	mc.SetUnknownTransaction(mc.ReturnsError)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckError(t, cc, []string{"somebadfunctionname"}, callType, mc.ReturnsError().Error())
	mc = myContract{}

	// Should return success when function unknown and set unknown function returns no error
	mc.SetUnknownTransaction(mc.ReturnsString)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"somebadfunctionname"}, callType, mc.ReturnsString())
	mc = myContract{}

	// Should return error when before function returns error and not call main function
	mc.SetBeforeTransaction(mc.ReturnsError)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckError(t, cc, []string{"ReturnsString"}, callType, mc.ReturnsError().Error())
	mc = myContract{}

	// Should return success from passed function when before function returns no error
	mc.SetBeforeTransaction(mc.ReturnsString)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"ReturnsString"}, callType, mc.ReturnsString())
	mc = myContract{}

	// Should return error when after function returns error
	mc.SetAfterTransaction(mc.ReturnsError)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckError(t, cc, []string{"ReturnsString"}, callType, mc.ReturnsError().Error())
	mc = myContract{}

	// Should return success from passed function when before function returns error
	mc.SetAfterTransaction(mc.ReturnsString)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"ReturnsString"}, callType, mc.ReturnsString())
	mc = myContract{}

	// Should call before, named then after functions in order
	mc.SetBeforeTransaction(mc.logBefore)
	mc.SetAfterTransaction(mc.logAfter)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"LogNamed"}, callType, "")
	assert.Equal(t, []string{"Before function called", "Named function called", "After function called"}, mc.called, "Expected called field of myContract to have logged in order before, named then after")
	mc = myContract{}

	// Should call before, unknown then after functions in order
	mc.SetBeforeTransaction(mc.logBefore)
	mc.SetAfterTransaction(mc.logAfter)
	mc.SetUnknownTransaction(mc.logUnknown)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"somebadfunctionname"}, callType, "")
	assert.Equal(t, []string{"Before function called", "Unknown function called", "After function called"}, mc.called, "Expected called field of myContract to have logged in order before, named then after")
	mc = myContract{}

	// should pass the stub into transaction context as expected
	callContractFunctionAndCheckSuccess(t, cc, []string{"CheckContextStub"}, callType, "Stub as expected")

	sc := simpleTestContractWithCustomContext{}
	sc.SetTransactionContextHandler(new(customContext))
	cc = convertC2CC(&sc)

	//should use a custom transaction context when one is set
	callContractFunctionAndCheckSuccess(t, cc, []string{"CheckCustomContext"}, callType, "I am custom context")

	//should use same ctx for all calls
	sc.SetBeforeTransaction(sc.SetValInCustomContext)
	cc = convertC2CC(&sc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"GetValInCustomContext", standardValue}, callType, standardValue)

	sc.SetAfterTransaction(sc.GetValInCustomContext)
	cc = convertC2CC(&sc)
	callContractFunctionAndCheckError(t, cc, []string{"SetValInCustomContext", "some other value"}, callType, "I wanted a standard value")
}

// ============== utils.go ==============
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
	var expectedSchema *Schema
	var actualSchema *Schema

	// string
	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"string"}
	actualSchema = stringTypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back string schema")

	// bool
	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"boolean"}
	actualSchema = boolTypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back bool schema")

	// ints
	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"integer"}
	expectedSchema.Format = "int64"
	actualSchema = intTypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back int schema")

	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"integer"}
	expectedSchema.Format = "int32"
	expectedSchema.Minimum = -128
	expectedSchema.Maximum = 127
	actualSchema = int8TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back int8 schema")

	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"integer"}
	expectedSchema.Format = "int32"
	expectedSchema.Minimum = -32768
	expectedSchema.Maximum = 32767
	actualSchema = int16TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back int16 schema")

	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"integer"}
	expectedSchema.Format = "int32"
	actualSchema = int32TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back int32 schema")

	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"integer"}
	expectedSchema.Format = "int64"
	actualSchema = int64TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back int64 schema")

	// uints
	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"number"}
	expectedSchema.Format = "float64"
	expectedSchema.Minimum = 0
	expectedSchema.Maximum = 18446744073709551615
	actualSchema = uintTypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back uint schema")

	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"integer"}
	expectedSchema.Format = "int32"
	expectedSchema.Minimum = 0
	expectedSchema.Maximum = 255
	actualSchema = uint8TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back uint8 schema")

	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"integer"}
	expectedSchema.Format = "int64"
	expectedSchema.Minimum = 0
	expectedSchema.Maximum = 65535
	actualSchema = uint16TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back uint16 schema")

	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"integer"}
	expectedSchema.Format = "int64"
	expectedSchema.Minimum = 0
	expectedSchema.Maximum = 4294967295
	actualSchema = uint32TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back uint32 schema")

	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"number"}
	expectedSchema.Format = "float64"
	expectedSchema.Minimum = 0
	expectedSchema.Maximum = 18446744073709551615
	actualSchema = uint64TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back uint64 schema")

	// floats
	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"number"}
	expectedSchema.Format = "float32"
	actualSchema = float32TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back float32 schema")

	expectedSchema = new(Schema)
	expectedSchema.Type = []string{"number"}
	expectedSchema.Format = "float64"
	actualSchema = float64TypeVar.getSchema()

	assert.Equal(t, expectedSchema, actualSchema, "should give back float64 schema")
}

func TestNewArraySchema(t *testing.T) {
	// Should return a schema of type array with schema in items
	lowerSchema := new(Schema)
	lowerSchema.Type = []string{"string"}

	schema := newArraySchema(lowerSchema)

	assert.Equal(t, StringOrArray([]string{"array"}), schema.Type, "should have set type to array")
	assert.Equal(t, lowerSchema, schema.Items.Schema, "should have set items schema")
}

func TestBuildArraySchema(t *testing.T) {
	var schema *Schema
	var err error

	// Should return an error when array is passed with a length of zero
	zeroArr := [0]int{}
	schema, err = buildArraySchema(reflect.ValueOf(zeroArr))

	assert.Equal(t, errors.New("Arrays must have length greater than 0"), err, "should throw error when 0 length array passed")
	assert.Nil(t, schema, "should not have returned a schema for zero array")

	// Should return error when buildArrayOrSliceSchema would
	schema, err = buildArraySchema(reflect.ValueOf([1]myContract{}))
	_, expectedErr := buildArrayOrSliceSchema(reflect.ValueOf([1]myContract{}))

	assert.Nil(t, schema, "spec should be nil when buildArrayOrSliceSchema fails from buildArraySchema")
	assert.Equal(t, expectedErr, err, "should have same error as buildArrayOrSliceSchema")
}

func TestBuildSliceSchema(t *testing.T) {
	var schema *Schema
	var err error

	// Should handle adding to the length of the slice if currently 0
	assert.NotPanics(t, func() { buildSliceSchema(reflect.ValueOf([]string{})) }, "shouldn't have panicked when slice sent was empty")

	// Should return error when buildArrayOrSliceSchema would
	schema, err = buildSliceSchema(reflect.ValueOf([]myContract{}))
	_, expectedErr := buildArrayOrSliceSchema(reflect.ValueOf([]myContract{myContract{}}))

	assert.Nil(t, schema, "spec should be nil when buildArrayOrSliceSchema fails from buildSliceSchema")
	assert.Equal(t, expectedErr, err, "should have same error as buildArrayOrSliceSchema")
}

func TestBuildArrayOrSliceSchema(t *testing.T) {
	var err error
	var schema *Schema

	stringArraySchema := newArraySchema(stringTypeVar.getSchema())
	boolArraySchema := newArraySchema(boolTypeVar.getSchema())
	intArraySchema := newArraySchema(intTypeVar.getSchema())
	int8ArraySchema := newArraySchema(int8TypeVar.getSchema())
	int16ArraySchema := newArraySchema(int16TypeVar.getSchema())
	int32ArraySchema := newArraySchema(int32TypeVar.getSchema())
	int64ArraySchema := newArraySchema(int64TypeVar.getSchema())
	uintArraySchema := newArraySchema(uintTypeVar.getSchema())
	uint8ArraySchema := newArraySchema(uint8TypeVar.getSchema())
	uint16ArraySchema := newArraySchema(uint16TypeVar.getSchema())
	uint32ArraySchema := newArraySchema(uint32TypeVar.getSchema())
	uint64ArraySchema := newArraySchema(uint64TypeVar.getSchema())
	float32ArraySchema := newArraySchema(float32TypeVar.getSchema())
	float64ArraySchema := newArraySchema(float64TypeVar.getSchema())

	validParams := make([]string, 0, len(basicTypes))
	for k := range basicTypes {
		validParams = append(validParams, k.String())
	}
	sort.Strings(validParams)

	// Should return error when array is not one of the basic types
	badArr := [1]myContract{}
	schema, err = buildArrayOrSliceSchema(reflect.ValueOf(badArr))

	assert.Equal(t, fmt.Errorf("Slices/Arrays can only have base types %s. Slice/Array has basic type struct", listBasicTypes()), err, "should throw error when invalid type passed")
	assert.Nil(t, schema, "should not have returned a schema for an array of bad type")

	// Should return an error when array is passed with sub array with a length of zero
	zeroSubArr := [1][0]int{}
	schema, err = buildArrayOrSliceSchema(reflect.ValueOf(zeroSubArr))

	assert.Equal(t, errors.New("Arrays must have length greater than 0"), err, "should throw error when 0 length array passed")
	assert.Nil(t, schema, "should not have returned a schema for zero array")

	// Should return schema for arrays made of each of the basic types
	testBuildArrayOrSliceSchema(t, [1]string{}, stringArraySchema)
	testBuildArrayOrSliceSchema(t, [1]bool{}, boolArraySchema)
	testBuildArrayOrSliceSchema(t, [1]int{}, intArraySchema)
	testBuildArrayOrSliceSchema(t, [1]int8{}, int8ArraySchema)
	testBuildArrayOrSliceSchema(t, [1]int16{}, int16ArraySchema)
	testBuildArrayOrSliceSchema(t, [1]int32{}, int32ArraySchema)
	testBuildArrayOrSliceSchema(t, [1]int64{}, int64ArraySchema)
	testBuildArrayOrSliceSchema(t, [1]uint{}, uintArraySchema)
	testBuildArrayOrSliceSchema(t, [1]uint8{}, uint8ArraySchema)
	testBuildArrayOrSliceSchema(t, [1]uint16{}, uint16ArraySchema)
	testBuildArrayOrSliceSchema(t, [1]uint32{}, uint32ArraySchema)
	testBuildArrayOrSliceSchema(t, [1]uint64{}, uint64ArraySchema)
	testBuildArrayOrSliceSchema(t, [1]float32{}, float32ArraySchema)
	testBuildArrayOrSliceSchema(t, [1]float64{}, float64ArraySchema)
	testBuildArrayOrSliceSchema(t, [1]byte{}, uint8ArraySchema)
	testBuildArrayOrSliceSchema(t, [1]rune{}, int32ArraySchema)

	// Should return schema for slices made of each of the basic types
	testBuildArrayOrSliceSchema(t, []string{""}, stringArraySchema)
	testBuildArrayOrSliceSchema(t, []bool{true}, boolArraySchema)
	testBuildArrayOrSliceSchema(t, []int{1}, intArraySchema)
	testBuildArrayOrSliceSchema(t, []int8{1}, int8ArraySchema)
	testBuildArrayOrSliceSchema(t, []int16{1}, int16ArraySchema)
	testBuildArrayOrSliceSchema(t, []int32{1}, int32ArraySchema)
	testBuildArrayOrSliceSchema(t, []int64{1}, int64ArraySchema)
	testBuildArrayOrSliceSchema(t, []uint{1}, uintArraySchema)
	testBuildArrayOrSliceSchema(t, []uint8{1}, uint8ArraySchema)
	testBuildArrayOrSliceSchema(t, []uint16{1}, uint16ArraySchema)
	testBuildArrayOrSliceSchema(t, []uint32{1}, uint32ArraySchema)
	testBuildArrayOrSliceSchema(t, []uint64{1}, uint64ArraySchema)
	testBuildArrayOrSliceSchema(t, []float32{1}, float32ArraySchema)
	testBuildArrayOrSliceSchema(t, []float64{1}, float64ArraySchema)
	testBuildArrayOrSliceSchema(t, []byte{1}, uint8ArraySchema)
	testBuildArrayOrSliceSchema(t, []rune{1}, int32ArraySchema)

	// Should return error when multidimensional array is not one of the basic types
	badMultiArr := [1][1]myContract{}
	schema, err = buildArrayOrSliceSchema(reflect.ValueOf(badMultiArr))

	assert.Equal(t, fmt.Errorf("Slices/Arrays can only have base types %s. Slice/Array has basic type struct", listBasicTypes()), err, "should throw error when 0 length array passed")
	assert.Nil(t, schema, "schema should be nil when sub array bad type")

	// Should return schema for multidimensional arrays made of each of the basic types
	testBuildArrayOrSliceSchema(t, [1][1]string{}, newArraySchema(stringArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]bool{}, newArraySchema(boolArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]int{}, newArraySchema(intArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]int8{}, newArraySchema(int8ArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]int16{}, newArraySchema(int16ArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]int32{}, newArraySchema(int32ArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]int64{}, newArraySchema(int64ArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]uint{}, newArraySchema(uintArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]uint8{}, newArraySchema(uint8ArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]uint16{}, newArraySchema(uint16ArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]uint32{}, newArraySchema(uint32ArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]uint64{}, newArraySchema(uint64ArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]float32{}, newArraySchema(float32ArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]float64{}, newArraySchema(float64ArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]byte{}, newArraySchema(uint8ArraySchema))
	testBuildArrayOrSliceSchema(t, [1][1]rune{}, newArraySchema(int32ArraySchema))

	// Should return schema for multidimensional slices made of each of the basic types
	testBuildArrayOrSliceSchema(t, [][]bool{[]bool{}}, newArraySchema(boolArraySchema))
	testBuildArrayOrSliceSchema(t, [][]int{[]int{}}, newArraySchema(intArraySchema))
	testBuildArrayOrSliceSchema(t, [][]int8{[]int8{}}, newArraySchema(int8ArraySchema))
	testBuildArrayOrSliceSchema(t, [][]int16{[]int16{}}, newArraySchema(int16ArraySchema))
	testBuildArrayOrSliceSchema(t, [][]int32{[]int32{}}, newArraySchema(int32ArraySchema))
	testBuildArrayOrSliceSchema(t, [][]int64{[]int64{}}, newArraySchema(int64ArraySchema))
	testBuildArrayOrSliceSchema(t, [][]uint{[]uint{}}, newArraySchema(uintArraySchema))
	testBuildArrayOrSliceSchema(t, [][]uint8{[]uint8{}}, newArraySchema(uint8ArraySchema))
	testBuildArrayOrSliceSchema(t, [][]uint16{[]uint16{}}, newArraySchema(uint16ArraySchema))
	testBuildArrayOrSliceSchema(t, [][]uint32{[]uint32{}}, newArraySchema(uint32ArraySchema))
	testBuildArrayOrSliceSchema(t, [][]uint64{[]uint64{}}, newArraySchema(uint64ArraySchema))
	testBuildArrayOrSliceSchema(t, [][]float32{[]float32{}}, newArraySchema(float32ArraySchema))
	testBuildArrayOrSliceSchema(t, [][]float64{[]float64{}}, newArraySchema(float64ArraySchema))
	testBuildArrayOrSliceSchema(t, [][]byte{[]byte{}}, newArraySchema(uint8ArraySchema))
	testBuildArrayOrSliceSchema(t, [][]rune{[]rune{}}, newArraySchema(int32ArraySchema))

	// Should handle an array many dimensions
	testBuildArrayOrSliceSchema(t, [1][2][3][4][5][6][7][8]string{}, newArraySchema(newArraySchema(newArraySchema(newArraySchema(newArraySchema(newArraySchema(newArraySchema(stringArraySchema))))))))

	// Should handle a slice of many dimensions
	testBuildArrayOrSliceSchema(t, [][][][][][][][]string{[][][][][][][]string{}}, newArraySchema(newArraySchema(newArraySchema(newArraySchema(newArraySchema(newArraySchema(newArraySchema(stringArraySchema))))))))

	// Should handle an array of slice
	testBuildArrayOrSliceSchema(t, [1][]string{}, newArraySchema(stringArraySchema))

	// Should handle a slice of array
	testBuildArrayOrSliceSchema(t, [][1]string{[1]string{}}, newArraySchema(stringArraySchema))

	// Should return error when multidimensional array/slice/array is bad
	badMixedArr := [1][][0]string{}
	schema, err = buildArrayOrSliceSchema(reflect.ValueOf(badMixedArr))

	assert.EqualError(t, err, "Arrays must have length greater than 0", "should throw error when 0 length array passed")
	assert.Nil(t, schema, "schema should be nil when sub array bad type")
}

func TestGetSchema(t *testing.T) {
	// should return error if passed type not in basic types
	schema, err := getSchema(reflect.TypeOf(complex128(1)))

	assert.Nil(t, schema, "schema should be nil when erroring")
	assert.EqualError(t, err, "complex128 was not a valid basic type", "should have returned correct error for bad type")

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

	// should return value returned by buildArraySchema when type is an array
	stringArray := reflect.TypeOf([0]string{})
	schema, err = getSchema(stringArray)
	basSchema, basErr := buildArraySchema(reflect.New(stringArray).Elem())

	assert.Equal(t, basSchema, schema, "schema should be same as buildArraySchema for array types")
	assert.Equal(t, basErr, err, "error should be same as buildArraySchema for array types")

	// should return value returned by buildArraySchema when type is an slice
	complexSlice := reflect.TypeOf([]complex64{})
	schema, err = getSchema(complexSlice)
	bssSchema, bssErr := buildSliceSchema(reflect.MakeSlice(complexSlice, 1, 1))

	assert.Equal(t, bssSchema, schema, "schema should be same as buildSliceSchema for array types")
	assert.Equal(t, bssErr, err, "error should be same as buildSliceSchema for array types")
}

// ============== transaction_context.go ==============
func TestSetStub(t *testing.T) {
	stub := new(shim.MockStub)
	stub.TxID = "some ID"

	ctx := TransactionContext{}

	ctx.SetStub(stub)

	assert.Equal(t, stub, ctx.stub, "should have set the same stub as passed")
}
func TestGetStub(t *testing.T) {
	stub := new(shim.MockStub)
	stub.TxID = "some ID"

	ctx := TransactionContext{}
	ctx.stub = stub

	assert.Equal(t, stub, ctx.GetStub(), "should have returned same stub as set")
}

// ============== contract_function.go ==============
func TestArrayOfValidType(t *testing.T) {
	var err error

	validParams := make([]string, 0, len(basicTypes))
	for k := range basicTypes {
		validParams = append(validParams, k.String())
	}
	sort.Strings(validParams)

	// Should return an error when array is passed with a length of zero
	zeroArr := [0]int{}
	err = arrayOfValidType(reflect.ValueOf(zeroArr))

	assert.Equal(t, errors.New("Arrays must have length greater than 0"), err, "should throw error when 0 length array passed")

	// Should return error when array is not one of the basic types
	badArr := [1]myContract{}
	err = arrayOfValidType(reflect.ValueOf(badArr))

	assert.Equal(t, fmt.Errorf("Arrays can only have base types %s. Array has basic type struct", listBasicTypes()), err, "should throw error when invalid type passed")

	// Should return nil for arrays made of each of the basic types
	testArrayOfValidTypeIsValid(t, [1]string{})
	testArrayOfValidTypeIsValid(t, [1]bool{})
	testArrayOfValidTypeIsValid(t, [1]int{})
	testArrayOfValidTypeIsValid(t, [1]int8{})
	testArrayOfValidTypeIsValid(t, [1]int16{})
	testArrayOfValidTypeIsValid(t, [1]int32{})
	testArrayOfValidTypeIsValid(t, [1]int64{})
	testArrayOfValidTypeIsValid(t, [1]uint{})
	testArrayOfValidTypeIsValid(t, [1]uint8{})
	testArrayOfValidTypeIsValid(t, [1]uint16{})
	testArrayOfValidTypeIsValid(t, [1]uint32{})
	testArrayOfValidTypeIsValid(t, [1]uint64{})
	testArrayOfValidTypeIsValid(t, [1]float32{})
	testArrayOfValidTypeIsValid(t, [1]float64{})
	testArrayOfValidTypeIsValid(t, [1]byte{})
	testArrayOfValidTypeIsValid(t, [1]rune{})

	// should return error for multidimensional array where length of inner array is 0
	zeroMultiArr := [1][0]int{}
	err = arrayOfValidType(reflect.ValueOf(zeroMultiArr))

	assert.Equal(t, errors.New("Arrays must have length greater than 0"), err, "should throw error when 0 length array passed")

	// Should return error when multidimensional array is not one of the basic types
	badMultiArr := [1][1]myContract{}
	err = arrayOfValidType(reflect.ValueOf(badMultiArr))

	assert.Equal(t, fmt.Errorf("Arrays can only have base types %s. Array has basic type struct", listBasicTypes()), err, "should throw error when 0 length array passed")

	// Should return nil for multidimensional arrays made of each of the basic types
	testArrayOfValidTypeIsValid(t, [1][1]string{})
	testArrayOfValidTypeIsValid(t, [1][1]bool{})
	testArrayOfValidTypeIsValid(t, [1][1]int{})
	testArrayOfValidTypeIsValid(t, [1][1]int8{})
	testArrayOfValidTypeIsValid(t, [1][1]int16{})
	testArrayOfValidTypeIsValid(t, [1][1]int32{})
	testArrayOfValidTypeIsValid(t, [1][1]int64{})
	testArrayOfValidTypeIsValid(t, [1][1]uint{})
	testArrayOfValidTypeIsValid(t, [1][1]uint8{})
	testArrayOfValidTypeIsValid(t, [1][1]uint16{})
	testArrayOfValidTypeIsValid(t, [1][1]uint32{})
	testArrayOfValidTypeIsValid(t, [1][1]uint64{})
	testArrayOfValidTypeIsValid(t, [1][1]float32{})
	testArrayOfValidTypeIsValid(t, [1][1]float64{})
	testArrayOfValidTypeIsValid(t, [1][1]byte{})
	testArrayOfValidTypeIsValid(t, [1][1]rune{})

	// Should handle an array many dimensions
	testArrayOfValidTypeIsValid(t, [1][2][3][4][5][6][7][8]string{})

	// Should handle an array of slices
	testArrayOfValidTypeIsValid(t, [2][]string{})
}

func TestSliceOfValidType(t *testing.T) {
	var err error

	validParams := make([]string, 0, len(basicTypes))
	for k := range basicTypes {
		validParams = append(validParams, k.String())
	}
	sort.Strings(validParams)

	// Should return error when array is not one of the basic types
	badSlice := []myContract{}
	err = sliceOfValidType(reflect.ValueOf(badSlice))

	assert.Equal(t, fmt.Errorf("Slices can only have base types %s. Slice has basic type struct", sliceAsCommaSentence(validParams)), err, "should throw error when invalid type passed")

	// Should return nil for slices made of each of the basic types
	testSliceOfValidTypeIsValid(t, []string{})
	testSliceOfValidTypeIsValid(t, []bool{})
	testSliceOfValidTypeIsValid(t, []int{})
	testSliceOfValidTypeIsValid(t, []int8{})
	testSliceOfValidTypeIsValid(t, []int16{})
	testSliceOfValidTypeIsValid(t, []int32{})
	testSliceOfValidTypeIsValid(t, []int64{})
	testSliceOfValidTypeIsValid(t, []uint{})
	testSliceOfValidTypeIsValid(t, []uint8{})
	testSliceOfValidTypeIsValid(t, []uint16{})
	testSliceOfValidTypeIsValid(t, []uint32{})
	testSliceOfValidTypeIsValid(t, []uint64{})
	testSliceOfValidTypeIsValid(t, []float32{})
	testSliceOfValidTypeIsValid(t, []float64{})
	testSliceOfValidTypeIsValid(t, []byte{})
	testSliceOfValidTypeIsValid(t, []rune{})

	// Should return error when multidimensional slice is not one of the basic types
	badMultiSlice := [][]myContract{}
	err = sliceOfValidType(reflect.ValueOf(badMultiSlice))

	assert.Equal(t, fmt.Errorf("Slices can only have base types %s. Slice has basic type struct", sliceAsCommaSentence(validParams)), err, "should throw error when 0 length array passed")

	// Should return nil for multidimensional slices made of each of the basic types
	testSliceOfValidTypeIsValid(t, [][]string{})
	testSliceOfValidTypeIsValid(t, [][]bool{})
	testSliceOfValidTypeIsValid(t, [][]int{})
	testSliceOfValidTypeIsValid(t, [][]int8{})
	testSliceOfValidTypeIsValid(t, [][]int32{})
	testSliceOfValidTypeIsValid(t, [][]int64{})
	testSliceOfValidTypeIsValid(t, [][]uint{})
	testSliceOfValidTypeIsValid(t, [][]uint8{})
	testSliceOfValidTypeIsValid(t, [][]uint16{})
	testSliceOfValidTypeIsValid(t, [][]uint32{})
	testSliceOfValidTypeIsValid(t, [][]uint64{})
	testSliceOfValidTypeIsValid(t, [][]float32{})
	testSliceOfValidTypeIsValid(t, [][]float64{})
	testSliceOfValidTypeIsValid(t, [][]byte{})
	testSliceOfValidTypeIsValid(t, [][]rune{})

	// Should handle a slice many dimensions
	testSliceOfValidTypeIsValid(t, [][][][][][][][]string{})

	// Should handle a slice of arrays
	testSliceOfValidTypeIsValid(t, [][2]string{})
}

func TestTypeIsValid(t *testing.T) {
	badArr := reflect.New(badArrayType).Elem()
	badSlice := reflect.MakeSlice(badSliceType, 1, 1)

	// Should return error is non-array/slice type is invalid
	assert.EqualError(t, typeIsValid(badType, []reflect.Type{}), fmt.Sprintf("Type %s is not valid. Expected one of the basic types %s or an array/slice of these", badType.String(), listBasicTypes()), "should have returned error for invalid type")

	// Should return error returned by array of valid type for invalid array
	assert.EqualError(t, typeIsValid(badArrayType, []reflect.Type{}), arrayOfValidType(badArr).Error(), "should have returned error for invalid array type")

	// Should return error returned by array of valid type for invalid array
	assert.EqualError(t, typeIsValid(badSliceType, []reflect.Type{}), sliceOfValidType(badSlice).Error(), "should have returned error for invalid slice type")

	// Should accept valid basic types
	assert.Nil(t, typeIsValid(boolRefType, []reflect.Type{}), "should not return an error for a bool type")
	assert.Nil(t, typeIsValid(stringRefType, []reflect.Type{}), "should not return an error for a string type")
	assert.Nil(t, typeIsValid(intRefType, []reflect.Type{}), "should not return an error for int type")
	assert.Nil(t, typeIsValid(int8RefType, []reflect.Type{}), "should not return an error for int8 type")
	assert.Nil(t, typeIsValid(int16RefType, []reflect.Type{}), "should not return an error for int16 type")
	assert.Nil(t, typeIsValid(int32RefType, []reflect.Type{}), "should not return an error for int32 type")
	assert.Nil(t, typeIsValid(int64RefType, []reflect.Type{}), "should not return an error for int64 type")
	assert.Nil(t, typeIsValid(uintRefType, []reflect.Type{}), "should not return an error for uint type")
	assert.Nil(t, typeIsValid(uint8RefType, []reflect.Type{}), "should not return an error for uint8 type")
	assert.Nil(t, typeIsValid(uint16RefType, []reflect.Type{}), "should not return an error for uint16 type")
	assert.Nil(t, typeIsValid(uint32RefType, []reflect.Type{}), "should not return an error for uint32 type")
	assert.Nil(t, typeIsValid(uint64RefType, []reflect.Type{}), "should not return an error for uint64 type")
	assert.Nil(t, typeIsValid(float32RefType, []reflect.Type{}), "should not return an error for float32 type")
	assert.Nil(t, typeIsValid(float64RefType, []reflect.Type{}), "should not return an error for float64 type")

	// Should accept valid array
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]string{}), []reflect.Type{}), "should not return an error for a string array type")

	// Should accept valid slice
	assert.Nil(t, typeIsValid(reflect.TypeOf([]string{}), []reflect.Type{}), "should not return an error for a string slice type")

	// Should accept value if not in basic types but in additional values
	assert.Nil(t, typeIsValid(badType, []reflect.Type{badType}), "should not error when type not in basic types but is in additional types")

	// Should not handle arrays as additional types
	assert.EqualError(t, typeIsValid(badArrayType, []reflect.Type{badArrayType}), arrayOfValidType(badArr).Error(), "should have returned error for invalid array type")

	// Should not handle slices as additional types
	assert.EqualError(t, typeIsValid(badSliceType, []reflect.Type{badSliceType}), sliceOfValidType(badSlice).Error(), "should have returned error for invalid slice type")
}

func TestMethod2ContractFunctionParams(t *testing.T) {
	testMethod2ContractFunctionParams(t, false)
	testMethod2ContractFunctionParams(t, true)
}

func TestMethod2ContractFunctionReturns(t *testing.T) {
	testMethod2ContractFunctionReturns(t, false)
	testMethod2ContractFunctionReturns(t, true)
}

func TestParseMethod(t *testing.T) {
	testParseMethod(t, false)
	testParseMethod(t, true)
}

func TestNewContractFunction(t *testing.T) {
	mc := new(myContract)
	fnValue := reflect.ValueOf(mc.ReturnsString)

	cfParams := contractFunctionParams{}
	cfParams.context = basicContextPtrType
	cfParams.fields = []reflect.Type{}

	cfReturns := contractFunctionReturns{}
	cfReturns.success = stringRefType
	cfReturns.error = true

	cf := newContractFunction(fnValue, cfParams, cfReturns)

	expectedResp := mc.ReturnsString()
	actualResp := cf.function.Call([]reflect.Value{})[0].Interface().(string)

	assert.Equal(t, expectedResp, actualResp, "should set reflect value of function as function")
	assert.Equal(t, cfParams, cf.params, "should have correct params")
	assert.Equal(t, cfReturns, cf.returns, "should have correct params")
}

func TestNewContractFunctionFromFunc(t *testing.T) {
	// Should panic when interface passed is not of type func
	assert.PanicsWithValue(t, "Cannot create new contract function from string. Can only use func", func() { newContractFunctionFromFunc("some string", basicContextPtrType) }, "should only allow funcs to be passed in")

	var bc *badContract
	// Should panic when function provided has invalid input params
	bc = new(badContract)
	assert.PanicsWithValue(t, fmt.Sprintf("Function contains invalid parameter type. %s", typeIsValid(badType, []reflect.Type{basicContextPtrType})), func() { newContractFunctionFromFunc(bc.TakesBadType, basicContextPtrType) }, "should panic if input params do not match what param parser wants")

	// Should panic when function provided has invalid return types
	bc = new(badContract)
	assert.PanicsWithValue(t, fmt.Sprintf("Function contains invalid single return type. %s", typeIsValid(badType, []reflect.Type{errorType})), func() { newContractFunctionFromFunc(bc.ReturnsBadType, basicContextPtrType) }, "should panic if returns types do not match what return parser wants")

	// Should create contractFunction for valid input
	mc := new(myContract)
	cf := newContractFunctionFromFunc(mc.ReturnsString, basicContextPtrType)
	testContractFunctionUsingReturnsString(t, mc, cf)
}

func TestNewContractFunctionFromReflect(t *testing.T) {
	bc := new(badContract)
	var typeMethod reflect.Method
	var valueMethod reflect.Value
	// Should panic when function provided has invalid input params
	typeMethod, valueMethod = generateMethodTypesAndValuesFromName(bc, "TakesBadType")
	assert.PanicsWithValue(t, fmt.Sprintf("TakesBadType contains invalid parameter type. %s", typeIsValid(badType, []reflect.Type{basicContextPtrType})), func() {
		newContractFunctionFromReflect(typeMethod, valueMethod, basicContextPtrType)
	}, "should panic if input params do not match what param parser wants")

	// Should panic when function provided has invalid return types
	typeMethod, valueMethod = generateMethodTypesAndValuesFromName(bc, "ReturnsBadType")
	assert.PanicsWithValue(t, fmt.Sprintf("ReturnsBadType contains invalid single return type. %s", typeIsValid(badType, []reflect.Type{errorType})), func() {
		newContractFunctionFromReflect(typeMethod, valueMethod, basicContextPtrType)
	}, "should panic if returns types do not match what return parser wants")

	// Should create contractFunction for valid input
	mc := new(myContract)
	typeMethod, valueMethod = generateMethodTypesAndValuesFromName(mc, "ReturnsString")
	cf := newContractFunctionFromReflect(typeMethod, valueMethod, basicContextPtrType)
	testContractFunctionUsingReturnsString(t, mc, cf)
}

func TestCreateArrayOrSlice(t *testing.T) {
	var val reflect.Value
	var err error

	arrType := reflect.TypeOf([2]string{})
	multiDArrType := reflect.TypeOf([2][1]string{})
	sliceType := reflect.TypeOf([]string{})
	multiDSliceType := reflect.TypeOf([][]string{})
	arrOfSliceType := reflect.TypeOf([2][]string{})
	sliceOfArrType := reflect.TypeOf([][2]string{})

	// should error when passed data is not json
	testCreateArrayOrSliceErrors(t, "bad JSON", arrType)

	// should error when passed data is json but not valid for the unmarshalling
	testCreateArrayOrSliceErrors(t, "{\"some\": \"object\"}", arrType)

	// Should error when array passed but it is too deep
	testCreateArrayOrSliceErrors(t, "[[\"a\"],[\"b\"]]", arrType)

	// Should error when array passed but it is too shallow
	testCreateArrayOrSliceErrors(t, "[\"a\",\"b\"]", multiDArrType)

	// Should error when slice passed but it is too deep
	testCreateArrayOrSliceErrors(t, "[[\"a\"],[\"b\"]]", sliceType)

	// Should error when slice passed but it is too deep
	testCreateArrayOrSliceErrors(t, "[\"a\",\"b\"]", multiDSliceType)

	// Should return error when array passed but contains data of the wrong type
	testCreateArrayOrSliceErrors(t, "[\"a\", 1]", arrType)

	// Should return error when slice passed but contains data of the wrong type
	testCreateArrayOrSliceErrors(t, "[\"a\", 1]", sliceType)

	// Should return error when type wrong for array of slice
	testCreateArrayOrSliceErrors(t, "[[\"a\"],[1]]", arrOfSliceType)

	// Should return error when type wrong for array of slice
	testCreateArrayOrSliceErrors(t, "[[\"a\", 1]]", sliceOfArrType)

	// Should return reflect value for array
	val, err = createArrayOrSlice("[\"a\",\"b\"]", arrType)

	assert.Nil(t, err, "should have nil error for valid array passed")
	assert.Equal(t, [2]string{"a", "b"}, val.Interface().([2]string), "should have returned value of array with filled in data")

	// Should return reflect value for md array
	val, err = createArrayOrSlice("[[\"a\"],[\"b\"]]", multiDArrType)

	assert.Nil(t, err, "should have nil error for valid array passed")
	assert.Equal(t, [2][1]string{{"a"}, {"b"}}, val.Interface().([2][1]string), "should have returned value of multi dimensional array with filled in data")

	// Should return reflect value for slice
	val, err = createArrayOrSlice("[\"a\",\"b\"]", sliceType)

	assert.Nil(t, err, "should have nil error for valid slice passed")
	assert.Equal(t, []string{"a", "b"}, val.Interface().([]string), "should have returned value of slice with filled in data")

	// Should return reflect value for md slice
	val, err = createArrayOrSlice("[[\"a\"],[\"b\"]]", multiDSliceType)

	assert.Nil(t, err, "should have nil error for valid slice passed")
	assert.Equal(t, [][]string{{"a"}, {"b"}}, val.Interface().([][]string), "should have returned value of multi dimensional slice with filled in data")

	// Should return reflect value for an array of slices
	val, err = createArrayOrSlice("[[\"a\"],[\"b\"]]", arrOfSliceType)

	assert.Nil(t, err, "should have nil error for valid slice passed")
	assert.Equal(t, [2][]string{{"a"}, {"b"}}, val.Interface().([2][]string), "should have returned value of array of slices with filled in data")

	// Should return reflect value for a slice of arrays
	val, err = createArrayOrSlice("[[\"a\", \"b\"]]", sliceOfArrType)

	assert.Nil(t, err, "should have nil error for valid slice passed")
	assert.Equal(t, [][2]string{{"a", "b"}}, val.Interface().([][2]string), "should have returned value of slice of arrays with filled in data")
}

func TestGetArgs(t *testing.T) {
	var values []reflect.Value
	var ok bool
	testParams := []string{"one", "two", "three"}

	ctx := new(TransactionContext)
	cf := contractFunction{}

	// Should return empty array when contract function takes no params
	setContractFunctionParams(&cf, nil, []reflect.Type{})

	callGetArgsAndBasicTest(t, cf, ctx, testParams)

	// Should return array using passed parameters when contract function takes same number of params as sent
	setContractFunctionParams(&cf, nil, []reflect.Type{
		stringRefType,
		stringRefType,
		stringRefType,
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, testParams)

	testReflectValueEqualSlice(t, values, testParams)

	// Should return array with first n passed params when contract function takes n params with n less than length of passed params
	setContractFunctionParams(&cf, nil, []reflect.Type{
		stringRefType,
		stringRefType,
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, testParams)

	testReflectValueEqualSlice(t, values, testParams)

	// Should return array with all passed params and bulked out when contract function takes n params with n greater than length of passed params for string
	setContractFunctionParams(&cf, nil, []reflect.Type{
		stringRefType,
		stringRefType,
		stringRefType,
		stringRefType,
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, testParams)

	testReflectValueEqualSlice(t, values, append(testParams, ""))

	// Should return array with all passed params and bulked out when contract function takes n params with n greater than length of passed params for array
	setContractFunctionParams(&cf, nil, []reflect.Type{
		stringRefType,
		stringRefType,
		stringRefType,
		reflect.TypeOf([3]int{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, testParams)

	testReflectValueEqualSlice(t, values, append(testParams, "[0 0 0]")) // <- array formatted as sprintf turns to string

	// Should return array with all passed params and bulked out when contract function takes n params with n greater than length of passed params for slice
	setContractFunctionParams(&cf, nil, []reflect.Type{
		stringRefType,
		stringRefType,
		stringRefType,
		reflect.TypeOf([]string{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, testParams)

	testReflectValueEqualSlice(t, values, append(testParams, "[]"))

	// Should include ctx in returned values and no params when function only takes ctx
	setContractFunctionParams(&cf, basicContextPtrType, []reflect.Type{})

	values = callGetArgsAndBasicTest(t, cf, ctx, testParams)

	_, ok = values[0].Interface().(*TransactionContext)

	assert.True(t, ok, "first parameter should be *TransactionContext when takesContext")

	// Should include ctx in returned values and params when function takes in params and ctx
	setContractFunctionParams(&cf, basicContextPtrType, []reflect.Type{
		stringRefType,
		stringRefType,
		stringRefType,
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, testParams)

	_, ok = values[0].Interface().(*TransactionContext)

	assert.True(t, ok, "first parameter should be *TransactionContext when takesContext")

	testReflectValueEqualSlice(t, values[1:], testParams)

	// Should be using context passed
	setContractFunctionParams(&cf, reflect.TypeOf(new(customContext)), []reflect.Type{})

	values, err := getArgs(cf, reflect.ValueOf(new(customContext)), testParams)

	assert.Nil(t, err, "should not return an error for a valid cf")
	assert.Equal(t, 1, len(values), "should return same length array list as number of fields plus 1 for context")

	_, ok = values[0].Interface().(*customContext)

	assert.True(t, ok, "first parameter should be *TransactionContext when takesContext")

	testReflectValueEqualSlice(t, values[1:], testParams)

	// Should handle bool
	setContractFunctionParams(&cf, nil, []reflect.Type{
		boolRefType,
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, []string{"true"})
	testReflectValueEqualSlice(t, values, []bool{true})

	// Should handle ints
	intTypes := map[reflect.Kind]interface{}{
		reflect.Int:   []int{1},
		reflect.Int8:  []int8{1},
		reflect.Int16: []int16{1},
		reflect.Int32: []int32{1},
		reflect.Int64: []int64{1},
	}

	testGetArgsWithTypes(t, intTypes, []string{"1"})

	// Should handle uints
	uintTypes := map[reflect.Kind]interface{}{
		reflect.Uint:   []uint{1},
		reflect.Uint8:  []uint8{1},
		reflect.Uint16: []uint16{1},
		reflect.Uint32: []uint32{1},
		reflect.Uint64: []uint64{1},
	}

	testGetArgsWithTypes(t, uintTypes, []string{"1"})

	// Should handle floats
	floatTypes := map[reflect.Kind]interface{}{
		reflect.Float32: []float32{1.1},
		reflect.Float64: []float64{1.1},
	}

	testGetArgsWithTypes(t, floatTypes, []string{"1.1"})

	// Should handle bytes
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf(byte(65)),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, []string{"65"})
	testReflectValueEqualSlice(t, values, []byte{65})

	// Should handle runes
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf(rune(65)),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, []string{"65"})
	testReflectValueEqualSlice(t, values, []rune{65})

	// Should return an error if conversion errors
	setContractFunctionParams(&cf, nil, []reflect.Type{
		intRefType,
	})

	values, err = getArgs(cf, reflect.ValueOf(ctx), []string{"abc"})

	assert.EqualError(t, err, "Param abc could not be converted to type int", "should have returned error when convert returns error")
	assert.Nil(t, values, "should not have returned value list on error")

	// Should handle array of basic type
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([4]int{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, []string{"[1,2,3,4]"})
	testReflectValueEqualSlice(t, values, [][4]int{{1, 2, 3, 4}})

	// Should handle multidimensional array of basic type
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([4][1]int{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, []string{"[[1],[2],[3],[4]]"})
	testReflectValueEqualSlice(t, values, [][4][1]int{{{1}, {2}, {3}, {4}}})

	// Should error when the array they pass is not the correct format
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([4]int{}),
	})

	values, err = getArgs(cf, reflect.ValueOf(ctx), []string{"[1,2,3,\"a\"]"})
	assert.EqualError(t, err, "Value [1,2,3,\"a\"] was not passed in expected format [4]int", "should have returned error when array conversion returns error")
	assert.Nil(t, values, "should not have returned value list on error")

	// Should error when the element in multidimensional array they pass is not the correct format
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([4][1]int{}),
	})

	values, err = getArgs(cf, reflect.ValueOf(ctx), []string{"[[1],[2],[3],[\"a\"]]"})
	assert.EqualError(t, err, "Value [[1],[2],[3],[\"a\"]] was not passed in expected format [4][1]int", "should have returned error when array conversion returns error")
	assert.Nil(t, values, "should not have returned value list on error")

	// Should handle an array of slices of a basic type
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([4][]int{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, []string{"[[1, 2],[3],[4],[5]]"})
	testReflectValueEqualSlice(t, values, [][4][]int{{{1, 2}, {3}, {4}, {5}}})

	// Should handle a slice of arrays of a basic type
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([][4]int{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, []string{"[[1,2,3,4]]"})
	testReflectValueEqualSlice(t, values, [][][4]int{{{1, 2, 3, 4}}})
}

func testHandleResponse(t *testing.T, successReturn reflect.Type, errorReturn bool, response []reflect.Value, expectedString string, expectedError error) {
	t.Helper()

	cf := contractFunction{}

	setContractFunctionReturns(&cf, successReturn, errorReturn)
	strResp, errResp := handleContractFunctionResponse(response, cf)

	assert.Equal(t, expectedString, strResp, "should have returned string value from response")
	assert.Equal(t, expectedError, errResp, "should have returned error value from response")
}

func TestHandleContractFunctionResponse(t *testing.T) {
	cf := contractFunction{}

	stringMsg := "some string"
	stringValue := reflect.ValueOf(stringMsg)

	nilErrorValue := reflect.ValueOf(nil)
	err := errors.New("Hello Error")
	errorValue := reflect.ValueOf(err)

	var response []reflect.Value

	// Should panic if response to handle is longer than the contractFunctions expected return
	setContractFunctionReturns(&cf, nil, false)
	assert.PanicsWithValue(t, "Response does not match expected return for given function.", func() { handleContractFunctionResponse([]reflect.Value{stringValue, errorValue}, cf) }, "should have panicked as response did not match the contractFunctions expected response format")

	setContractFunctionReturns(&cf, stringRefType, false)
	assert.PanicsWithValue(t, "Response does not match expected return for given function.", func() { handleContractFunctionResponse([]reflect.Value{stringValue, errorValue}, cf) }, "should have panicked as response did not match the contractFunctions expected response format")

	setContractFunctionReturns(&cf, nil, true)
	assert.PanicsWithValue(t, "Response does not match expected return for given function.", func() { handleContractFunctionResponse([]reflect.Value{stringValue, errorValue}, cf) }, "should have panicked as response did not match the contractFunctions expected response format")

	setContractFunctionReturns(&cf, stringRefType, true)
	assert.PanicsWithValue(t, "Response does not match expected return for given function.", func() { handleContractFunctionResponse([]reflect.Value{stringValue}, cf) }, "should have panicked as response did not match the contractFunctions expected response format")

	setContractFunctionReturns(&cf, stringRefType, true)
	assert.PanicsWithValue(t, "Response does not match expected return for given function.", func() { handleContractFunctionResponse([]reflect.Value{errorValue}, cf) }, "should have panicked as response did not match the contractFunctions expected response format")

	setContractFunctionReturns(&cf, stringRefType, true)
	assert.PanicsWithValue(t, "Response does not match expected return for given function.", func() { handleContractFunctionResponse([]reflect.Value{stringValue, stringValue, errorValue}, cf) }, "should have panicked as response did not match the contractFunctions expected response format")

	setContractFunctionReturns(&cf, stringRefType, true)
	assert.PanicsWithValue(t, "Response does not match expected return for given function.", func() { handleContractFunctionResponse([]reflect.Value{}, cf) }, "should have panicked as response did not match the contractFunctions expected response format")

	// Should return string and nil error values when response contains string and nil error and expecting both
	response = []reflect.Value{stringValue, nilErrorValue}
	testHandleResponse(t, stringRefType, true, response, stringMsg, nil)

	// Should return response string and nil for error when one value returned and expecting only string
	response = []reflect.Value{stringValue}
	testHandleResponse(t, stringRefType, false, response, stringMsg, nil)

	// Should return blank string and response error when one value returned and expecting only error
	response = []reflect.Value{errorValue}
	testHandleResponse(t, nil, true, response, "", err)

	// Should return blank string and nil error when response is empty array and expecting no string or error
	response = []reflect.Value{}
	testHandleResponse(t, nil, false, response, "", nil)

	// Should return basic types in string form
	response = []reflect.Value{reflect.ValueOf(1)}
	testHandleResponse(t, intRefType, false, response, "1", nil)

	response = []reflect.Value{reflect.ValueOf(int8(1))}
	testHandleResponse(t, int8RefType, false, response, "1", nil)

	response = []reflect.Value{reflect.ValueOf(int16(1))}
	testHandleResponse(t, int16RefType, false, response, "1", nil)

	response = []reflect.Value{reflect.ValueOf(int32(1))}
	testHandleResponse(t, int32RefType, false, response, "1", nil)

	response = []reflect.Value{reflect.ValueOf(int64(1))}
	testHandleResponse(t, int64RefType, false, response, "1", nil)

	response = []reflect.Value{reflect.ValueOf(uint(1))}
	testHandleResponse(t, uintRefType, false, response, "1", nil)

	response = []reflect.Value{reflect.ValueOf(uint8(1))}
	testHandleResponse(t, uint8RefType, false, response, "1", nil)

	response = []reflect.Value{reflect.ValueOf(uint16(1))}
	testHandleResponse(t, uint16RefType, false, response, "1", nil)

	response = []reflect.Value{reflect.ValueOf(uint32(1))}
	testHandleResponse(t, uint32RefType, false, response, "1", nil)

	response = []reflect.Value{reflect.ValueOf(uint64(1))}
	testHandleResponse(t, uint64RefType, false, response, "1", nil)

	response = []reflect.Value{reflect.ValueOf(float32(1.1))}
	testHandleResponse(t, float32RefType, false, response, "1.1", nil)

	response = []reflect.Value{reflect.ValueOf(float64(1.1))}
	testHandleResponse(t, float64RefType, false, response, "1.1", nil)

	// Should return array responses as JSON strings
	intArray := [4]int{1, 2, 3, 4}
	response = []reflect.Value{reflect.ValueOf(intArray)}
	testHandleResponse(t, reflect.TypeOf(intArray), false, response, "[1,2,3,4]", nil)

	intMdArray := [4][]int{{1}, {2, 3}, {4, 5, 6}, {7}}
	response = []reflect.Value{reflect.ValueOf(intMdArray)}
	testHandleResponse(t, reflect.TypeOf(intMdArray), false, response, "[[1],[2,3],[4,5,6],[7]]", nil)

	// Should return slice responses as JSON strings
	intSlice := []int{1, 2, 3, 4}
	response = []reflect.Value{reflect.ValueOf(intSlice)}
	testHandleResponse(t, reflect.TypeOf(intSlice), false, response, "[1,2,3,4]", nil)

	intMdSlice := [][]int{{1}, {2, 3}, {4, 5, 6}, {7}}
	response = []reflect.Value{reflect.ValueOf(intMdSlice)}
	testHandleResponse(t, reflect.TypeOf(intMdSlice), false, response, "[[1],[2,3],[4,5,6],[7]]", nil)
}

func TestCall(t *testing.T) {
	var expectedStr string
	var expectedErr error
	var actualStr string
	var actualErr error

	cf := new(contractFunction)
	ctx := new(TransactionContext)
	mc := myContract{}

	// Should call function of contract function with correct params and return expected values for context and param function
	cf = newContractFunctionFromFunc(mc.UsesContext, basicContextPtrType)

	expectedStr, expectedErr = mc.UsesContext(ctx, standardAssetID, standardValue)
	actualStr, actualErr = cf.call(reflect.ValueOf(ctx), standardAssetID, standardValue)

	assert.Equal(t, expectedStr, actualStr, "Should have returned string as a regular call to UsesContext would")
	assert.Equal(t, expectedErr, actualErr, "Should have returned error as a regular call to UsesContext would")

	// Should call function of contract function with correct params and return expected values for function returning nothing
	cf = newContractFunctionFromFunc(mc.ReturnsNothing, basicContextPtrType)

	actualStr, actualErr = cf.call(reflect.ValueOf(ctx))

	assert.Equal(t, "", actualStr, "Should have returned blank string")
	assert.Nil(t, actualErr, "Should have returned nil")

	// Should call function of contract function with correct params and return expected values for function returning string
	cf = newContractFunctionFromFunc(mc.ReturnsString, basicContextPtrType)

	expectedStr = mc.ReturnsString()

	actualStr, actualErr = cf.call(reflect.ValueOf(ctx))

	assert.Equal(t, expectedStr, actualStr, "Should have returned string as regular call to ReturnsString would")
	assert.Nil(t, actualErr, "Should have returned nil")

	// Should call function of contract function with correct params and return expected values for function returning string
	cf = newContractFunctionFromFunc(mc.UsesBasics, basicContextPtrType)

	expectedStr = mc.UsesBasics("some string", true, 123, 45, 6789, 101112, 131415, 123, 45, 6789, 101112, 131415, 1.1, 2.2, 65, 66)

	actualStr, actualErr = cf.call(reflect.ValueOf(ctx), "some string", "true", "123", "45", "6789", "101112", "131415", "123", "45", "6789", "101112", "131415", "1.1", "2.2", "65", "66")

	assert.Equal(t, expectedStr, actualStr, "Should have returned string as regular call to ReturnsString would")
	assert.Nil(t, actualErr, "Should have returned nil")

	// Should call function of contract function with correct params and return expected values for function returning error
	cf = newContractFunctionFromFunc(mc.ReturnsError, basicContextPtrType)

	expectedErr = mc.ReturnsError()

	actualStr, actualErr = cf.call(reflect.ValueOf(ctx))

	assert.Equal(t, "", actualStr, "Should have returned blank string")
	assert.EqualError(t, actualErr, expectedErr.Error(), "Should have returned error as a regular call to ReturnsError would")

	// Should return error when getArgs returns an error
	cf = newContractFunctionFromFunc(mc.UsesArray, basicContextPtrType)

	expectedErr = errors.New("Value [1] was not passed in expected format [1]string")

	actualStr, actualErr = cf.call(reflect.ValueOf(ctx), "[1]")

	assert.Equal(t, "", actualStr, "Should have returned blank string")
	assert.EqualError(t, actualErr, expectedErr.Error(), "Should have returned error from getArgs would")
}

func TestExists(t *testing.T) {
	mc := myContract{}
	cf := contractFunction{}

	assert.False(t, cf.exists(), "should return false when contractFunction function is not set")

	cf.function = reflect.ValueOf(nil)
	assert.False(t, cf.exists(), "should return false when contractFunction function is set and nil")

	cf.function = reflect.ValueOf(mc.ReturnsString)
	assert.True(t, cf.exists(), "should return true when contractFunction function is set and not nil")
}

// ============== contract_def.go ==============
func TestSetUnknownTransaction(t *testing.T) {
	mc := myContract{}

	// Should set unknown transaction
	mc.SetUnknownTransaction(mc.ReturnsString)
	assert.Equal(t, mc.ReturnsString(), mc.unknownTransaction.(func() string)(), "unknown transaction should have been set to value passed")
}

func TestGetUnknownTransaction(t *testing.T) {
	var mc myContract
	var unknownFn interface{}
	var err error
	// Should throw an error when unknown transaction not set
	mc = myContract{}

	unknownFn, err = mc.GetUnknownTransaction()

	assert.EqualError(t, err, "unknown transaction not set", "should return an error when unknown transaction not set")
	assert.Nil(t, unknownFn, "should not return contractFunction when unknown transaction not set")

	// Should return the call value of the stored unknown transaction when set
	mc = myContract{}
	mc.unknownTransaction = mc.ReturnsInt

	unknownFn, err = mc.GetUnknownTransaction()

	assert.Nil(t, err, "should not return error when unknown function set")
	assert.Equal(t, mc.ReturnsInt(), unknownFn.(func() int)(), "function returned should be same value as set for unknown transaction")
}

func TestSetBeforeTransaction(t *testing.T) {
	mc := myContract{}

	// Should set before transaction
	mc.SetBeforeTransaction(mc.ReturnsString)
	assert.Equal(t, mc.ReturnsString(), mc.beforeTransaction.(func() string)(), "before transaction should have been set to value passed")
}

func TestGetBeforeTransaction(t *testing.T) {
	var mc myContract
	var beforeFn interface{}
	var err error
	// Should throw an error when before transaction not set
	mc = myContract{}

	beforeFn, err = mc.GetBeforeTransaction()

	assert.EqualError(t, err, "before transaction not set", "should return an error when before transaction not set")
	assert.Nil(t, beforeFn, "should not return contractFunction when before transaction not set")

	// Should return the call value of the stored before transaction when set
	mc = myContract{}
	mc.beforeTransaction = mc.ReturnsInt

	beforeFn, err = mc.GetBeforeTransaction()

	assert.Nil(t, err, "should not return error when before transaction set")
	assert.Equal(t, mc.ReturnsInt(), beforeFn.(func() int)(), "function returned should be same value as set for before transaction")
}

func TestSetAfterTransaction(t *testing.T) {
	mc := myContract{}

	// Should set after transaction
	mc.SetAfterTransaction(mc.ReturnsString)
	assert.Equal(t, mc.ReturnsString(), mc.afterTransaction.(func() string)(), "after transaction should have been set to value passed")
}

func TestGetAfterTransaction(t *testing.T) {
	var mc myContract
	var afterFn interface{}
	var err error
	// Should throw an error when after transaction not set
	mc = myContract{}

	afterFn, err = mc.GetAfterTransaction()

	assert.EqualError(t, err, "after transaction not set", "should return an error when after transaction not set")
	assert.Nil(t, afterFn, "should not return contractFunction when after transaction not set")

	// Should return the call value of the stored after transaction when set
	mc = myContract{}
	mc.afterTransaction = mc.ReturnsInt

	afterFn, err = mc.GetAfterTransaction()

	assert.Nil(t, err, "should not return error when after transaction set")
	assert.Equal(t, mc.ReturnsInt(), afterFn.(func() int)(), "function returned should be same value as set for after transaction")
}

func TestSetName(t *testing.T) {
	mc := myContract{}

	mc.SetName("myname")

	assert.NotNil(t, mc.name, "should have set name")
	assert.Equal(t, "myname", mc.name, "name set incorrectly")
}

func TestGetName(t *testing.T) {
	mc := myContract{}

	assert.Equal(t, "", mc.GetName(), "should have returned blank ns when not set")

	mc.name = "myname"
	assert.Equal(t, "myname", mc.GetName(), "should have returned custom ns when set")
}

func TestSetTransactionContextHandler(t *testing.T) {
	sc := simpleTestContractWithCustomContext{}
	ctx := new(customContext)

	// should set the context handler value
	sc.SetTransactionContextHandler(ctx)
	assert.Equal(t, sc.contextHandler, ctx, "should set contextHandler")
	sc = simpleTestContractWithCustomContext{}
}

func TestGetTransactionContextHandler(t *testing.T) {
	sc := simpleTestContractWithCustomContext{}

	// Should return default transaction context type
	assert.Equal(t, new(TransactionContext), sc.GetTransactionContextHandler(), "should return default transaction context type when unset")

	// Should return set transaction context type
	sc.contextHandler = new(customContext)
	assert.Equal(t, new(customContext), sc.GetTransactionContextHandler(), "should return custom context when set")
}

// ============== system_contract.go ==============
func TestSetMetadata(t *testing.T) {
	sc := systemContract{}
	sc.setMetadata("my metadata")

	assert.Equal(t, "my metadata", sc.metadata, "should have set metadata field")
}

func TestGetMetadata(t *testing.T) {
	sc := systemContract{}
	sc.metadata = "my metadata"

	assert.Equal(t, "my metadata", sc.GetMetadata(), "should have returned metadata field")
}

// ============== metadata.go ==============
func TestGetJSONSchema(t *testing.T) {
	schemaLoader := gojsonschema.NewBytesLoader([]byte(GetJSONSchema()))
	_, err := gojsonschema.NewSchema(schemaLoader)

	assert.Nil(t, err, "value returned by GetJSONSchema should be a valid JSON schema")
}

func TestSchemaOrBooleanMarshalJSON(t *testing.T) {
	var sob SchemaOrBoolean
	var marshalBytes []byte
	var marshalErr error

	var expectedBytes []byte
	var expectedErr error

	// Should return bytes for Boolean when Schema nil
	sob = SchemaOrBoolean{}
	sob.Boolean = true

	marshalBytes, marshalErr = sob.MarshalJSON()
	expectedBytes, expectedErr = json.Marshal(sob.Boolean)

	assert.Equal(t, expectedBytes, marshalBytes, "should return json marshal for boolean")
	assert.Equal(t, expectedErr, marshalErr, "should return json marshal for boolean")

	// Should return bytes for Schema when Schema not nil
	schema := new(Schema)
	schema.Type = []string{"string"}

	sob = SchemaOrBoolean{}
	sob.Schema = schema

	marshalBytes, marshalErr = sob.MarshalJSON()
	expectedBytes, expectedErr = json.Marshal(sob.Schema)

	assert.Equal(t, expectedBytes, marshalBytes, "should return json marshal for schema")
	assert.Equal(t, expectedErr, marshalErr, "should return json marshal for schema")
}

func TestSchemaOrBooleanUnmarshalJSON(t *testing.T) {
	var sob *SchemaOrBoolean
	var err error

	// Should set Schema when schema json sent

	expectedSchema := new(Schema)
	expectedSchema.Type = []string{"object"}
	expectedSchema.Format = "some format"

	testJSON, _ := json.Marshal(expectedSchema)

	sob = new(SchemaOrBoolean)
	err = sob.UnmarshalJSON(testJSON)

	assert.Nil(t, err, "should not error for valid schema json")
	assert.Equal(t, sob.Schema, expectedSchema, "should set schema to schema")

	// Should set boolean when boolean json sent
	sob = new(SchemaOrBoolean)
	err = sob.UnmarshalJSON([]byte("true"))

	assert.Nil(t, err, "should not return error for valid boolean true value")
	assert.True(t, sob.Boolean, "should have set boolean value when bytes sent json for true bool")

	sob = new(SchemaOrBoolean)
	err = sob.UnmarshalJSON([]byte("false"))

	assert.Nil(t, err, "should not return error for valid boolean false value")
	assert.False(t, sob.Boolean, "should have set boolean value when bytes sent json for false bool")

	// Should return error when bytes neither boolean or schema
	sob = new(SchemaOrBoolean)
	err = sob.UnmarshalJSON([]byte("123"))

	assert.EqualError(t, err, "Can only unmarshal to SchemaOrBoolean if value is boolean or Schema format")

	// Should return error when invalid JSON
	sob = new(SchemaOrBoolean)

	expectedErr := json.Unmarshal([]byte("bad json"), "useless value")
	err = sob.UnmarshalJSON([]byte("bad json"))

	assert.Equal(t, expectedErr, err, "should have an error value when bad JSON")
}

func TestSchemaOrArrayMarshalJSON(t *testing.T) {
	var soa SchemaOrArray
	var marshalBytes []byte
	var marshalErr error

	var expectedBytes []byte
	var expectedErr error

	var schema *Schema

	// Should return bytes for Schema when Schema not nil
	schema = new(Schema)
	schema.Type = []string{"string"}

	soa = SchemaOrArray{}
	soa.Schema = schema

	marshalBytes, marshalErr = soa.MarshalJSON()
	expectedBytes, expectedErr = json.Marshal(soa.Schema)

	assert.Equal(t, expectedBytes, marshalBytes, "should return json marshal for schema")
	assert.Equal(t, expectedErr, marshalErr, "should return json marshal for schema")

	// Should return bytes for schema array when schema nil
	schema = new(Schema)
	schema.Type = []string{"string"}

	soa = SchemaOrArray{}
	soa.SchemaArray = []*Schema{schema}

	marshalBytes, marshalErr = soa.MarshalJSON()
	expectedBytes, expectedErr = json.Marshal(soa.SchemaArray)

	assert.Equal(t, expectedBytes, marshalBytes, "should return json marshal for schemaArray")
	assert.Equal(t, expectedErr, marshalErr, "should return json marshal for schemaArray")
}

func TestSchemaOrArrayUnmarshalJSON(t *testing.T) {
	var soa *SchemaOrArray
	var err error
	var testJSON []byte

	expectedSchema := new(Schema)
	expectedSchema.Type = []string{"object"}
	expectedSchema.Format = "some format"

	// Should set Schema when schema json sent
	testJSON, _ = json.Marshal(expectedSchema)

	soa = new(SchemaOrArray)
	err = soa.UnmarshalJSON(testJSON)

	assert.Nil(t, err, "should not error for valid schema json")
	assert.Equal(t, soa.Schema, expectedSchema, "should set schema to schema")

	// Should set schema array when schema array json sent
	expectedSchemaArray := []*Schema{
		expectedSchema,
	}

	testJSON, _ = json.Marshal(expectedSchemaArray)

	soa = new(SchemaOrArray)
	err = soa.UnmarshalJSON(testJSON)

	assert.Nil(t, err, "should not return error for valid boolean true value")
	assert.Equal(t, soa.SchemaArray, expectedSchemaArray, "should set schemaArray to schemaArray from json")

	// Should return error when bytes neither boolean or schema
	soa = new(SchemaOrArray)
	err = soa.UnmarshalJSON([]byte("123"))

	assert.EqualError(t, err, "Can only unmarshal to SchemaOrArray if value is Schema format or array of Schema formats")
}

func TestStringOrArrayMarshalJSON(t *testing.T) {
	var soa StringOrArray
	var marshalBytes []byte
	var marshalErr error

	var expectedBytes []byte
	var expectedErr error

	// Should return bytes for string when array is length 1
	soa = []string{"abc"}

	marshalBytes, marshalErr = soa.MarshalJSON()
	expectedBytes, expectedErr = json.Marshal([]string(soa)[0])

	assert.Equal(t, expectedBytes, marshalBytes, "should return json marshal for single string")
	assert.Equal(t, expectedErr, marshalErr, "should return json marshal for single string")

	// Should return bytes for array when array is length > 1
	soa = []string{"abc", "def"}

	marshalBytes, marshalErr = soa.MarshalJSON()
	expectedBytes, expectedErr = json.Marshal([]string(soa))

	assert.Equal(t, expectedBytes, marshalBytes, "should return json marshal for array")
	assert.Equal(t, expectedErr, marshalErr, "should return json marshal for array")
}

func TestStringOrArrayUnmarshalJSON(t *testing.T) {
	var soa StringOrArray
	var err error

	// Should create stringOrArray when string sent as JSON
	soa.UnmarshalJSON([]byte("\"abc\""))

	assert.Nil(t, err, "should not error for valid schema json")
	assert.Equal(t, []string(soa), []string{"abc"}, "should set string json as single el array")

	// // Should set boolean when array json sent as JSON
	soa.UnmarshalJSON([]byte("[\"abc\", \"def\"]"))

	assert.Nil(t, err, "should not error for valid schema json")
	assert.Equal(t, []string(soa), []string{"abc", "def"}, "should set array json as multi el array")

	// Should return error when bytes neither boolean or schema
	err = soa.UnmarshalJSON([]byte("123"))

	assert.EqualError(t, err, "Can only unmarshal to StringOrArray if value is []string or string")

	// Should return error when invalid JSON
	soa = []string{"abc", "def"}

	expectedErr := json.Unmarshal([]byte("bad json"), "useless value")
	err = soa.UnmarshalJSON([]byte("bad json"))

	assert.Equal(t, expectedErr, err, "should have an error value when bad JSON")
}

func TestGenerateMetadata(t *testing.T) {
	cc := ContractChaincode{}

	// ============================
	// metadata file tests
	// ============================

	var filepath string
	var metadataBytes []byte

	// Should panic when cannot read file but it exists
	filepath = createMetadataJSONFile([]byte("some file contents"), 0000)
	_, readfileErr := ioutil.ReadFile(filepath)
	assert.PanicsWithValue(t, fmt.Sprintf("Failed to generate metadata. Could not read file %s. %s", filepath, readfileErr), func() { generateMetadata(cc) }, "should panic when cannot read file but it exists")
	cleanupMetadataJSONFile()

	// should panic when file does not match schema
	metadataBytes = []byte("{\"some\":\"json\"}")

	filepath = createMetadataJSONFile(metadataBytes, os.ModePerm)
	schemaLoader := gojsonschema.NewBytesLoader([]byte(GetJSONSchema()))
	metadataLoader := gojsonschema.NewBytesLoader(metadataBytes)

	result, _ := gojsonschema.Validate(schemaLoader, metadataLoader)

	var errors string

	for index, desc := range result.Errors() {
		errors = errors + "\n" + strconv.Itoa(index+1) + ".\t" + desc.String()
	}

	assert.PanicsWithValue(t, fmt.Sprintf("Failed to generate metadata. Given file did not match schema: %s", errors), func() { generateMetadata(cc) }, "should panic when file does not meet schema")
	cleanupMetadataJSONFile()

	// should use metadata file data
	metadataBytes = []byte("{\"info\":{\"title\":\"my contract\",\"version\":\"0.0.1\"},\"contracts\":[],\"components\":{}}")

	filepath = createMetadataJSONFile(metadataBytes, os.ModePerm)
	assert.Equal(t, string(metadataBytes), generateMetadata(cc), "should return metadata from file")
	cleanupMetadataJSONFile()

	// ============================
	// Non metadata file tests
	// ============================

	complexType := reflect.TypeOf(complex64(1))

	var getSchemaErr error

	// Should panic if get schema panics
	someBadFunctionContractFunction := new(contractFunction)
	someBadFunctionContractFunction.params = contractFunctionParams{
		basicContextPtrType,
		[]reflect.Type{stringRefType, complexType},
	}
	bcFuncs := make(map[string]*contractFunction)
	bcFuncs["BadFunction"] = someBadFunctionContractFunction
	bcccn := contractChaincodeContract{
		bcFuncs, nil, nil, nil, nil, nil,
	}

	cc.contracts = map[string]contractChaincodeContract{
		"": bcccn,
	}

	_, getSchemaErr = getSchema(complexType)

	assert.PanicsWithValue(t, fmt.Sprintf("Failed to generate metadata. Invalid function parameter type. %s", getSchemaErr), func() { generateMetadata(cc) }, "should have panicked with bad contract function params")

	// Should panic if get schema panics
	anotherBadFunctionContractFunction := new(contractFunction)
	anotherBadFunctionContractFunction.params = contractFunctionParams{
		basicContextPtrType,
		[]reflect.Type{stringRefType},
	}
	anotherBadFunctionContractFunction.returns = contractFunctionReturns{}
	anotherBadFunctionContractFunction.returns.success = complexType
	abcFuncs := make(map[string]*contractFunction)
	abcFuncs["AnotherBadFunction"] = anotherBadFunctionContractFunction
	abcccn := contractChaincodeContract{
		abcFuncs, nil, nil, nil, nil, nil,
	}

	cc.contracts = map[string]contractChaincodeContract{
		"": abcccn,
	}

	_, getSchemaErr = getSchema(complexType)

	assert.PanicsWithValue(t, fmt.Sprintf("Failed to generate metadata. Invalid function success return type. %s", getSchemaErr), func() { generateMetadata(cc) }, "should have panicked with bad contract function success return")

	// setup for not panicking tests

	errorSchema := Schema{}
	errorSchema.Type = []string{"object"}
	errorSchema.Format = "error"

	someFunctionContractFunction := new(contractFunction)

	someFunctionMetadata := TransactionMetadata{}
	someFunctionMetadata.Name = "SomeFunction"

	anotherFunctionContractFunction := new(contractFunction)
	anotherFunctionContractFunction.params = contractFunctionParams{
		basicContextPtrType,
		[]reflect.Type{stringRefType, intRefType},
	}
	anotherFunctionContractFunction.returns = contractFunctionReturns{
		float64RefType,
		true,
	}

	param0AsParam := ParameterMetadata{}
	param0AsParam.Name = "param0"
	param0AsParam.Required = true
	param0AsParam.Schema = *(stringTypeVar.getSchema())

	param1AsParam := ParameterMetadata{}
	param1AsParam.Name = "param1"
	param1AsParam.Required = true
	param1AsParam.Schema = *(intTypeVar.getSchema())

	anotherFunctionMetadata := TransactionMetadata{}
	anotherFunctionMetadata.Parameters = []ParameterMetadata{
		param0AsParam,
		param1AsParam,
	}

	successAsParam := ParameterMetadata{}
	successAsParam.Name = "success"
	successAsParam.Schema = *(float64TypeVar.getSchema())

	errorAsParam := ParameterMetadata{}
	errorAsParam.Name = "error"
	errorAsParam.Schema = errorSchema

	anotherFunctionMetadata.Returns = []ParameterMetadata{
		successAsParam,
		errorAsParam,
	}
	anotherFunctionMetadata.Name = "AnotherFunction"

	var expectedMetadata ContractChaincodeMetadata

	scFuncs := make(map[string]*contractFunction)
	scFuncs["SomeFunction"] = someFunctionContractFunction
	scccn := contractChaincodeContract{
		scFuncs, nil, nil, nil, nil, nil,
	}

	cscFuncs := make(map[string]*contractFunction)
	cscFuncs["SomeFunction"] = someFunctionContractFunction

	cscFuncs["AnotherFunction"] = anotherFunctionContractFunction
	cscccn := contractChaincodeContract{
		cscFuncs, nil, nil, nil, nil, nil,
	}

	// Should handle generating metadata for a single name with default namespacing
	cc.contracts = map[string]contractChaincodeContract{
		"": scccn,
	}
	expectedMetadata = ContractChaincodeMetadata{}
	expectedMetadata.Contracts = []ContractMetadata{
		ContractMetadata{
			Name: "",
			Transactions: []TransactionMetadata{
				someFunctionMetadata,
			},
		},
	}

	testMetadata(t, generateMetadata(cc), expectedMetadata)

	// Should handle generating metadata for a single name with custom name and order functions alphabetically on ID
	cc.contracts = map[string]contractChaincodeContract{
		"customname": cscccn,
	}
	expectedMetadata = ContractChaincodeMetadata{}
	expectedMetadata.Contracts = []ContractMetadata{
		ContractMetadata{
			Name: "customname",
			Transactions: []TransactionMetadata{
				anotherFunctionMetadata,
				someFunctionMetadata,
			},
		},
	}
	testMetadata(t, generateMetadata(cc), expectedMetadata)

	// should handle generating metadata for multiple names
	cc.contracts = map[string]contractChaincodeContract{
		"":           scccn,
		"customname": cscccn,
	}
	expectedMetadata = ContractChaincodeMetadata{}
	expectedMetadata.Contracts = []ContractMetadata{
		ContractMetadata{
			Name: "",
			Transactions: []TransactionMetadata{
				someFunctionMetadata,
			},
		},
		ContractMetadata{
			Name: "customname",
			Transactions: []TransactionMetadata{
				anotherFunctionMetadata,
				someFunctionMetadata,
			},
		},
	}
	testMetadata(t, generateMetadata(cc), expectedMetadata)

	// Should sort the contracts by alphabetical order on their name
	cc.contracts = map[string]contractChaincodeContract{
		"somename":   scccn,
		"customname": cscccn,
	}
	expectedMetadata = ContractChaincodeMetadata{}
	expectedMetadata.Contracts = []ContractMetadata{
		ContractMetadata{
			Name: "customname",
			Transactions: []TransactionMetadata{
				anotherFunctionMetadata,
				someFunctionMetadata,
			},
		},
		ContractMetadata{
			Name: "somename",
			Transactions: []TransactionMetadata{
				someFunctionMetadata,
			},
		},
	}
	testMetadata(t, generateMetadata(cc), expectedMetadata)

	// Should use reflected metadata when getting executable location fails

	oldOsHelper := osHelper
	osHelper = osExcTestStr{}

	metadataBytes = []byte("{\"info\":{\"title\":\"my contract\",\"version\":\"0.0.1\"},\"contracts\":[],\"components\":{}}")

	filepath = createMetadataJSONFile(metadataBytes, os.ModePerm)

	cc.contracts = map[string]contractChaincodeContract{
		"customname": cscccn,
	}
	expectedMetadata = ContractChaincodeMetadata{}
	expectedMetadata.Contracts = []ContractMetadata{
		ContractMetadata{
			Name: "customname",
			Transactions: []TransactionMetadata{
				anotherFunctionMetadata,
				someFunctionMetadata,
			},
		},
	}
	testMetadata(t, generateMetadata(cc), expectedMetadata)

	cleanupMetadataJSONFile()
	osHelper = oldOsHelper
}

// ============== contract_chaincode_helpers.go ==============
func TestAddContract(t *testing.T) {
	ciT := reflect.TypeOf((*ContractInterface)(nil)).Elem()
	var fullExclude []string
	for i := 0; i < ciT.NumMethod(); i++ {
		fullExclude = append(fullExclude, ciT.Method(i).Name)
	}

	cT := reflect.TypeOf(new(Contract))
	for i := 0; i < cT.NumMethod(); i++ {
		methodName := cT.Method(i).Name
		if !stringInSlice(methodName, fullExclude) {
			fullExclude = append(fullExclude, methodName)
		}
	}

	var cc *ContractChaincode
	sc := simpleTestContract{}
	csc := simpleTestContract{}
	csc.name = "customname"

	// Should panic when contract passed with non unique name
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	cc.contracts[""] = contractChaincodeContract{}
	assert.PanicsWithValue(t, "Multiple contracts being merged into chaincode without a name", func() { cc.addContract(new(simpleTestContract), []string{}) }, "didn't panic when multiple contracts share same name")

	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	cc.contracts["customname"] = contractChaincodeContract{}
	sc = simpleTestContract{}
	sc.SetName("customname")
	assert.PanicsWithValue(t, "Multiple contracts being merged into chaincode with name customname", func() { cc.addContract(&sc, []string{}) }, "didn't panic when multiple contracts share same custom name")
	sc = simpleTestContract{}

	// Should add contract with default name to chaincode
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	cc.addContract(&sc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts[""], sc)

	// Should add contract with custom name to chaincode
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	cc.addContract(&csc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts["customname"], csc)

	// Should add to map of chaincode not remove other chaincodes
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	cc.addContract(&sc, fullExclude)
	cc.addContract(&csc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts[""], sc)
	testContractChaincodeContractRepresentsContract(t, cc.contracts["customname"], csc)

	// Should add contract to map with unknown transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	sc.unknownTransaction = sc.DoSomething
	cc.addContract(&sc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts[""], sc)
	sc.unknownTransaction = nil

	// Should add contract to map with before transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	sc.beforeTransaction = sc.DoSomething
	cc.addContract(&sc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts[""], sc)
	sc.beforeTransaction = nil

	// Should add contract to map with after transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	sc.afterTransaction = sc.DoSomething
	cc.addContract(&sc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts[""], sc)
	sc.afterTransaction = nil
}

func TestConvertC2CC(t *testing.T) {
	sc := simpleTestContract{}

	csc := simpleTestContract{}
	csc.name = "customname"

	// Should create a valid chaincode from a single contract with a default ns
	testConvertCC(t, []simpleTestContract{sc})

	// Should create a valid chaincode from a single contract with a custom ns
	testConvertCC(t, []simpleTestContract{csc})

	// Should create a valid chaincode from multiple smart contracts
	testConvertCC(t, []simpleTestContract{sc, csc})

	// Should panic when contract has function with same name as a Contract function but does not embed Contract and function is invalid
	assert.PanicsWithValue(t, fmt.Sprintf("SetAfterTransaction contains invalid parameter type. Type interface {} is not valid. Expected one of the basic types %s, an array/slice of these, or one of these additional types %s", listBasicTypes(), basicContextPtrType.String()), func() { convertC2CC(new(Contract)) }, "should have panicked due to bad function format")
}

// ============== contract_chaincode_def.go ==============

func TestCreateNewChaincode(t *testing.T) {
	mc := new(myContract)

	// Should call shim.Start
	assert.EqualError(t, CreateNewChaincode(mc), shim.Start(convertC2CC(mc)).Error(), "should return same as shim.start")
}

func TestInit(t *testing.T) {
	// Should just return when no function name passed
	cc := convertC2CC()
	mockStub := shim.NewMockStub("blank fcn", cc)
	assert.Equal(t, shim.Success([]byte("Default initiator successful.")), cc.Init(mockStub), "should just return success on init with no function passed")

	// Should call via invoke
	testCallingContractFunctions(t, initType)
}

func TestInvoke(t *testing.T) {
	testCallingContractFunctions(t, invokeType)
}
