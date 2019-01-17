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
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
)

// ================================
// Helpers
// ================================

func testConvertCC(t *testing.T, testData []simpleTestContract) {
	t.Helper()

	contractInterfaces := []ContractInterface{}

	for i := 0; i < len(testData); i++ {
		contractInterfaces = append(contractInterfaces, &testData[i])
	}

	cc := convertC2CC(contractInterfaces...)

	// Plus 1 as system contract
	assert.Equal(t, len(testData)+1, len(cc.contracts), "Didn't map correct number of smart contracts")

	expectedSysMetadata := ContractChaincodeMetadata{}
	expectedSysMetadata.Info = spec.Info{}
	expectedSysMetadata.Info.Title = "undefined"
	expectedSysMetadata.Info.Version = "latest"
	expectedSysMetadata.Contracts = make(map[string]ContractMetadata)

	successSchema := spec.Schema{}
	successSchema.Type = []string{"string"}

	errorSchema := spec.Schema{}
	errorSchema.Type = []string{"object"}
	errorSchema.Format = "error"

	successMetadata := ParameterMetadata{}
	successMetadata.Name = "success"
	successMetadata.Schema = successSchema

	errorMetadata := ParameterMetadata{}
	errorMetadata.Name = "error"
	errorMetadata.Schema = errorSchema

	simpleContractFunctionMetadata := TransactionMetadata{}
	simpleContractFunctionMetadata.Name = "DoSomething"
	simpleContractFunctionMetadata.Tag = []string{"submitTx"}
	simpleContractFunctionMetadata.Returns = []ParameterMetadata{successMetadata, errorMetadata}

	// Test that the data set for each contract in chaincode is correct e.g. unknown fn set etc
	for i := 0; i < len(testData); i++ {
		contract := testData[i]
		ns := contract.GetName()

		if ns == "" {
			ns = reflect.TypeOf(contract).Name()
		}

		nsContract, ok := cc.contracts[ns]

		contractMetadata := ContractMetadata{}
		contractMetadata.Info = spec.Info{}
		contractMetadata.Info.Title = ns
		contractMetadata.Info.Version = "latest"
		contractMetadata.Name = ns
		contractMetadata.Transactions = []TransactionMetadata{
			simpleContractFunctionMetadata,
		}

		expectedSysMetadata.Contracts[ns] = contractMetadata

		assert.True(t, ok, "should have name in map of contracts")

		// simpleTestContract should only have 1 function DoSomething
		assert.Equal(t, 1, len(nsContract.functions), "should have same number of functions as a simpleTestContract")

		testContractChaincodeContractRepresentsContract(t, nsContract, contract)
	}

	// should have system contract
	sysContract, ok := cc.contracts[SystemContractName]

	assert.True(t, ok, "should have added a system contract with other contracts")

	fn, ok := sysContract.functions["GetMetadata"]

	assert.True(t, ok, "should have GetMetadata for system contract")

	systemContractFunctionMetadata := TransactionMetadata{}
	systemContractFunctionMetadata.Name = "GetMetadata"
	systemContractFunctionMetadata.Returns = []ParameterMetadata{
		successMetadata,
	}

	systemContractMetadata := ContractMetadata{}
	systemContractMetadata.Info = spec.Info{}
	systemContractMetadata.Info.Title = "org.hyperledger.fabric"
	systemContractMetadata.Info.Version = "latest"
	systemContractMetadata.Name = SystemContractName
	systemContractMetadata.Transactions = []TransactionMetadata{
		systemContractFunctionMetadata,
	}

	expectedSysMetadata.Contracts[SystemContractName] = systemContractMetadata

	metadata, _ := fn.call(reflect.Value{}, TransactionMetadata{}, ComponentMetadata{})

	ccMetadata := ContractChaincodeMetadata{}

	json.Unmarshal([]byte(metadata), &ccMetadata)

	testMetadata(t, ccMetadata, expectedSysMetadata)
}

// ================================
// Tests
// ================================

func TestConvertC2CC(t *testing.T) {
	sc := simpleTestContract{}

	csc := simpleTestContract{}
	csc.name = "customname"

	// Should create a valid chaincode from a single contract handling no ns
	testConvertCC(t, []simpleTestContract{sc})

	// Should create a valid chaincode from a single contract with a custom ns
	testConvertCC(t, []simpleTestContract{csc})

	// Should create a valid chaincode from multiple smart contracts
	testConvertCC(t, []simpleTestContract{sc, csc})

	// Should panic when contract has function with same name as a Contract function but does not embed Contract and function is invalid
	assert.PanicsWithValue(t, fmt.Sprintf("SetAfterTransaction contains invalid parameter type. Type interface {} is not valid. Expected a struct, one of the basic types %s, an array/slice of these, or one of these additional types %s", listBasicTypes(), basicContextPtrType.String()), func() { convertC2CC(new(Contract)) }, "should have panicked due to bad function format")
}
