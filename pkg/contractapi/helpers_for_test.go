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
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/xeipuuv/gojsonschema"
)

// ================================
// Helpful vars for testing
// ================================

const standardAssetID = "ABC123"
const standardTxID = "txID"
const standardValue = "100"

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
var interfaceTypeVar = new(interfaceType)

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

var goodStructPropertiesMap = map[string]spec.Schema{
	"Prop1": *stringTypeVar.getSchema(),
	"prop2": *intTypeVar.getSchema(),
}

var goodStructMetadata = ObjectMetadata{
	Properties:           goodStructPropertiesMap,
	Required:             []string{"Prop1", "prop2"},
	AdditionalProperties: false,
}

// ================================
// Helpful test functions
// ================================
func testContractChaincodeContractRepresentsContract(t *testing.T, ccns contractChaincodeContract, contract simpleTestContract) {
	t.Helper()

	assert.Equal(t, len(expectedSimpleContractFuncs), len(ccns.functions), "should only have one function as simpleTestContract")

	assert.Equal(t, ccns.functions["DoSomething"].params, contractFunctionParams{nil, nil}, "should set correct params for contract function")
	assert.Equal(t, ccns.functions["DoSomething"].returns, contractFunctionReturns{stringRefType, true}, "should set correct returns for contract function")

	transactionContextHandler := reflect.ValueOf(contract.GetTransactionContextHandler()).Elem().Type()
	transactionContextPtrHandler := reflect.ValueOf(contract.GetTransactionContextHandler()).Type()

	assert.Equal(t, ccns.transactionContextHandler, transactionContextHandler, "should have correct transaction context set")
	assert.Equal(t, ccns.transactionContextPtrHandler, transactionContextPtrHandler, "should have correct transaction context set")

	ut := contract.GetUnknownTransaction()

	if ut == nil {
		assert.Nil(t, ccns.unknownTransaction, "should be nil when contract has no unknown transaction")
	} else {
		assert.Equal(t, ccns.unknownTransaction, newTransactionHandler(ut, transactionContextPtrHandler, unknown), "should have set correct unknown transaction when set")
	}

	if contract.GetVersion() == "" {
		assert.Equal(t, "latest", ccns.version, "should set correct version when get version blank")
	} else {
		assert.Equal(t, contract.GetVersion(), ccns.version, "should set correct version when get version blank")
	}

	bt := contract.GetBeforeTransaction()

	if bt == nil {
		assert.Nil(t, ccns.beforeTransaction, "should be nil when contract has no before transaction")
	} else {
		assert.Equal(t, ccns.beforeTransaction, newTransactionHandler(bt, transactionContextPtrHandler, before), "should have set correct before transaction when set")
	}

	at := contract.GetAfterTransaction()

	if at == nil {
		assert.Nil(t, ccns.afterTransaction, "should be nil when contract has no after transaction")
	} else {
		assert.Equal(t, ccns.afterTransaction, newTransactionHandler(at, transactionContextPtrHandler, after), "should have set correct after transaction when set")
	}
}

func testMetadata(t *testing.T, ccMetadata ContractChaincodeMetadata, expectedMetadata ContractChaincodeMetadata) {
	t.Helper()

	// Should be valid against schema
	schemaLoader := gojsonschema.NewBytesLoader([]byte(GetJSONSchema()))
	toValidateLoader := gojsonschema.NewGoLoader(ccMetadata)

	schema, err := gojsonschema.NewSchema(schemaLoader)

	if err != nil {
		assert.Fail(t, fmt.Sprintf("Invalid schema from GetJSONSchema: %s", err.Error()), "should have valid metadata schema to test against")
	}

	result, _ := schema.Validate(toValidateLoader)

	if !result.Valid() {
		assert.Fail(t, fmt.Sprintf("Failed testing metadata. Given metadata did not match schema: %s", validateErrorsToString(result.Errors())), "metadata should validate")
	}

	assert.Equal(t, expectedMetadata, ccMetadata, "Should match expected metadata")
}

func createMetadataJSONFile(data []byte, permissions os.FileMode) string {
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)

	folderPath := filepath.Join(exPath, metadataFolder)
	filePath := filepath.Join(folderPath, metadataFile)

	os.MkdirAll(folderPath, os.ModePerm)
	ioutil.WriteFile(filePath, data, permissions)

	return filePath
}

func cleanupMetadataJSONFile() {
	ex, _ := os.Executable()
	exPath := filepath.Dir(ex)

	folderPath := filepath.Join(exPath, metadataFolder)

	os.RemoveAll(folderPath)
}

// ================================
// Helpful structs that aren't contracts
// ================================

type GoodStruct struct {
	Prop1        string
	Prop2        int `json:"prop2"`
	shouldIgnore string
}

type AnotherGoodStruct struct {
	StringProp string     `json:"StringProp"`
	StructProp GoodStruct `json:"StructProp"`
}

type BadStruct struct {
	Prop1 string    `json:"Prop1"`
	Prop2 complex64 `json:"prop2"`
}

// ================================
// Helpful contracts for testing
// ================================

type myContract struct {
	Contract
	called []string
}

func (mc *myContract) logBefore() {
	mc.called = append(mc.called, "Before function called")
}

func (mc *myContract) LogNamed() string {
	mc.called = append(mc.called, "Named function called")
	return "named response"
}

