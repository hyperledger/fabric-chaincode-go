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
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

type contractChaincodeContract struct {
	functions                    map[string]*contractFunction
	unknownTransaction           *contractFunction
	beforeTransaction            *contractFunction
	afterTransaction             *contractFunction
	transactionContextHandler    reflect.Type
	transactionContextPtrHandler reflect.Type
}

// ContractChaincode a struct to meet the chaincode interface and provide routing of calls to contracts
type ContractChaincode struct {
	contracts map[string]contractChaincodeContract
}

// SystemContractName the name of the system smart contract
const SystemContractName = "org.hyperledger.fabric"

// CreateNewChaincode creates a new chaincode using contracts passed. The function parses each
// of the passed functions and stores details about their make-up to be used by the chaincode.
// Public functions of the contracts are stored an are made callable in the chaincode. The function
// will panic if contracts are invalid e.g. public functions take in illegal types. If no panic occurs
// the a new chaincode handling the contracts is started in the shim. A system contract is added to the
// chaincode which provides functionality for getting the metadata of the chaincode. The generated
// metadata is a JSON formatted MetadataContractChaincode containing each contract as a name and details
// of the public functions. The names for parameters do not match those used in the functions instead they are
// recorded as param0, param1, ..., paramN. If there exists a file contract-metadata/metadata.json then this
// will overwrite the generated metadata. The contents of this file must validate against the schema.
func CreateNewChaincode(contracts ...ContractInterface) error {
	return shim.Start(convertC2CC(contracts...))
}

// Init is called during Instantiate transaction after the chaincode container
// has been established for the first time, passes off details of the request to Invoke
// for handling the request if a function name is passed, otherwise returns shim.Success
func (cc *ContractChaincode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	nsFcn, _ := stub.GetFunctionAndParameters()
	if nsFcn == "" {
		return shim.Success([]byte("Default initiator successful."))
	}

	return cc.Invoke(stub)
}

// Invoke is called to update or query the ledger in a proposal transaction. Takes the
// args passed in the transaction and uses the first argument to identify the contract
// and function of that contract to be called. The remaining args are then used as
// parameters to that function. Args are converted from strings to the expected parameter
// types of the function before being passed. A transaction context is generated and is passed,
// if required, as the first parameter to the named function. Before and after functions are
// called before and after the named function passed if the contract defines such functions to
// exist. If the before function returns an error the named function is not called and its error
// is returned in shim.Error. If the after function returns an error then its value is returned
// to shim.Error otherwise the value returned from the named function is returned as shim.Success.
// If an unknown name is passed as part of the first arg a shim.Error is returned. If a valid
// name is passed but the function name is unknown then the contract with that name's
// unknown function is called and its value returned as success or error depending on it return. If no
// unknown function is defined for the contract then shim.Error is returned by Invoke. In the case of
// unknown function names being passed or the named function returning an error then the after function
// if defined is not called. The same transaction context is passed as a pointer to before, after, named
// and unknown functions on each Invoke.
func (cc *ContractChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	nsFcn, params := stub.GetFunctionAndParameters()

	li := strings.LastIndex(nsFcn, ":")

	var ns string
	var fn string

	if li == -1 {
		return shim.Error("Name was not passed")
	}

	ns = nsFcn[:li]
	fn = nsFcn[li+1:]

	if _, ok := cc.contracts[ns]; !ok {
		return shim.Error(fmt.Sprintf("Contract not found with name %s", ns))
	}

	nsContract := cc.contracts[ns]

	ctx := reflect.New(nsContract.transactionContextHandler)
	ctxIface := ctx.Interface().(TransactionContextInterface)
	ctxIface.SetStub(stub)

	beforeTransaction := nsContract.beforeTransaction

	if beforeTransaction != nil {
		_, errRes := beforeTransaction.call(ctx, params...)

		if errRes != nil {
			return shim.Error(errRes.Error())
		}
	}

	var successReturn string
	var errorReturn error

	if _, ok := nsContract.functions[fn]; !ok {
		unknownTransaction := nsContract.unknownTransaction
		if unknownTransaction == nil {
			return shim.Error(fmt.Sprintf("Function %s not found in contract %s", fn, ns))
		}

		successReturn, errorReturn = unknownTransaction.call(ctx, params...)
	} else {
		successReturn, errorReturn = nsContract.functions[fn].call(ctx, params...)
	}

	if errorReturn != nil {
		return shim.Error(errorReturn.Error())
	}

	afterTransaction := nsContract.afterTransaction

	if afterTransaction != nil {
		_, errRes := afterTransaction.call(ctx, params...)

		if errRes != nil {
			return shim.Error(errRes.Error())
		}
	}

	return shim.Success([]byte(successReturn))
}

func (cc *ContractChaincode) addContract(contract ContractInterface, excludeFuncs []string) {
	ns := contract.GetName()

	if ns == "" {
		ns = reflect.TypeOf(contract).Elem().Name()
	}

	if _, ok := cc.contracts[ns]; ok {
		panic(fmt.Sprintf("Multiple contracts being merged into chaincode with name %s", contract.GetName()))
	}

	ccn := contractChaincodeContract{}
	ccn.transactionContextHandler = reflect.ValueOf(contract.GetTransactionContextHandler()).Elem().Type()
	ccn.transactionContextPtrHandler = reflect.ValueOf(contract.GetTransactionContextHandler()).Type()
	ccn.functions = make(map[string]*contractFunction)

	scT := reflect.PtrTo(reflect.TypeOf(contract).Elem())
	scV := reflect.ValueOf(contract).Elem().Addr()

	ut, err := contract.GetUnknownTransaction()

	if err == nil && ut != nil {
		ccn.unknownTransaction = newContractFunctionFromFunc(ut, ccn.transactionContextPtrHandler)
	}

	bt, err := contract.GetBeforeTransaction()

	if err == nil && bt != nil {
		ccn.beforeTransaction = newContractFunctionFromFunc(bt, ccn.transactionContextPtrHandler)
	}

	at, err := contract.GetAfterTransaction()

	if err == nil && at != nil {
		ccn.afterTransaction = newContractFunctionFromFunc(at, ccn.transactionContextPtrHandler)
	}

	for i := 0; i < scT.NumMethod(); i++ {
		typeMethod := scT.Method(i)
		valueMethod := scV.Method(i)

		if !stringInSlice(typeMethod.Name, excludeFuncs) {
			ccn.functions[typeMethod.Name] = newContractFunctionFromReflect(typeMethod, valueMethod, ccn.transactionContextPtrHandler)
		}
	}

	cc.contracts[ns] = ccn
}
