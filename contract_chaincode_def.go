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
	"sort"
	"strings"

	"github.com/go-openapi/spec"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

type contractChaincodeContract struct {
	version                      string
	functions                    map[string]*contractFunction
	unknownTransaction           *transactionHandler
	beforeTransaction            *transactionHandler
	afterTransaction             *transactionHandler
	transactionContextHandler    reflect.Type
	transactionContextPtrHandler reflect.Type
}

// ContractChaincode a struct to meet the chaincode interface and provide routing of calls to contracts
type ContractChaincode struct {
	defaultContract string
	contracts       map[string]contractChaincodeContract
	metadata        ContractChaincodeMetadata
	title           string
	version         string
}

// SystemContractName the name of the system smart contract
const SystemContractName = "org.hyperledger.fabric"

// CreateNewChaincode creates a new chaincode using contracts passed. The function parses each
// of the passed functions and stores details about their make-up to be used by the chaincode.
// Public functions of the contracts are stored an are made callable in the chaincode. The function
// will panic if contracts are invalid e.g. public functions take in illegal types. A system contract is added
// to the chaincode which provides functionality for getting the metadata of the chaincode. The generated
// metadata is a JSON formatted MetadataContractChaincode containing each contract as a name and details
// of the public functions. It also outlines version details for contracts and the chaincode. If these are blank
// strings this is set to latest. The names for parameters do not match those used in the functions instead they are
// recorded as param0, param1, ..., paramN. If there exists a file META-INF/chaincode/metadata.json then this
// will overwrite the generated metadata. The contents of this file must validate against the schema.
func CreateNewChaincode(contracts ...ContractInterface) ContractChaincode {
	return convertC2CC(contracts...)
}

// Start starts the chaincode in the fabric shim
func (cc *ContractChaincode) Start() error {
	return shim.Start(cc)
}

// SetTitle sets the title
func (cc *ContractChaincode) SetTitle(title string) {
	cc.title = title
}

// SetVersion sets the version
func (cc *ContractChaincode) SetVersion(version string) {
	cc.version = version
}

