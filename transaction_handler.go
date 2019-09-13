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
	"fmt"
	"reflect"
)

// UndefinedInterface the type of nil passed to an after transaction when
// the contract function called as part of the transaction does not specify
// a success return type or its return type is interface{} and value nil
type UndefinedInterface struct{}

type transactionHandlerType int

const (
	before transactionHandlerType = iota + 1
	unknown
	after
)

func (tht transactionHandlerType) String() string {
	switch tht {
	case before:
		return "Before"
	case after:
		return "After"
	case unknown:
		return "Unknown"
	default:
		panic("Invalid value")
	}
}

type transactionHandler struct {
	contractFunction
	handlesType transactionHandlerType
}

func (th transactionHandler) call(ctx reflect.Value, data interface{}) (string, interface{}, error) {
	values := []reflect.Value{}

	if th.params.context != nil {
		values = append(values, ctx)
	}

	if th.handlesType == after && len(th.params.fields) == 1 {
		if data == nil {
			values = append(values, reflect.Zero(reflect.TypeOf(new(UndefinedInterface))))
		} else {
			values = append(values, reflect.ValueOf(data))
		}
	}

	someResp := th.function.Call(values)

	return handleContractFunctionResponse(someResp, th.contractFunction)
}

func newTransactionHandler(fn interface{}, contextHandlerType reflect.Type, handlesType transactionHandlerType) *transactionHandler {
	cf := newContractFunctionFromFunc(fn, contextHandlerType)

	if handlesType != after && len(cf.params.fields) > 0 {
		logger.Error()
		panic(fmt.Sprintf("%s transactions may not take any params other than the transaction context", handlesType.String()))
	} else if handlesType == after && len(cf.params.fields) > 1 {
		panic("After transactions must take at most one non-context param")
	} else if handlesType == after && len(cf.params.fields) == 1 && cf.params.fields[0].Kind() != reflect.Interface {
		panic("After transaction must take type interface{} as their only non-context param")
	}

	th := transactionHandler{
		*cf,
		handlesType,
	}

	return &th
}
