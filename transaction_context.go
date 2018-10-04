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
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// TransactionContextInterface defines functions a valid transaction context
// should have. Transaction context's set for contract's to be used in chaincode
// must implement this interface.
type TransactionContextInterface interface {
	// SetStub should provide a way to pass the stub from a chaincode transaction
	// call to the transaction context so that it can be used by contract functions.
	// This is called by Init/Invoke with the stub passed.
	SetStub(shim.ChaincodeStubInterface)
}

// TransactionContext is a basic transaction context to be used in contracts,
// containing minimal required functionality use in contracts as part of
// chaincode. Provides access to the stub and clientIdentity of a transaction.
// If a contract implements the ContractInterface using the Contract struct then
// this is the default transaction context that will be used.
type TransactionContext struct {
	stub shim.ChaincodeStubInterface
}

// SetStub stores the passed stub in the transaction context
func (ctx *TransactionContext) SetStub(stub shim.ChaincodeStubInterface) {
	ctx.stub = stub
}

// GetStub returns the current set stub
func (ctx *TransactionContext) GetStub() shim.ChaincodeStubInterface {
	return ctx.stub
}