// SetDefault sets the default contract name
func (cc *ContractChaincode) SetDefault(c ContractInterface) {
	cc.defaultContract = c.GetName()
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
// unknown function names being passed (and the unknown handler returns an error) or the named function returning an error then the after function
// if defined is not called. The same transaction context is passed as a pointer to before, after, named
// and unknown functions on each Invoke. If no contract name is passed then the default contract is used.
func (cc *ContractChaincode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	nsFcn, params := stub.GetFunctionAndParameters()

	li := strings.LastIndex(nsFcn, ":")

	var ns string
	var fn string

	if li == -1 {
		ns = cc.defaultContract
		fn = nsFcn
	} else {
		ns = nsFcn[:li]
		fn = nsFcn[li+1:]
	}

	if _, ok := cc.contracts[ns]; !ok {
		return shim.Error(fmt.Sprintf("Contract not found with name %s", ns))
	}

	nsContract := cc.contracts[ns]

	ctx := reflect.New(nsContract.transactionContextHandler)
	ctxIface := ctx.Interface().(TransactionContextInterface)
	ctxIface.SetStub(stub)

	beforeTransaction := nsContract.beforeTransaction

	if beforeTransaction != nil {
		_, _, errRes := beforeTransaction.call(ctx, nil)

		if errRes != nil {
			return shim.Error(errRes.Error())
		}
	}

	var successReturn string
	var successIFace interface{}
	var errorReturn error

	if _, ok := nsContract.functions[fn]; !ok {
		unknownTransaction := nsContract.unknownTransaction
		if unknownTransaction == nil {
			return shim.Error(fmt.Sprintf("Function %s not found in contract %s", fn, ns))
		}

		successReturn, successIFace, errorReturn = unknownTransaction.call(ctx, nil)
	} else {
		var transactionSchema *TransactionMetadata

		for _, v := range cc.metadata.Contracts[ns].Transactions {
			if v.Name == fn {
				transactionSchema = &v
				break
			}
		}

		successReturn, successIFace, errorReturn = nsContract.functions[fn].call(ctx, transactionSchema, &cc.metadata.Components, params...)
	}

	if errorReturn != nil {
		return shim.Error(errorReturn.Error())
	}

	afterTransaction := nsContract.afterTransaction

	if afterTransaction != nil {
		_, _, errRes := afterTransaction.call(ctx, successIFace)

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
	ccn.version = contract.GetVersion()

	if ccn.version == "" {
		ccn.version = "latest"
	}

	scT := reflect.PtrTo(reflect.TypeOf(contract).Elem())
	scV := reflect.ValueOf(contract).Elem().Addr()

	ut, err := contract.GetUnknownTransaction()

	if err == nil && ut != nil {
		ccn.unknownTransaction = newTransactionHandler(ut, ccn.transactionContextPtrHandler, unknown)
	}

	bt, err := contract.GetBeforeTransaction()

	if err == nil && bt != nil {
		ccn.beforeTransaction = newTransactionHandler(bt, ccn.transactionContextPtrHandler, before)
	}

	at, err := contract.GetAfterTransaction()

	if err == nil && at != nil {
		ccn.afterTransaction = newTransactionHandler(at, ccn.transactionContextPtrHandler, after)
	}

	for i := 0; i < scT.NumMethod(); i++ {
		typeMethod := scT.Method(i)
		valueMethod := scV.Method(i)

		if !stringInSlice(typeMethod.Name, excludeFuncs) {
			ccn.functions[typeMethod.Name] = newContractFunctionFromReflect(typeMethod, valueMethod, ccn.transactionContextPtrHandler)
		}
	}

	cc.contracts[ns] = ccn

	if cc.defaultContract == "" {
		cc.defaultContract = ns
	}
}

func (cc *ContractChaincode) reflectMetadata() ContractChaincodeMetadata {
	reflectedMetadata := ContractChaincodeMetadata{}
	reflectedMetadata.Contracts = make(map[string]ContractMetadata)
	reflectedMetadata.Info.Version = cc.version
	reflectedMetadata.Info.Title = cc.title
	reflectedMetadata.Components.Schemas = make(map[string]ObjectMetadata)

	if reflectedMetadata.Info.Version == "" {
		reflectedMetadata.Info.Version = "latest"
	}

	if reflectedMetadata.Info.Title == "" {
		reflectedMetadata.Info.Title = "undefined"
	}

	for key, contract := range cc.contracts {
		contractMetadata := ContractMetadata{}
		contractMetadata.Name = key
		contractMetadata.Info.Version = contract.version
		contractMetadata.Info.Title = key

		for key, fn := range contract.functions {
			transactionMetadata := TransactionMetadata{}
			transactionMetadata.Name = key
			transactionMetadata.Tag = []string{}

			if contractMetadata.Name != SystemContractName {
				transactionMetadata.Tag = append(transactionMetadata.Tag, "submitTx")
			}

			for index, field := range fn.params.fields {
				schema, err := getSchema(field, &reflectedMetadata.Components)

				if err != nil {
					panic(fmt.Sprintf("Failed to generate metadata. Invalid function parameter type. %s", err))
				}

				param := ParameterMetadata{}
				param.Name = fmt.Sprintf("param%d", index)
				param.Schema = *schema

				transactionMetadata.Parameters = append(transactionMetadata.Parameters, param)
			}

			if fn.returns.success != nil {
				schema, err := getSchema(fn.returns.success, &reflectedMetadata.Components)

				if err != nil {
					panic(fmt.Sprintf("Failed to generate metadata. Invalid function success return type. %s", err))
				}

				param := ParameterMetadata{}
				param.Name = "success"
				param.Schema = *schema

				transactionMetadata.Returns = append(transactionMetadata.Returns, param)
			}

			if fn.returns.error {
				schema := spec.Schema{}
				schema.Type = []string{"object"}
				schema.Format = "error"

				param := ParameterMetadata{}
				param.Name = "error"
				param.Schema = schema

				transactionMetadata.Returns = append(transactionMetadata.Returns, param)
			}

			contractMetadata.Transactions = append(contractMetadata.Transactions, transactionMetadata)
		}

		sort.Slice(contractMetadata.Transactions, func(i, j int) bool {
			return contractMetadata.Transactions[i].Name < contractMetadata.Transactions[j].Name
		})

		reflectedMetadata.Contracts[key] = contractMetadata
	}

	return reflectedMetadata
}

func (cc *ContractChaincode) augmentMetadata() {
	fileMetadata := readMetadataFile()
	reflectedMetadata := cc.reflectMetadata()

	fileMetadata.append(reflectedMetadata)

	cc.metadata = fileMetadata
}
