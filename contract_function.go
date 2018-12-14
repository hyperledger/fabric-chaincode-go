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
	"fmt"
	"reflect"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
)

type contractFunctionParams struct {
	context reflect.Type
	fields  []reflect.Type
}

type contractFunctionReturns struct {
	success reflect.Type
	error   bool
}

type contractFunction struct {
	function reflect.Value
	params   contractFunctionParams
	returns  contractFunctionReturns
}

func (cf contractFunction) call(ctx reflect.Value, supplementaryMetadata TransactionMetadata, params ...string) (string, error) {
	values, err := getArgs(cf, ctx, supplementaryMetadata, params)

	if err != nil {
		return "", err
	}

	someResp := cf.function.Call(values)

	return handleContractFunctionResponse(someResp, cf)
}

func (cf contractFunction) exists() bool {
	if cf.function.IsValid() && !cf.function.IsNil() {
		return true
	}
	return false
}

func arrayOfValidType(array reflect.Value) error {
	if array.Len() < 1 {
		return fmt.Errorf("Arrays must have length greater than 0")
	}

	if array.Index(0).Kind() == reflect.Array {
		return arrayOfValidType(array.Index(0))
	} else if array.Index(0).Kind() == reflect.Slice {
		return sliceOfValidType(array.Index(0))
	} else if _, ok := basicTypes[array.Index(0).Kind()]; !ok {
		return fmt.Errorf("Arrays can only have base types %s. Array has basic type %s", listBasicTypes(), array.Index(0).Kind().String())
	}
	return nil
}

func sliceOfValidType(slice reflect.Value) error {
	if slice.Len() < 1 {
		slice = reflect.MakeSlice(slice.Type(), 1, 10)
	}
	if slice.Index(0).Kind() == reflect.Slice {
		return sliceOfValidType(slice.Index(0))
	} else if slice.Index(0).Kind() == reflect.Array {
		return arrayOfValidType(slice.Index(0))
	} else if _, ok := basicTypes[slice.Index(0).Kind()]; !ok {
		return fmt.Errorf("Slices can only have base types %s. Slice has basic type %s", listBasicTypes(), slice.Index(0).Kind().String())
	}
	return nil
}

func typeIsValid(t reflect.Type, additionalTypes []reflect.Type) error {
	if t.Kind() == reflect.Array {
		array := reflect.New(t).Elem()

		return arrayOfValidType(array)

	} else if t.Kind() == reflect.Slice {
		slice := reflect.MakeSlice(t, 1, 1)

		return sliceOfValidType(slice)

	} else if _, ok := basicTypes[t.Kind()]; !ok {
		if len(additionalTypes) > 0 {
			if typeInSlice(t, additionalTypes) {
				return nil
			}

			additionalTypesString := []string{}

			for _, el := range additionalTypes {
				additionalTypesString = append(additionalTypesString, el.String())
			}

			return fmt.Errorf("Type %s is not valid. Expected one of the basic types %s, an array/slice of these, or one of these additional types %s", t.String(), listBasicTypes(), sliceAsCommaSentence(additionalTypesString))
		}

		return fmt.Errorf("Type %s is not valid. Expected one of the basic types %s or an array/slice of these", t.String(), listBasicTypes())

	}
	return nil
}

func method2ContractFunctionParams(typeMethod reflect.Method, contextHandlerType reflect.Type) (contractFunctionParams, error) {
	myContractFnParams := contractFunctionParams{}

	usesCtx := (reflect.Type)(nil)

	numIn := typeMethod.Type.NumIn()

	startIndex := 1
	methodName := typeMethod.Name

	if methodName == "" {
		startIndex = 0
		methodName = "Function"
	}

	for i := startIndex; i < numIn; i++ {
		inType := typeMethod.Type.In(i)

		typeError := typeIsValid(inType, []reflect.Type{contextHandlerType})

		if typeError != nil {
			return contractFunctionParams{}, fmt.Errorf("%s contains invalid parameter type. %s", methodName, typeError.Error())
		} else if i != startIndex && inType == contextHandlerType {
			return contractFunctionParams{}, fmt.Errorf("Functions requiring the TransactionContext must require it as the first parameter. %s takes it in as parameter %d", methodName, i-startIndex)
		} else if inType == contextHandlerType {
			usesCtx = contextHandlerType
		} else {
			myContractFnParams.fields = append(myContractFnParams.fields, inType)
		}
	}

	myContractFnParams.context = usesCtx
	return myContractFnParams, nil
}

func method2ContractFunctionReturns(typeMethod reflect.Method) (contractFunctionReturns, error) {
	numOut := typeMethod.Type.NumOut()

	methodName := typeMethod.Name

	if methodName == "" {
		methodName = "Function"
	}

	if numOut > 2 {
		return contractFunctionReturns{}, fmt.Errorf("Functions may only return a maximum of two values. %s returns %d", methodName, numOut)
	} else if numOut == 1 {
		outType := typeMethod.Type.Out(0)

		errorType := reflect.TypeOf((*error)(nil)).Elem()

		typeError := typeIsValid(outType, []reflect.Type{errorType})

		if typeError != nil {
			return contractFunctionReturns{}, fmt.Errorf("%s contains invalid single return type. %s", methodName, typeError.Error())
		} else if outType == errorType {
			return contractFunctionReturns{nil, true}, nil
		}
		return contractFunctionReturns{outType, false}, nil
	} else if numOut == 2 {
		firstOut := typeMethod.Type.Out(0)
		secondOut := typeMethod.Type.Out(1)

		firstTypeError := typeIsValid(firstOut, []reflect.Type{})
		if firstTypeError != nil {
			return contractFunctionReturns{}, fmt.Errorf("%s contains invalid first return type. %s", methodName, firstTypeError.Error())
		} else if secondOut.String() != "error" {
			return contractFunctionReturns{}, fmt.Errorf("%s contains invalid second return type. Type %s is not valid. Expected error", methodName, secondOut.String())
		}
		return contractFunctionReturns{firstOut, true}, nil
	}
	return contractFunctionReturns{nil, false}, nil
}

