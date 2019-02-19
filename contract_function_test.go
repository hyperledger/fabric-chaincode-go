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
	"reflect"
	"sort"
	"testing"

	"github.com/go-openapi/spec"

	"github.com/stretchr/testify/assert"
)

// ================================
// Helpers
// ================================

var badType = reflect.TypeOf(complex64(1))
var badArrayType = reflect.TypeOf([1]complex64{})
var badSliceType = reflect.TypeOf([]complex64{})
var badMapItemType = reflect.TypeOf(map[string]complex64{})
var badMapKeyType = reflect.TypeOf(map[complex64]string{})

var errorType = reflect.TypeOf((*error)(nil)).Elem()

const basicErr = "Type %s is not valid. Expected a struct or one of the basic types %s or an array/slice of these"

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

	assert.EqualError(t, err, fmt.Sprintf("%s contains invalid first return type. Type error is not valid. Expected a struct or one of the basic types %s or an array/slice of these", methodName, listBasicTypes()), "should return expected error for bad first return type")

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

func testCreateArraySliceMapOrStructErrors(t *testing.T, json string, arrType reflect.Type) {
	t.Helper()

	val, err := createArraySliceMapOrStruct(json, arrType)

	assert.EqualError(t, err, fmt.Sprintf("Value %s was not passed in expected format %s", json, arrType.String()), "should error when invalid JSON")
	assert.Equal(t, reflect.Value{}, val, "should return an empty value when error found")
}

func setContractFunctionParams(cf *contractFunction, context reflect.Type, fields []reflect.Type) {
	cfp := contractFunctionParams{}

	cfp.context = context
	cfp.fields = fields
	cf.params = cfp
}

func callGetArgsAndBasicTest(t *testing.T, cf contractFunction, ctx *TransactionContext, supplementaryMetadata *TransactionMetadata, components *ComponentMetadata, testParams []string) []reflect.Value {
	t.Helper()

	values, err := getArgs(cf, reflect.ValueOf(ctx), supplementaryMetadata, components, testParams)

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

		values := callGetArgsAndBasicTest(t, cf, ctx, nil, nil, params)
		testReflectValueEqualSlice(t, values, expectedArgs)
	}
}

func setContractFunctionReturns(cf *contractFunction, successReturn reflect.Type, returnsError bool) {
	cfr := contractFunctionReturns{}
	cfr.success = successReturn
	cfr.error = returnsError

	cf.returns = cfr
}

func testHandleResponse(t *testing.T, successReturn reflect.Type, errorReturn bool, response []reflect.Value, expectedString string, expectedValue interface{}, expectedError error) {
	t.Helper()

	cf := contractFunction{}

	setContractFunctionReturns(&cf, successReturn, errorReturn)
	strResp, valueResp, errResp := handleContractFunctionResponse(response, cf)

	assert.Equal(t, expectedString, strResp, "should have returned string value from response")
	assert.Equal(t, expectedValue, valueResp, "should have returned actual value from response")
	assert.Equal(t, expectedError, errResp, "should have returned error value from response")
}

// ================================
// Tests
// ================================

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

	// Should return error when typeIsValid would error for base type
	badArr := [1]complex128{}
	err = arrayOfValidType(reflect.ValueOf(badArr))

	assert.EqualError(t, err, typeIsValid(reflect.TypeOf(complex128(1)), []reflect.Type{}).Error(), "should throw error when invalid type passed")
}