func (mc *myContract) logAfter(data interface{}) {
	mc.called = append(mc.called, fmt.Sprintf("After function called with %v", data))
}

func (mc *myContract) logUnknown() {
	mc.called = append(mc.called, "Unknown function called")
}

func (mc *myContract) BeforeTransaction(ctx *TransactionContext) (string, error) {
	return "some before transaction", errors.New("Some before error")
}

func (mc *myContract) UnknownTransaction(ctx *TransactionContext) (string, error) {
	return "some unknown transaction", errors.New("Some unknown error")
}

func (mc *myContract) AfterTransaction(ctx *TransactionContext) (string, error) {
	return "some after transaction", errors.New("some after error")
}

func (mc *myContract) AfterTransactionWithInterface(ctx *TransactionContext, param0 interface{}) (string, error) {
	return reflect.TypeOf(param0).String(), errors.New("some after with iFace error")
}

func (mc *myContract) CheckContextStub(ctx *TransactionContext) (string, error) {
	if ctx.GetStub().GetTxID() != standardTxID {
		return "", fmt.Errorf("You used a non standard txID [%s]", ctx.GetStub().GetTxID())
	}

	return "Stub as expected", nil
}

func (mc *myContract) UsesContext(ctx *TransactionContext, assetID string, value string) (string, error) {
	if assetID != standardAssetID {
		return "", fmt.Errorf("You used a non standard assetID [%s]", assetID)
	} else if value != standardValue {
		return "", fmt.Errorf("You used a non standard value [%s]", value)
	}

	return "You called a function that uses the ctx", nil
}

func (mc *myContract) NotUsesContext(assetID string, value string) (string, error) {
	if assetID != standardAssetID {
		return "", fmt.Errorf("You used a non standard assetID [%s]", assetID)
	} else if value != standardValue {
		return "", fmt.Errorf("You used a non standard value [%s]", value)
	}

	return "You called a function that does not use the ctx", nil
}

func (mc *myContract) UsesBasics(str string, tf bool, i int, i8 int8, i16 int16, i32 int32, i64 int64, u uint, u8 uint8, u16 uint16, u32 uint32, u64 uint64, f32 float32, f64 float64, byt byte, run rune) string {
	return fmt.Sprintf("You passed %s, %t, %d, %d, %d, %d, %d, %d, %d, %d, %d, %d, %f, %f, %d, %d", str, tf, i, i8, i16, i32, i64, u, u8, u16, u32, u64, f32, f64, byt, run)
}

func (mc *myContract) UsesArray(args [1]string) {}

func (mc *myContract) UsesSlices(args []string) {}

func (mc *myContract) ReturnsStringAndError(shouldError string) (string, error) {
	if shouldError == "true" {
		return "", errors.New("An error as requested")
	}

	return "A string as requested", nil
}

func (mc *myContract) ReturnsString() string {
	return "Some string"
}

func (mc *myContract) ReturnsInt() int {
	return 1
}

func (mc *myContract) ReturnsArray() [1]uint {
	return [1]uint{1}
}

func (mc *myContract) ReturnsSlice() []uint {
	return []uint{1, 2, 3}
}

func (mc *myContract) ReturnsError() error {
	return errors.New("Some error")
}

func (mc *myContract) ReturnsNil() error {
	return nil
}

func (mc *myContract) ReturnsNothing() {}

type simpleTestContract struct {
	Contract
}

var expectedSimpleContractFuncs = []string{"DoSomething"}

func (sc *simpleTestContract) DoSomething() (string, error) {
	return "Done something", nil
}

type customContext struct {
	TransactionContext
	someVal string
}

func (cc *customContext) ReturnString() string {
	return "I am custom context"
}

type simpleTestContractWithCustomContext struct {
	Contract
}

func (sc *simpleTestContractWithCustomContext) SetValInCustomContext(ctx *customContext) {
	_, params := ctx.GetStub().GetFunctionAndParameters()
	ctx.someVal = params[0]
}

func (sc *simpleTestContractWithCustomContext) GetValInCustomContext(ctx *customContext) (string, error) {
	if ctx.someVal != standardValue {
		return "", errors.New("I wanted a standard value")
	}

	return ctx.someVal, nil
}

func (sc *simpleTestContractWithCustomContext) CheckCustomContext(ctx *customContext) string {
	return ctx.ReturnString()
}

type badContract struct {
	ContractInterface
}

func (sc *badContract) SetAfterTransaction(fn interface{}) {}

func (sc *badContract) TakesBadType(cplx complex64)     {}
func (sc *badContract) TakesBadArray(cplx [1]complex64) {}
func (sc *badContract) TakesBadSlice(cplx []complex64)  {}

func (sc *badContract) TakesContextBadly(str string, ctx *TransactionContext) {}

func (sc *badContract) ReturnsBadType() complex64 {
	return 1
}

func (sc *badContract) ReturnsBadArray() [1]complex64 {
	return [1]complex64{1}
}

func (sc *badContract) ReturnsBadSlice() []complex64 {
	return []complex64{1}
}

func (sc *badContract) ReturnsBadTypeAndError() (complex64, error) {
	return 1, nil
}

func (sc *badContract) ReturnsStringAndInt() (string, int) {
	return "", 1
}
