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
)

type myContract struct {
	Contract
	called []string
}

func (mc *myContract) logBefore() {
	mc.called = append(mc.called, "Before function called")
}

func (mc *myContract) LogNamed() {
	mc.called = append(mc.called, "Named function called")
}

func (mc *myContract) logAfter() {
	mc.called = append(mc.called, "After function called")
}

func (mc *myContract) logUnknown() {
	mc.called = append(mc.called, "Unknown function called")
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

func (sc *simpleTestContractWithCustomContext) SetValInCustomContext(ctx *customContext, newVal string) {
	ctx.someVal = newVal
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

func (sc *badContract) ReturnsWrongOrder() (error, string) {
	return nil, ""
}

func (sc *badContract) ReturnsBadTypeAndError() (complex64, error) {
	return 1, nil
}

func (sc *badContract) ReturnsStringAndInt() (string, int) {
	return "", 1
}

func (sc *badContract) ReturnsStringErrorAndInt() (string, error, int) {
	return "", nil, 1
}