func TestStructOfValidType(t *testing.T) {
	// should handle pointers
	assert.Nil(t, structOfValidType(reflect.TypeOf(new(GoodStruct))), "should not return an error for a pointer struct")

	// should handle bad pointer
	assert.Nil(t, structOfValidType(reflect.TypeOf(new(GoodStruct))), "should not return an error for a pointer struct")

	// Should allow a struct where properties are of valid types
	assert.Nil(t, structOfValidType(reflect.TypeOf(GoodStruct{})), "should not return an error for a valid struct")

	// Should return an error when properties are not valid types
	assert.EqualError(t, structOfValidType(reflect.TypeOf(BadStruct{})), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should return an error for invalid struct")
}

func TestTypeIsValid(t *testing.T) {
	badArr := reflect.New(badArrayType).Elem()

	// Should return error is non-array/slice type is invalid
	assert.EqualError(t, typeIsValid(badType, []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should have returned error for invalid type")

	// Should return error for bad array returned by arrayOfValidType
	assert.EqualError(t, typeIsValid(badArrayType, []reflect.Type{}), arrayOfValidType(badArr).Error(), "should have returned error for invalid array type")

	// Should return error for bad slice
	assert.EqualError(t, typeIsValid(badSliceType, []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should have returned error for invalid slice type")

	// Should return error for bad map item
	assert.EqualError(t, typeIsValid(badMapItemType, []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should have returned error for invalid slice type")

	// Should return error for bad map key
	assert.EqualError(t, typeIsValid(badMapKeyType, []reflect.Type{}), "Map key type complex64 is not valid. Expected string", "should have returned error for invalid slice type")

	// Should accept basic types
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

	mc := myContract{}
	mcFuncType := reflect.TypeOf(mc.AfterTransactionWithInterface)

	assert.Nil(t, typeIsValid(mcFuncType.In(1), []reflect.Type{}))

	// Should return nil for arrays made of each of the basic types
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]string{}), []reflect.Type{}), "should not return an error for a string array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]bool{}), []reflect.Type{}), "should not return an error for a bool array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]int{}), []reflect.Type{}), "should not return an error for an int array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]int8{}), []reflect.Type{}), "should not return an error for an int8 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]int16{}), []reflect.Type{}), "should not return an error for an int16 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]int32{}), []reflect.Type{}), "should not return an error for an int32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]int64{}), []reflect.Type{}), "should not return an error for an int64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]uint{}), []reflect.Type{}), "should not return an error for a uint array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]uint8{}), []reflect.Type{}), "should not return an error for a uint8 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]uint16{}), []reflect.Type{}), "should not return an error for a uint16 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]uint32{}), []reflect.Type{}), "should not return an error for a uint32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]uint64{}), []reflect.Type{}), "should not return an error for a uint64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]float32{}), []reflect.Type{}), "should not return an error for a float32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]float64{}), []reflect.Type{}), "should not return an error for a float64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]byte{}), []reflect.Type{}), "should not return an error for a float64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]rune{}), []reflect.Type{}), "should not return an error for a float64 array type")

	// should return error for multidimensional array where length of inner array is 0
	zeroMultiArr := [1][0]int{}
	err := typeIsValid(reflect.TypeOf(zeroMultiArr), []reflect.Type{})

	assert.Equal(t, errors.New("Arrays must have length greater than 0"), err, "should throw error when 0 length array passed")

	// Should return error when multidimensional array is not valid
	badMultiArr := [1][1]complex128{}
	err = typeIsValid(reflect.TypeOf(badMultiArr), []reflect.Type{})

	assert.Equal(t, fmt.Errorf(basicErr, "complex128", listBasicTypes()), err, "should throw error when bad multidimensional array passed")

	// Should return nil for multidimensional arrays made of each of the basic types
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]string{}), []reflect.Type{}), "should not return an error for a multidimensional string array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]bool{}), []reflect.Type{}), "should not return an error for a multidimensional bool array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]int{}), []reflect.Type{}), "should not return an error for an multidimensional int array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]int8{}), []reflect.Type{}), "should not return an error for an multidimensional int8 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]int16{}), []reflect.Type{}), "should not return an error for an multidimensional int16 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]int32{}), []reflect.Type{}), "should not return an error for an multidimensional int32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]int64{}), []reflect.Type{}), "should not return an error for an multidimensional int64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]uint{}), []reflect.Type{}), "should not return an error for a multidimensional uint array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]uint8{}), []reflect.Type{}), "should not return an error for a multidimensional uint8 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]uint16{}), []reflect.Type{}), "should not return an error for a multidimensional uint16 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]uint32{}), []reflect.Type{}), "should not return an error for a multidimensional uint32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]uint64{}), []reflect.Type{}), "should not return an error for a multidimensional uint64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]float32{}), []reflect.Type{}), "should not return an error for a multidimensional float32 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]float64{}), []reflect.Type{}), "should not return an error for a multidimensional float64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]byte{}), []reflect.Type{}), "should not return an error for a multidimensional float64 array type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][1]rune{}), []reflect.Type{}), "should not return an error for a multidimensional float64 array type")

	// Should handle an array many dimensions
	assert.Nil(t, typeIsValid(reflect.TypeOf([1][2][3][4][5][6][7][8]string{}), []reflect.Type{}), "should not return an error for a very multidimensional string array type")

	// Should handle an array of slices
	assert.Nil(t, typeIsValid(reflect.TypeOf([2][]string{}), []reflect.Type{}), "should not return an error for a string array of slice type")

	// Should return nil for arrays made of each of the basic types
	assert.Nil(t, typeIsValid(reflect.TypeOf([]string{}), []reflect.Type{}), "should not return an error for a string slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]bool{}), []reflect.Type{}), "should not return an error for a bool slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]int{}), []reflect.Type{}), "should not return an error for a int slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]int8{}), []reflect.Type{}), "should not return an error for a int8 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]int16{}), []reflect.Type{}), "should not return an error for a int16 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]int32{}), []reflect.Type{}), "should not return an error for a int32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]int64{}), []reflect.Type{}), "should not return an error for a int64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]uint{}), []reflect.Type{}), "should not return an error for a uint slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]uint8{}), []reflect.Type{}), "should not return an error for a uint8 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]uint16{}), []reflect.Type{}), "should not return an error for a uint16 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]uint32{}), []reflect.Type{}), "should not return an error for a uint32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]uint64{}), []reflect.Type{}), "should not return an error for a uint64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]float32{}), []reflect.Type{}), "should not return an error for a float32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]float64{}), []reflect.Type{}), "should not return an error for a float64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]byte{}), []reflect.Type{}), "should not return an error for a byte slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([]rune{}), []reflect.Type{}), "should not return an error for a rune slice type")

	// Should return error when multidimensional slice is not valid
	badMultiSlice := [][]complex128{}
	err = typeIsValid(reflect.TypeOf(badMultiSlice), []reflect.Type{})

	assert.Equal(t, fmt.Errorf(basicErr, "complex128", listBasicTypes()), err, "should throw error when 0 length array passed")

	// Should return nil for multidimensional slices made of each of the basic types
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]string{}), []reflect.Type{}), "should not return an error for a multidimensional string slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]bool{}), []reflect.Type{}), "should not return an error for a multidimensional bool slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]int{}), []reflect.Type{}), "should not return an error for a multidimensional int slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]int8{}), []reflect.Type{}), "should not return an error for a multidimensional int8 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]int16{}), []reflect.Type{}), "should not return an error for a multidimensional int16 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]int32{}), []reflect.Type{}), "should not return an error for a multidimensional int32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]int64{}), []reflect.Type{}), "should not return an error for a multidimensional int64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]uint{}), []reflect.Type{}), "should not return an error for a multidimensional uint slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]uint8{}), []reflect.Type{}), "should not return an error for a multidimensional uint8 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]uint16{}), []reflect.Type{}), "should not return an error for a multidimensional uint16 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]uint32{}), []reflect.Type{}), "should not return an error for a multidimensional uint32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]uint64{}), []reflect.Type{}), "should not return an error for a multidimensional uint64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]float32{}), []reflect.Type{}), "should not return an error for a multidimensional float32 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]float64{}), []reflect.Type{}), "should not return an error for a multidimensional float64 slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]byte{}), []reflect.Type{}), "should not return an error for a multidimensional byte slice type")
	assert.Nil(t, typeIsValid(reflect.TypeOf([][]rune{}), []reflect.Type{}), "should not return an error for a multidimensional rune slice type")

	// Should handle a slice many dimensions
	assert.Nil(t, typeIsValid(reflect.TypeOf([][][][][][][][]string{}), []reflect.Type{}), "should not return an error for a very multidimensional string slice type")

	// Should handle a slice of arrays
	assert.Nil(t, typeIsValid(reflect.TypeOf([2][]string{}), []reflect.Type{}), "should not return an error for a string slice of array type")

	// Should allow a struct where properties are of valid types
	assert.Nil(t, typeIsValid(reflect.TypeOf(GoodStruct{}), []reflect.Type{}), "should not return an error for a valid struct")

	// Should allow array of valid struct
	assert.Nil(t, typeIsValid(reflect.TypeOf([1]GoodStruct{}), []reflect.Type{}), "should not return an error for an array of valid struct")

	// Should allow slice of valid struct
	assert.Nil(t, typeIsValid(reflect.TypeOf([]GoodStruct{}), []reflect.Type{}), "should not return an error for a slice of valid struct")

	// Should allow maps with value of basic types
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]string{}), []reflect.Type{}), "should not return an error for a map string item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]bool{}), []reflect.Type{}), "should not return an error for a map bool item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]int{}), []reflect.Type{}), "should not return an error for a map int item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]int8{}), []reflect.Type{}), "should not return an error for a map int8 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]int16{}), []reflect.Type{}), "should not return an error for a map int16 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]int32{}), []reflect.Type{}), "should not return an error for a map int32 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]int64{}), []reflect.Type{}), "should not return an error for a map int64 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]uint{}), []reflect.Type{}), "should not return an error for a map uint item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]uint8{}), []reflect.Type{}), "should not return an error for a map uint8 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]uint16{}), []reflect.Type{}), "should not return an error for a map uint16 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]uint32{}), []reflect.Type{}), "should not return an error for a map uint32 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]uint64{}), []reflect.Type{}), "should not return an error for a map uint64 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]float32{}), []reflect.Type{}), "should not return an error for a map float32 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]float64{}), []reflect.Type{}), "should not return an error for a map float64 item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]byte{}), []reflect.Type{}), "should not return an error for a map byte item type")
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]rune{}), []reflect.Type{}), "should not return an error for a map rune item type")

	// Should allow maps of maps
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]map[string]string{}), []reflect.Type{}), "should not return an error for a map of map")

	// Should allow maps of structs
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string]GoodStruct{}), []reflect.Type{}), "should not return an error for a map with struct item type")

	// Should allow maps of arrays
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string][1]string{}), []reflect.Type{}), "should not return an error for a map with string array item type")

	// Should allow maps of slices
	assert.Nil(t, typeIsValid(reflect.TypeOf(map[string][]string{}), []reflect.Type{}), "should not return an error for a map with string slice item type")

	// Should allow struct with property of struct
	type GoodStruct2 struct {
		Prop1 GoodStruct
	}

	assert.Nil(t, typeIsValid(reflect.TypeOf(GoodStruct2{}), []reflect.Type{}), "should not return an error for a valid struct with struct property")

	// Should allow struct with pointer property
	type GoodStruct3 struct {
		Prop1 *GoodStruct
	}

	assert.Nil(t, typeIsValid(reflect.TypeOf(GoodStruct3{}), []reflect.Type{}), "should not return an error for a valid struct with struct ptr property")

	// should return error for array of invalid struct
	assert.EqualError(t, typeIsValid(reflect.TypeOf([]BadStruct{}), []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should return an error for array of invalid struct")

	// should return error for slice of invalid struct
	assert.EqualError(t, typeIsValid(reflect.TypeOf([]BadStruct{}), []reflect.Type{}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should return an error for slice of invalid struct")

	// Should not return an error when struct is bad but is listed in allowedTypes
	assert.Nil(t, typeIsValid(reflect.TypeOf(BadStruct{}), []reflect.Type{reflect.TypeOf(BadStruct{})}), "should not return error when bad struct is in list of additional types")

	// Should return error when struct has property that is a bad struct
	type BadStruct2 struct {
		Prop1 BadStruct
	}

	assert.EqualError(t, structOfValidType(reflect.TypeOf(BadStruct2{})), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should return an error for struct with invalid property of a struct")

	type BadStruct3 struct {
		Prop1 *BadStruct
	}

	assert.EqualError(t, structOfValidType(reflect.TypeOf(BadStruct2{})), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should return an error for struct with invalid property of a pointer to struct")

	// Should not return an error when struct is bad but is listed in allowedTypes
	assert.Nil(t, typeIsValid(reflect.TypeOf(BadStruct{}), []reflect.Type{reflect.TypeOf(BadStruct{})}), "should not return error when bad struct is in list of additional types")

	// Should accept value if not in basic types but in additional types
	assert.Nil(t, typeIsValid(badType, []reflect.Type{badType}), "should not error when type not in basic types but is in additional types")

	// Should not handle arrays as additional types
	assert.EqualError(t, typeIsValid(badArrayType, []reflect.Type{badArrayType}), arrayOfValidType(badArr).Error(), "should have returned error for invalid array type")

	// Should not handle slices as additional types
	assert.EqualError(t, typeIsValid(badSliceType, []reflect.Type{badSliceType}), fmt.Sprintf(basicErr, badType.String(), listBasicTypes()), "should have returned error for invalid slice type")
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

func TestCreateArraySliceMapOrStruct(t *testing.T) {
	var val reflect.Value
	var err error

	arrType := reflect.TypeOf([2]string{})
	multiDArrType := reflect.TypeOf([2][1]string{})
	sliceType := reflect.TypeOf([]string{})
	multiDSliceType := reflect.TypeOf([][]string{})
	arrOfSliceType := reflect.TypeOf([2][]string{})
	sliceOfArrType := reflect.TypeOf([][2]string{})
	goodStructType := reflect.TypeOf(GoodStruct{})
	anotherGoodStructType := reflect.TypeOf(AnotherGoodStruct{})
	arrayGoodStructType := reflect.TypeOf([1]GoodStruct{})

	// should error when passed data is not json
	testCreateArraySliceMapOrStructErrors(t, "bad JSON", arrType)

	// should error when passed data is json but not valid for the unmarshalling
	testCreateArraySliceMapOrStructErrors(t, "{\"some\": \"object\"}", arrType)

	// Should error when array passed but it is too deep
	testCreateArraySliceMapOrStructErrors(t, "[[\"a\"],[\"b\"]]", arrType)

	// Should error when array passed but it is too shallow
	testCreateArraySliceMapOrStructErrors(t, "[\"a\",\"b\"]", multiDArrType)

	// Should error when slice passed but it is too deep
	testCreateArraySliceMapOrStructErrors(t, "[[\"a\"],[\"b\"]]", sliceType)

	// Should error when slice passed but it is too deep
	testCreateArraySliceMapOrStructErrors(t, "[\"a\",\"b\"]", multiDSliceType)

	// Should return error when array passed but contains data of the wrong type
	testCreateArraySliceMapOrStructErrors(t, "[\"a\", 1]", arrType)

	// Should return error when slice passed but contains data of the wrong type
	testCreateArraySliceMapOrStructErrors(t, "[\"a\", 1]", sliceType)

	// Should return error when type wrong for array of slice
	testCreateArraySliceMapOrStructErrors(t, "[[\"a\"],[1]]", arrOfSliceType)

	// Should return error when type wrong for array of slice
	testCreateArraySliceMapOrStructErrors(t, "[[\"a\", 1]]", sliceOfArrType)

	// Should return error when doesn't match struct type
	testCreateArraySliceMapOrStructErrors(t, "{\"Prop1\": 1}", goodStructType)

	// Should return error when doesn't match sub struct type
	testCreateArraySliceMapOrStructErrors(t, "{\"StructProp\": {\"Prop1\": 1}}", anotherGoodStructType)

	// Should return error when doesn't match struct type in array
	testCreateArraySliceMapOrStructErrors(t, "[{\"Prop1\": 1}]", arrayGoodStructType)

	// Should return reflect value for array
	val, err = createArraySliceMapOrStruct("[\"a\",\"b\"]", arrType)

	assert.Nil(t, err, "should have nil error for valid array passed")
	assert.Equal(t, [2]string{"a", "b"}, val.Interface().([2]string), "should have returned value of array with filled in data")

	// Should return reflect value for md array
	val, err = createArraySliceMapOrStruct("[[\"a\"],[\"b\"]]", multiDArrType)

	assert.Nil(t, err, "should have nil error for valid array passed")
	assert.Equal(t, [2][1]string{{"a"}, {"b"}}, val.Interface().([2][1]string), "should have returned value of multidimensional array with filled in data")

	// Should return reflect value for slice
	val, err = createArraySliceMapOrStruct("[\"a\",\"b\"]", sliceType)

	assert.Nil(t, err, "should have nil error for valid slice passed")
	assert.Equal(t, []string{"a", "b"}, val.Interface().([]string), "should have returned value of slice with filled in data")

	// Should return reflect value for md slice
	val, err = createArraySliceMapOrStruct("[[\"a\"],[\"b\"]]", multiDSliceType)

	assert.Nil(t, err, "should have nil error for valid slice passed")
	assert.Equal(t, [][]string{{"a"}, {"b"}}, val.Interface().([][]string), "should have returned value of multidimensional slice with filled in data")

	// Should return reflect value for an array of slices
	val, err = createArraySliceMapOrStruct("[[\"a\"],[\"b\"]]", arrOfSliceType)

	assert.Nil(t, err, "should have nil error for valid slice passed")
	assert.Equal(t, [2][]string{{"a"}, {"b"}}, val.Interface().([2][]string), "should have returned value of array of slices with filled in data")

	// Should return reflect value for a slice of arrays
	val, err = createArraySliceMapOrStruct("[[\"a\", \"b\"]]", sliceOfArrType)

	assert.Nil(t, err, "should have nil error for valid slice passed")
	assert.Equal(t, [][2]string{{"a", "b"}}, val.Interface().([][2]string), "should have returned value of slice of arrays with filled in data")

	// Should return reflect value for map
	val, err = createArraySliceMapOrStruct("{\"bob\": 1}", reflect.TypeOf(map[string]int{}))

	assert.Nil(t, err, "should have nil error for valid map passed")
	assert.Equal(t, map[string]int{
		"bob": 1,
	}, val.Interface().(map[string]int), "should have returned value of array with filled in data")

	// Should return reflect value for map of struct
	val, err = createArraySliceMapOrStruct("{\"bob\": {\"Prop1\": \"hello\",\"prop2\": 1}}", reflect.TypeOf(map[string]GoodStruct{}))

	assert.Nil(t, err, "should have nil error for valid map passed")
	assert.Equal(t, map[string]GoodStruct{
		"bob": GoodStruct{
			"hello",
			1,
			"",
		},
	}, val.Interface().(map[string]GoodStruct), "should have returned value of array with filled in data")

	// Should return reflect value for map of map
	val, err = createArraySliceMapOrStruct("{\"bob\": {\"fred\": 1}}", reflect.TypeOf(map[string]map[string]int{}))

	assert.Nil(t, err, "should have nil error for valid map passed")
	assert.Equal(t, map[string]map[string]int{
		"bob": map[string]int{
			"fred": 1,
		},
	}, val.Interface().(map[string]map[string]int), "should have returned value of array with filled in data")

	// should return reflect value for a struct
	val, err = createArraySliceMapOrStruct("{\"Prop1\": \"Hello world\", \"prop2\": 1}", goodStructType)

	assert.Nil(t, err, "should have nil error for valid slice passed")
	assert.Equal(t, GoodStruct{"Hello world", 1, ""}, val.Interface().(GoodStruct), "should have returned value of slice of arrays with filled in data")

	// should return reflect value for a struct array
	val, err = createArraySliceMapOrStruct("[{\"Prop1\": \"Hello world\", \"prop2\": 1}]", arrayGoodStructType)

	assert.Nil(t, err, "should have nil error for valid slice passed")
	assert.Equal(t, [1]GoodStruct{GoodStruct{"Hello world", 1, ""}}, val.Interface().([1]GoodStruct), "should have returned value of slice of arrays with filled in data")

	// should return reflect value for a struct containing a struct
	val, err = createArraySliceMapOrStruct("{\"StringProp\": \"Hello World\", \"StructProp\": {\"Prop1\": \"Hello world\", \"prop2\": 1}}", anotherGoodStructType)

	assert.Nil(t, err, "should have nil error for valid slice passed")
	assert.Equal(t, AnotherGoodStruct{"Hello World", GoodStruct{"Hello world", 1, ""}}, val.Interface().(AnotherGoodStruct), "should have returned value of slice of arrays with filled in data")
}

func TestGetArgs(t *testing.T) {
	var values []reflect.Value
	var err error
	var ok bool
	testParams := []string{"one", "two", "three"}

	ctx := new(TransactionContext)
	cf := contractFunction{}

	// Should error when not enough params sent
	setContractFunctionParams(&cf, nil, []reflect.Type{
		stringRefType,
	})

	values, err = getArgs(cf, reflect.ValueOf(ctx), nil, nil, []string{})
	assert.Nil(t, values, "should not return values when parameter data bad")
	assert.Contains(t, err.Error(), "Incorrect number of params. Expected 1, received 0", "should error when missing params")

	// should error when supplementary JSON has not enough params
	tm := new(TransactionMetadata)
	tm.Parameters = []ParameterMetadata{}

	setContractFunctionParams(&cf, nil, []reflect.Type{
		stringRefType,
	})

	values, err = getArgs(cf, reflect.ValueOf(ctx), tm, nil, []string{})
	assert.Nil(t, values, "should not return values when parameter data bad")
	assert.Contains(t, err.Error(), "Incorrect number of params in supplementary metadata. Expected 1, received 0", "should error when missing params")

	// Should return empty array when contract function takes no params
	setContractFunctionParams(&cf, nil, []reflect.Type{})

	callGetArgsAndBasicTest(t, cf, ctx, nil, nil, testParams)

	// Should return array using passed parameters when contract function takes same number of params as sent
	setContractFunctionParams(&cf, nil, []reflect.Type{
		stringRefType,
		stringRefType,
		stringRefType,
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, testParams)

	testReflectValueEqualSlice(t, values, testParams)

	testReflectValueEqualSlice(t, values, append(testParams, "[0 0 0]")) // <- array formatted as sprintf turns to string

	// Should include ctx in returned values and no params when function only takes ctx
	setContractFunctionParams(&cf, basicContextPtrType, []reflect.Type{})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, testParams)

	_, ok = values[0].Interface().(*TransactionContext)

	assert.True(t, ok, "first parameter should be *TransactionContext when takesContext")

	// Should include ctx in returned values and params when function takes in params and ctx
	setContractFunctionParams(&cf, basicContextPtrType, []reflect.Type{
		stringRefType,
		stringRefType,
		stringRefType,
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, testParams)

	_, ok = values[0].Interface().(*TransactionContext)

	assert.True(t, ok, "first parameter should be *TransactionContext when takesContext")

	testReflectValueEqualSlice(t, values[1:], testParams)

	// Should be using context passed
	setContractFunctionParams(&cf, reflect.TypeOf(new(customContext)), []reflect.Type{})

	values, err = getArgs(cf, reflect.ValueOf(new(customContext)), nil, nil, testParams)

	assert.Nil(t, err, "should not return an error for a valid cf")
	assert.Equal(t, 1, len(values), "should return same length array list as number of fields plus 1 for context")

	_, ok = values[0].Interface().(*customContext)

	assert.True(t, ok, "first parameter should be *TransactionContext when takesContext")

	testReflectValueEqualSlice(t, values[1:], testParams)

	// Should handle bool
	setContractFunctionParams(&cf, nil, []reflect.Type{
		boolRefType,
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, []string{"true"})
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

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, []string{"65"})
	testReflectValueEqualSlice(t, values, []byte{65})

	// Should handle runes
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf(rune(65)),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, []string{"65"})
	testReflectValueEqualSlice(t, values, []rune{65})

	// Should handle interface by just returning what was sent
	mc := myContract{}
	mcFuncType := reflect.TypeOf(mc.AfterTransactionWithInterface)

	setContractFunctionParams(&cf, nil, []reflect.Type{
		mcFuncType.In(1),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, []string{"interface!"})
	testReflectValueEqualSlice(t, values, []string{"interface!"})

	// Should return an error if conversion errors
	setContractFunctionParams(&cf, nil, []reflect.Type{
		intRefType,
	})

	values, err = getArgs(cf, reflect.ValueOf(ctx), nil, nil, []string{"abc"})

	assert.EqualError(t, err, "Param abc could not be converted to type int", "should have returned error when convert returns error")
	assert.Nil(t, values, "should not have returned value list on error")

	// Should handle array of basic type
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([4]int{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, []string{"[1,2,3,4]"})
	testReflectValueEqualSlice(t, values, [][4]int{{1, 2, 3, 4}})

	// Should handle multidimensional array of basic type
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([4][1]int{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, []string{"[[1],[2],[3],[4]]"})
	testReflectValueEqualSlice(t, values, [][4][1]int{{{1}, {2}, {3}, {4}}})

	// Should error when the array they pass is not the correct format
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([4]int{}),
	})

	values, err = getArgs(cf, reflect.ValueOf(ctx), nil, nil, []string{"[1,2,3,\"a\"]"})
	assert.EqualError(t, err, "Value [1,2,3,\"a\"] was not passed in expected format [4]int", "should have returned error when array conversion returns error")
	assert.Nil(t, values, "should not have returned value list on error")

	// Should error when the element in multidimensional array they pass is not the correct format
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([4][1]int{}),
	})

	values, err = getArgs(cf, reflect.ValueOf(ctx), nil, nil, []string{"[[1],[2],[3],[\"a\"]]"})
	assert.EqualError(t, err, "Value [[1],[2],[3],[\"a\"]] was not passed in expected format [4][1]int", "should have returned error when array conversion returns error")
	assert.Nil(t, values, "should not have returned value list on error")

	// should handle map of basic type
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf(map[string]int{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, []string{"{\"bob\": 1}"})
	testReflectValueEqualSlice(t, values, []map[string]int{map[string]int{
		"bob": 1,
	}})

	// should handle struct
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf(GoodStruct{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, []string{"{\"Prop1\": \"Hello world\", \"prop2\": 1}"})
	testReflectValueEqualSlice(t, values, []GoodStruct{{"Hello world", 1, ""}})

	// should error when struct properties are invalid
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf(GoodStruct{}),
	})

	values, err = getArgs(cf, reflect.ValueOf(ctx), nil, nil, []string{"{\"Prop1\": \"Hello world\" \"prop2\": \"\"}"})
	assert.EqualError(t, err, "Value {\"Prop1\": \"Hello world\" \"prop2\": \"\"} was not passed in expected format contractapi.GoodStruct", "should have returned error when array conversion returns error")
	assert.Nil(t, values, "should not have returned value list on error")

	// should handle struct in struct
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf(AnotherGoodStruct{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, []string{"{\"StringProp\": \"Hello world\", \"StructProp\": {\"Prop1\": \"Goodbye world\", \"prop2\": 1}}"})
	testReflectValueEqualSlice(t, values, []AnotherGoodStruct{{"Hello world", GoodStruct{"Goodbye world", 1, ""}}})

	// Should handle an array of slices of a basic type
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([4][]int{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, []string{"[[1, 2],[3],[4],[5]]"})
	testReflectValueEqualSlice(t, values, [][4][]int{{{1, 2}, {3}, {4}, {5}}})

	// Should handle a slice of arrays of a basic type
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf([][4]int{}),
	})

	values = callGetArgsAndBasicTest(t, cf, ctx, nil, nil, []string{"[[1,2,3,4]]"})
	testReflectValueEqualSlice(t, values, [][][4]int{{{1, 2, 3, 4}}})

	// Should error when a parameter given doesn't match the schema
	setContractFunctionParams(&cf, nil, []reflect.Type{
		intRefType,
	})

	txMetadata := TransactionMetadata{}
	paramsMetadata := ParameterMetadata{}
	min := float64(0)
	paramsMetadata.Schema.Minimum = &min
	txMetadata.Parameters = make([]ParameterMetadata, 1)
	txMetadata.Parameters[0] = paramsMetadata

	values, err = getArgs(cf, reflect.ValueOf(ctx), &txMetadata, nil, []string{"-1"})
	assert.Nil(t, values, "should not return values when parameter data bad")
	assert.Contains(t, err.Error(), "did not match schema", "should error when schema bad")

	// Should error on missing required fields in metadata
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf(GoodStruct{}),
	})

	txMetadata = TransactionMetadata{}
	paramSchema := spec.Schema{}
	paramSchema.Required = []string{"prop1"}
	paramsMetadata = ParameterMetadata{}
	paramsMetadata.Schema = paramSchema
	txMetadata.Parameters = make([]ParameterMetadata, 1)
	txMetadata.Parameters[0] = paramsMetadata

	values, err = getArgs(cf, reflect.ValueOf(ctx), &txMetadata, nil, []string{"{}"})
	assert.Nil(t, values, "should not return values when parameter data bad")
	assert.Contains(t, err.Error(), "did not match schema", "should error when schema bad")

	// Should error on no extra fields in metadata
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf(GoodStruct{}),
	})

	txMetadata = TransactionMetadata{}
	paramSchema = spec.Schema{}
	paramSchema.AdditionalProperties = &spec.SchemaOrBool{false, nil}
	paramsMetadata = ParameterMetadata{}
	paramsMetadata.Schema = paramSchema
	txMetadata.Parameters = make([]ParameterMetadata, 1)
	txMetadata.Parameters[0] = paramsMetadata

	values, err = getArgs(cf, reflect.ValueOf(ctx), &txMetadata, nil, []string{"{\"additionalProp\": \"some val\"}"})
	assert.Nil(t, values, "should not return values when parameter data bad")
	assert.Contains(t, err.Error(), "did not match schema", "should error when schema bad")

	// Should handle a ref to component
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf(GoodStruct{}),
	})

	txMetadata = TransactionMetadata{}
	paramSchema = *(spec.RefProperty("#/components/schemas/GoodStruct"))
	paramsMetadata = ParameterMetadata{}
	paramsMetadata.Schema = paramSchema
	txMetadata.Parameters = make([]ParameterMetadata, 1)
	txMetadata.Parameters[0] = paramsMetadata

	components := ComponentMetadata{}
	components.Schemas = make(map[string]ObjectMetadata)
	components.Schemas["GoodStruct"] = goodStructMetadata

	values = callGetArgsAndBasicTest(t, cf, ctx, &txMetadata, &components, []string{"{\"Prop1\": \"hello world\", \"prop2\": 1}"})
	testReflectValueEqualSlice(t, values, []GoodStruct{{"hello world", 1, ""}})

	// Should error when ref to component is bad
	txMetadata = TransactionMetadata{}
	paramSchema = *(spec.RefProperty("#/components/somethingodd/GoodStruct"))
	paramsMetadata = ParameterMetadata{}
	paramsMetadata.Name = "some param"
	paramsMetadata.Schema = paramSchema
	txMetadata.Parameters = make([]ParameterMetadata, 1)
	txMetadata.Parameters[0] = paramsMetadata

	components = ComponentMetadata{}
	components.Schemas = make(map[string]ObjectMetadata)
	components.Schemas["GoodStruct"] = goodStructMetadata

	values, err = getArgs(cf, reflect.ValueOf(ctx), &txMetadata, &components, []string{"{\"Prop1\": \"hello world\", \"prop2\": 1}"})
	assert.Nil(t, values, "should not return values when parameter data bad")
	assert.Contains(t, err.Error(), "Invalid schema for parameter \"some param\"", "should error when schema bad")

	// Should handle ref to component and not matching schema in component
	setContractFunctionParams(&cf, nil, []reflect.Type{
		reflect.TypeOf(GoodStruct{}),
	})

	txMetadata = TransactionMetadata{}
	paramSchema = *(spec.RefProperty("#/components/schemas/GoodStruct"))
	paramsMetadata = ParameterMetadata{}
	paramsMetadata.Schema = paramSchema
	txMetadata.Parameters = make([]ParameterMetadata, 1)
	txMetadata.Parameters[0] = paramsMetadata

	components = ComponentMetadata{}
	components.Schemas = make(map[string]ObjectMetadata)
	customMetadata := ObjectMetadata{
		Properties:           make(map[string]spec.Schema),
		Required:             []string{"Prop1", "prop2"},
		AdditionalProperties: false,
	}
	prop2Schema := spec.Int64Property()
	max := float64(0)
	prop2Schema.Maximum = &max
	customMetadata.Properties["Prop1"] = goodStructMetadata.Properties["Prop1"]
	customMetadata.Properties["prop2"] = *prop2Schema
	components.Schemas["GoodStruct"] = customMetadata

	values, err = getArgs(cf, reflect.ValueOf(ctx), &txMetadata, &components, []string{"{\"Prop1\": \"hello world\", \"prop2\": 1}"})
	assert.Nil(t, values, "should not return values when parameter data bad")
	assert.Contains(t, err.Error(), "did not match schema", "should error when schema bad")
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
	testHandleResponse(t, stringRefType, true, response, stringMsg, stringMsg, nil)

	// Should return response string and nil for error when one value returned and expecting only string
	response = []reflect.Value{stringValue}
	testHandleResponse(t, stringRefType, false, response, stringMsg, stringMsg, nil)

	// Should return blank string and response error when one value returned and expecting only error
	response = []reflect.Value{errorValue}
	testHandleResponse(t, nil, true, response, "", nil, err)

	// Should return blank string and nil error when response is empty array and expecting no string or error
	response = []reflect.Value{}
	testHandleResponse(t, nil, false, response, "", nil, nil)

	// Should return basic types in string form
	response = []reflect.Value{reflect.ValueOf(1)}
	testHandleResponse(t, intRefType, false, response, "1", 1, nil)

	response = []reflect.Value{reflect.ValueOf(int8(1))}
	testHandleResponse(t, int8RefType, false, response, "1", int8(1), nil)

	response = []reflect.Value{reflect.ValueOf(int16(1))}
	testHandleResponse(t, int16RefType, false, response, "1", int16(1), nil)

	response = []reflect.Value{reflect.ValueOf(int32(1))}
	testHandleResponse(t, int32RefType, false, response, "1", int32(1), nil)

	response = []reflect.Value{reflect.ValueOf(int64(1))}
	testHandleResponse(t, int64RefType, false, response, "1", int64(1), nil)

	response = []reflect.Value{reflect.ValueOf(uint(1))}
	testHandleResponse(t, uintRefType, false, response, "1", uint(1), nil)

	response = []reflect.Value{reflect.ValueOf(uint8(1))}
	testHandleResponse(t, uint8RefType, false, response, "1", uint8(1), nil)

	response = []reflect.Value{reflect.ValueOf(uint16(1))}
	testHandleResponse(t, uint16RefType, false, response, "1", uint16(1), nil)

	response = []reflect.Value{reflect.ValueOf(uint32(1))}
	testHandleResponse(t, uint32RefType, false, response, "1", uint32(1), nil)

	response = []reflect.Value{reflect.ValueOf(uint64(1))}
	testHandleResponse(t, uint64RefType, false, response, "1", uint64(1), nil)

	response = []reflect.Value{reflect.ValueOf(float32(1.1))}
	testHandleResponse(t, float32RefType, false, response, "1.1", float32(1.1), nil)

	response = []reflect.Value{reflect.ValueOf(float64(1.1))}
	testHandleResponse(t, float64RefType, false, response, "1.1", float64(1.1), nil)

	// Should handle interface return when interface is not to be JSON marshalled
	mc := myContract{}
	mcFuncType := reflect.TypeOf(mc.AfterTransactionWithInterface)

	response = []reflect.Value{reflect.ValueOf(float64(1.1))}
	testHandleResponse(t, mcFuncType.Out(0), false, response, "1.1", float64(1.1), nil)

	// Should handle interface return when interface is returned as nil
	response = []reflect.Value{reflect.ValueOf(nil)}
	testHandleResponse(t, mcFuncType.Out(0), false, response, "", nil, nil)

	// Should handle interface return when interface is returned as nil but of type
	var strPtr *string
	response = []reflect.Value{reflect.ValueOf(strPtr)}
	testHandleResponse(t, mcFuncType.Out(0), false, response, "", strPtr, nil)

	// Should return array responses as JSON strings
	intArray := [4]int{1, 2, 3, 4}
	response = []reflect.Value{reflect.ValueOf(intArray)}
	testHandleResponse(t, reflect.TypeOf(intArray), false, response, "[1,2,3,4]", intArray, nil)

	intMdArray := [4][]int{{1}, {2, 3}, {4, 5, 6}, {7}}
	response = []reflect.Value{reflect.ValueOf(intMdArray)}
	testHandleResponse(t, reflect.TypeOf(intMdArray), false, response, "[[1],[2,3],[4,5,6],[7]]", intMdArray, nil)

	// Should return map responses as JSON strings
	stringIntMap := map[string]int{
		"bob":  1,
		"fred": 10,
	}
	response = []reflect.Value{reflect.ValueOf(stringIntMap)}
	testHandleResponse(t, reflect.TypeOf(stringIntMap), false, response, "{\"bob\":1,\"fred\":10}", stringIntMap, nil)
	stringMapIntMap := map[string]map[string]int{
		"bob": map[string]int{
			"fred": 10,
		},
	}
	response = []reflect.Value{reflect.ValueOf(stringMapIntMap)}
	testHandleResponse(t, reflect.TypeOf(stringMapIntMap), false, response, "{\"bob\":{\"fred\":10}}", stringMapIntMap, nil)

	// Should return a json object for a struct
	myStruct := GoodStruct{"Hello World", 100, "Goodbye"}
	response = []reflect.Value{reflect.ValueOf(myStruct)}
	testHandleResponse(t, reflect.TypeOf(myStruct), false, response, "{\"Prop1\":\"Hello World\",\"prop2\":100}", myStruct, nil)

	// Should return a json object for a pointer to struct
	myPtrStruct := new(GoodStruct)
	myPtrStruct.Prop1 = "Hello World"
	myPtrStruct.Prop2 = 100
	myPtrStruct.shouldIgnore = "Goodbye"
	response = []reflect.Value{reflect.ValueOf(myPtrStruct)}
	testHandleResponse(t, reflect.TypeOf(myPtrStruct), false, response, "{\"Prop1\":\"Hello World\",\"prop2\":100}", myPtrStruct, nil)

	// Should return slice responses as JSON strings
	intSlice := []int{1, 2, 3, 4}
	response = []reflect.Value{reflect.ValueOf(intSlice)}
	testHandleResponse(t, reflect.TypeOf(intSlice), false, response, "[1,2,3,4]", intSlice, nil)

	intMdSlice := [][]int{{1}, {2, 3}, {4, 5, 6}, {7}}
	response = []reflect.Value{reflect.ValueOf(intMdSlice)}
	testHandleResponse(t, reflect.TypeOf(intMdSlice), false, response, "[[1],[2,3],[4,5,6],[7]]", intMdSlice, nil)
}

