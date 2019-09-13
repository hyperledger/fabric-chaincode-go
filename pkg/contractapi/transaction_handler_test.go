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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ================================
// Helpers
// ================================

// ================================
// Tests
// ================================

func TestString(t *testing.T) {
	// Should return string version of the integer
	assert.Equal(t, "Before", before.String(), "should output Before for before type")
	assert.Equal(t, "After", after.String(), "should output Before for before type")
	assert.Equal(t, "Unknown", unknown.String(), "should output Before for before type")

	// should panic when invalid
	assert.PanicsWithValue(t, "Invalid value", func() { transactionHandlerType(after + 1).String() }, "should panic with value")
}

func TestNewTransactionHandler(t *testing.T) {
	var th *transactionHandler
	var cf *contractFunction

	mc := myContract{}

	// Should panic when txn passed does not match structure for a before txn
	assert.PanicsWithValue(t, before.String()+" transactions may not take any params other than the transaction context", func() { newTransactionHandler(mc.UsesBasics, basicContextPtrType, before) }, "should error when before does not match expected structure")

	// Should panic when txn passed does not match structure for an unknown txn
	assert.PanicsWithValue(t, unknown.String()+" transactions may not take any params other than the transaction context", func() { newTransactionHandler(mc.UsesBasics, basicContextPtrType, unknown) }, "should error when unknown does not match expected structure")

	// Should panic when txn passed does not match structure for an after txn as too many params
	assert.PanicsWithValue(t, "After transactions must take at most one non-context param", func() { newTransactionHandler(mc.UsesBasics, basicContextPtrType, after) }, "should error when after does not match expected structure, too few params")

	// Should panic when txn passed does not match structure for an after txn as only param is not interface{}
	assert.PanicsWithValue(t, "After transaction must take type interface{} as their only non-context param", func() { newTransactionHandler(mc.UsesSlices, basicContextPtrType, after) }, "should error when after does not match expected structure, wrong param type")

	// Should create a txn handler for a before transaction
	th = newTransactionHandler(mc.BeforeTransaction, basicContextPtrType, before)
	cf = newContractFunctionFromFunc(mc.BeforeTransaction, basicContextPtrType)
	assert.Equal(t, before, th.handlesType, "should create a txn handler for a before txn")
	assert.Equal(t, th.params, cf.params, "should create a txn handler for a before txn")
	assert.Equal(t, th.returns, cf.returns, "should create a txn handler for a before txn")

	// Should create a txn handler for an unknown transaction
	th = newTransactionHandler(mc.UnknownTransaction, basicContextPtrType, unknown)
	cf = newContractFunctionFromFunc(mc.UnknownTransaction, basicContextPtrType)
	assert.Equal(t, unknown, th.handlesType, "should create a txn handler for an unknown txn")
	assert.Equal(t, th.params, cf.params, "should create a txn handler for an unknown txn")
	assert.Equal(t, th.returns, cf.returns, "should create a txn handler for an unknown txn")

	// Should create a txn handler for an after transaction
	th = newTransactionHandler(mc.AfterTransaction, basicContextPtrType, after)
	cf = newContractFunctionFromFunc(mc.AfterTransaction, basicContextPtrType)
	assert.Equal(t, after, th.handlesType, "should create a txn handler for an after txn")
	assert.Equal(t, th.params, cf.params, "should create a txn handler for an after txn")
	assert.Equal(t, th.returns, cf.returns, "should create a txn handler for an after txn")

	// Should create a txn handler for an after transaction with param
	th = newTransactionHandler(mc.AfterTransactionWithInterface, basicContextPtrType, after)
	cf = newContractFunctionFromFunc(mc.AfterTransactionWithInterface, basicContextPtrType)
	assert.Equal(t, after, th.handlesType, "should create a txn handler for an after txn")
	assert.Equal(t, th.params, cf.params, "should create a txn handler for an after txn")
	assert.Equal(t, th.returns, cf.returns, "should create a txn handler for an after txn")
}

