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
	"testing"

	"github.com/stretchr/testify/assert"
)

// ================================
// Tests
// ================================

func TestSetUnknownTransaction(t *testing.T) {
	mc := myContract{}

	// Should set unknown transaction
	mc.SetUnknownTransaction(mc.ReturnsString)
	assert.Equal(t, mc.ReturnsString(), mc.unknownTransaction.(func() string)(), "unknown transaction should have been set to value passed")
}

func TestGetUnknownTransaction(t *testing.T) {
	var mc myContract
	var unknownFn interface{}
	// Should throw an error when unknown transaction not set
	mc = myContract{}

	unknownFn = mc.GetUnknownTransaction()

	assert.Nil(t, unknownFn, "should not return contractFunction when unknown transaction not set")

	// Should return the call value of the stored unknown transaction when set
	mc = myContract{}
	mc.unknownTransaction = mc.ReturnsInt

	unknownFn = mc.GetUnknownTransaction()

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
	// Should throw an error when before transaction not set
	mc = myContract{}

	beforeFn = mc.GetBeforeTransaction()

	assert.Nil(t, beforeFn, "should not return contractFunction when before transaction not set")

	// Should return the call value of the stored before transaction when set
	mc = myContract{}
	mc.beforeTransaction = mc.ReturnsInt

	beforeFn = mc.GetBeforeTransaction()

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
	// Should throw an error when after transaction not set
	mc = myContract{}

	afterFn = mc.GetAfterTransaction()

	assert.Nil(t, afterFn, "should not return contractFunction when after transaction not set")

	// Should return the call value of the stored after transaction when set
	mc = myContract{}
	mc.afterTransaction = mc.ReturnsInt

	afterFn = mc.GetAfterTransaction()

	assert.Equal(t, mc.ReturnsInt(), afterFn.(func() int)(), "function returned should be same value as set for after transaction")
}

func TestSetVersion(t *testing.T) {
	c := Contract{}
	c.SetVersion("some version")

	assert.Equal(t, "some version", c.version, "should set the version")
}

func TestGetVersion(t *testing.T) {
	c := Contract{}
	c.version = "some version"

	assert.Equal(t, "some version", c.GetVersion(), "should set the version")
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