func TestCall(t *testing.T) {
	var expectedStr string
	var expectedErr error
	var actualStr string
	var actualValue interface{}
	var actualErr error

	cf := new(contractFunction)
	ctx := new(TransactionContext)
	mc := myContract{}

	// Should call function of contract function with correct params and return expected values for context and param function
	cf = newContractFunctionFromFunc(mc.UsesContext, basicContextPtrType)

	expectedStr, expectedErr = mc.UsesContext(ctx, standardAssetID, standardValue)
	actualStr, actualValue, actualErr = cf.call(reflect.ValueOf(ctx), nil, nil, standardAssetID, standardValue)

	assert.Equal(t, expectedStr, actualStr, "Should have returned string as a regular call to UsesContext would")
	assert.Equal(t, expectedStr, actualValue, "Should have returned the string value returned by UsesContext as actual value")
	assert.Equal(t, expectedErr, actualErr, "Should have returned error as a regular call to UsesContext would")

	// Should call function of contract function with correct params and return expected values for function returning nothing
	cf = newContractFunctionFromFunc(mc.ReturnsNothing, basicContextPtrType)

	actualStr, actualValue, actualErr = cf.call(reflect.ValueOf(ctx), nil, nil)

	assert.Equal(t, "", actualStr, "Should have returned blank string")
	assert.Nil(t, actualValue, "should have returned nil when no value defined to return")
	assert.Nil(t, actualErr, "Should have returned nil")

	// Should call function of contract function with correct params and return expected values for function returning string
	cf = newContractFunctionFromFunc(mc.ReturnsString, basicContextPtrType)

	expectedStr = mc.ReturnsString()

	actualStr, actualValue, actualErr = cf.call(reflect.ValueOf(ctx), nil, nil)

	assert.Equal(t, expectedStr, actualStr, "Should have returned string as regular call to ReturnsString would")
	assert.Equal(t, expectedStr, actualValue, "Should have returned string that ReturnsString returns as the actual value")
	assert.Nil(t, actualErr, "Should have returned nil")

	// Should call function of contract function with correct params and return expected values for function returning string
	cf = newContractFunctionFromFunc(mc.UsesBasics, basicContextPtrType)

	expectedStr = mc.UsesBasics("some string", true, 123, 45, 6789, 101112, 131415, 123, 45, 6789, 101112, 131415, 1.1, 2.2, 65, 66)

	actualStr, actualValue, actualErr = cf.call(reflect.ValueOf(ctx), nil, nil, "some string", "true", "123", "45", "6789", "101112", "131415", "123", "45", "6789", "101112", "131415", "1.1", "2.2", "65", "66")

	assert.Equal(t, expectedStr, actualStr, "Should have returned string as regular call to UsesBasics would")
	assert.Equal(t, expectedStr, actualValue, "Should have returned string that UsesBasics returns as the actual value")
	assert.Nil(t, actualErr, "Should have returned nil")

	// Should call function of contract function with correct params and return expected values for function returning error
	cf = newContractFunctionFromFunc(mc.ReturnsError, basicContextPtrType)

	expectedErr = mc.ReturnsError()

	actualStr, actualValue, actualErr = cf.call(reflect.ValueOf(ctx), nil, nil)

	assert.Equal(t, "", actualStr, "Should have returned blank string")
	assert.Nil(t, actualValue, "should be nil as ReturnsError returns no success type")
	assert.EqualError(t, actualErr, expectedErr.Error(), "Should have returned error as a regular call to ReturnsError would")

	// Should return error when getArgs returns an error
	cf = newContractFunctionFromFunc(mc.UsesArray, basicContextPtrType)

	expectedErr = errors.New("Value [1] was not passed in expected format [1]string")

	actualStr, actualValue, actualErr = cf.call(reflect.ValueOf(ctx), nil, nil, "[1]")

	assert.Equal(t, "", actualStr, "Should have returned blank string")
	assert.Nil(t, nil, "Should have returned nil as getArgs causes an error")
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