func TestTHCall(t *testing.T) {
	var expectedValue interface{}
	var expectedStr string
	var expectedErr error
	var actualStr string
	var actualValue interface{}
	var actualErr error

	th := new(transactionHandler)
	ctx := new(TransactionContext)
	mc := myContract{}

	// Should call before transaction type
	th = newTransactionHandler(mc.BeforeTransaction, basicContextPtrType, before)
	expectedStr, expectedErr = mc.BeforeTransaction(new(TransactionContext))
	actualStr, actualValue, actualErr = th.call(reflect.ValueOf(ctx), nil)

	assert.Equal(t, expectedStr, actualStr, "Should have returned string as a regular call to BeforeTransaction would")
	assert.Equal(t, expectedStr, actualValue, "Should have returned the string value returned by BeforeTransaction as actual value")
	assert.Equal(t, expectedErr, actualErr, "Should have returned error as a regular call to BeforeTransaction would")

	// Should call unknown transaction type
	th = newTransactionHandler(mc.UnknownTransaction, basicContextPtrType, unknown)
	expectedStr, expectedErr = mc.UnknownTransaction(new(TransactionContext))
	actualStr, actualValue, actualErr = th.call(reflect.ValueOf(ctx), nil)

	assert.Equal(t, expectedStr, actualStr, "Should have returned string as a regular call to UnknownTransaction would")
	assert.Equal(t, expectedStr, actualValue, "Should have returned the string value returned by UnknownTransaction as actual value")
	assert.Equal(t, expectedErr, actualErr, "Should have returned error as a regular call to UnknownTransaction would")

	// Should call after transaction type
	th = newTransactionHandler(mc.AfterTransaction, basicContextPtrType, after)
	expectedStr, expectedErr = mc.AfterTransaction(new(TransactionContext))
	actualStr, actualValue, actualErr = th.call(reflect.ValueOf(ctx), nil)

	assert.Equal(t, expectedStr, actualStr, "Should have returned string as a regular call to AfterTransaction would")
	assert.Equal(t, expectedStr, actualValue, "Should have returned the string value returned by AfterTransaction as actual value")
	assert.Equal(t, expectedErr, actualErr, "Should have returned error as a regular call to AfterTransaction would")

	// Should call after transaction type with interface
	th = newTransactionHandler(mc.AfterTransactionWithInterface, basicContextPtrType, after)
	expectedValue, expectedErr = mc.AfterTransactionWithInterface(new(TransactionContext), "some value")
	actualStr, actualValue, actualErr = th.call(reflect.ValueOf(ctx), "some value")

	assert.Equal(t, expectedValue, actualStr, "Should have returned string as a regular call to AfterTransactionWithInterface would")
	assert.Equal(t, expectedValue, actualValue, "Should have returned the string value returned by AfterTransactionWithInterface as actual value")
	assert.Equal(t, expectedErr, actualErr, "Should have returned error as a regular call to AfterTransactionWithInterface would")

	// Should handle when after called with nil because no success type
	th = newTransactionHandler(mc.AfterTransactionWithInterface, basicContextPtrType, after)
	expectedValue, expectedErr = mc.AfterTransactionWithInterface(new(TransactionContext), (*UndefinedInterface)(nil))
	actualStr, actualValue, actualErr = th.call(reflect.ValueOf(ctx), nil)

	assert.Equal(t, "*contractapi.UndefinedInterface", actualStr, "Should have returned string as a regular call to AfterTransactionWithInterface would")
	assert.Equal(t, expectedValue, actualValue, "Should have returned the string value returned by AfterTransactionWithInterface as actual value")
	assert.Equal(t, expectedErr, actualErr, "Should have returned error as a regular call to AfterTransactionWithInterface would")

	// Should handle when after called with nil but with success type
	th = newTransactionHandler(mc.AfterTransactionWithInterface, basicContextPtrType, after)
	expectedValue, expectedErr = mc.AfterTransactionWithInterface(new(TransactionContext), (*string)(nil))
	actualStr, actualValue, actualErr = th.call(reflect.ValueOf(ctx), (*string)(nil))

	assert.Equal(t, "*string", actualStr, "Should have returned string as a regular call to AfterTransactionWithInterface would")
	assert.Equal(t, expectedValue, actualValue, "Should have returned the string value returned by AfterTransactionWithInterface as actual value")
	assert.Equal(t, expectedErr, actualErr, "Should have returned error as a regular call to AfterTransactionWithInterface would")
}