func parseMethod(typeMethod reflect.Method, contextHandlerType reflect.Type) (contractFunctionParams, contractFunctionReturns, error) {
	myContractFnParams, err := method2ContractFunctionParams(typeMethod, contextHandlerType)

	if err != nil {
		return contractFunctionParams{}, contractFunctionReturns{}, err
	}

	myContractFnReturns, err := method2ContractFunctionReturns(typeMethod)

	if err != nil {
		return contractFunctionParams{}, contractFunctionReturns{}, err
	}

	return myContractFnParams, myContractFnReturns, nil
}

func newContractFunction(fnValue reflect.Value, paramDetails contractFunctionParams, returnDetails contractFunctionReturns) *contractFunction {
	scf := contractFunction{}
	scf.function = fnValue
	scf.params = paramDetails
	scf.returns = returnDetails

	return &scf
}

func newContractFunctionFromFunc(fn interface{}, contextHandlerType reflect.Type) *contractFunction {
	fnType := reflect.TypeOf(fn)
	fnValue := reflect.ValueOf(fn)

	if fnType.Kind() != reflect.Func {
		panic(fmt.Sprintf("Cannot create new contract function from %s. Can only use func", fnType.Kind()))
	}

	myMethod := reflect.Method{}
	myMethod.Func = fnValue
	myMethod.Type = fnType

	paramDetails, returnDetails, err := parseMethod(myMethod, contextHandlerType)

	if err != nil {
		panic(err.Error())
	}

	return newContractFunction(fnValue, paramDetails, returnDetails)
}

func newContractFunctionFromReflect(typeMethod reflect.Method, valueMethod reflect.Value, contextHandlerType reflect.Type) *contractFunction {
	paramDetails, returnDetails, err := parseMethod(typeMethod, contextHandlerType)

	if err != nil {
		panic(err.Error())
	}

	return newContractFunction(valueMethod, paramDetails, returnDetails)
}

func createArrayOrSlice(param string, objType reflect.Type) (reflect.Value, error) {
	obj := reflect.New(objType)

	err := json.Unmarshal([]byte(param), obj.Interface())

	if err != nil {
		return reflect.Value{}, fmt.Errorf("Value %s was not passed in expected format %s", param, objType.String())
	}

	return obj.Elem(), nil
}

func getArgs(fn contractFunction, ctx reflect.Value, supplementaryMetadata TransactionMetadata, params []string) ([]reflect.Value, error) {
	var shouldValidate bool

	if !reflect.DeepEqual(supplementaryMetadata, TransactionMetadata{}) {
		shouldValidate = true
	}

	values := []reflect.Value{}

	numParams := len(fn.params.fields)

	if fn.params.context != nil {
		values = append(values, ctx)
	}

	for i := 0; i < numParams; i++ {
		inParamRange := true

		if i >= len(params) {
			inParamRange = false
			params = append(params, "")
		}

		fieldType := fn.params.fields[i]

		var converted reflect.Value
		var err error
		if fieldType.Kind() == reflect.Array || fieldType.Kind() == reflect.Slice {
			if !inParamRange {
				params[i] = "[]"
			}

			converted, err = createArrayOrSlice(params[i], fieldType)

			if err != nil {
				return nil, err
			}

		} else {
			converted, err = basicTypes[fieldType.Kind()].convert(params[i])

			if err != nil {
				return nil, fmt.Errorf("Param %s could not be converted to type %s", params[i], fieldType.String())
			}
		}

		if shouldValidate {
			err := validate.AgainstSchema(&supplementaryMetadata.Parameters[i].Schema, converted.Interface(), strfmt.Default)

			if err != nil {
				return nil, fmt.Errorf("Value passed for parameter \"%s\" did not match schema: %s", supplementaryMetadata.Parameters[i].Name, err)
			}
		}

		values = append(values, converted)
	}

	return values, nil
}

func handleContractFunctionResponse(response []reflect.Value, function contractFunction) (string, error) {
	expectedLength := 0

	returnsSuccess := function.returns.success != nil

	if returnsSuccess && function.returns.error {
		expectedLength = 2
	} else if returnsSuccess || function.returns.error {
		expectedLength = 1
	}

	if len(response) == expectedLength {

		var successResponse reflect.Value
		var errorResponse reflect.Value

		if returnsSuccess && function.returns.error {
			successResponse = response[0]
			errorResponse = response[1]
		} else if returnsSuccess {
			successResponse = response[0]
		} else if function.returns.error {
			errorResponse = response[0]
		}

		var successString string
		var errorError error

		if successResponse.IsValid() {
			if function.returns.success.Kind() == reflect.Array || function.returns.success.Kind() == reflect.Slice {
				bytes, _ := json.Marshal(successResponse.Interface())
				successString = string(bytes)
			} else {
				successString = fmt.Sprint(successResponse.Interface())
			}
		}

		if errorResponse.IsValid() && !errorResponse.IsNil() {
			errorError = errorResponse.Interface().(error)
		}

		return successString, errorError
	}

	panic("Response does not match expected return for given function.")
}
