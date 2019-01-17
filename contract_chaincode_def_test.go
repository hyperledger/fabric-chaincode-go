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
	"os"
	"reflect"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"github.com/stretchr/testify/assert"
)

// ================================
// Helpers
// ================================

const invokeType = "INVOKE"
const initType = "INIT"

func callContractFunctionAndCheckError(t *testing.T, cc ContractChaincode, arguments []string, callType string, expectedMessage string) {
	t.Helper()

	callContractFunctionAndCheckResponse(t, cc, arguments, callType, expectedMessage, "error")
}

func callContractFunctionAndCheckSuccess(t *testing.T, cc ContractChaincode, arguments []string, callType string, expectedMessage string) {
	t.Helper()

	callContractFunctionAndCheckResponse(t, cc, arguments, callType, expectedMessage, "success")
}

func callContractFunctionAndCheckResponse(t *testing.T, cc ContractChaincode, arguments []string, callType string, expectedMessage string, expectedType string) {
	t.Helper()

	args := [][]byte{}
	for _, str := range arguments {
		arg := []byte(str)
		args = append(args, arg)
	}

	mockStub := shim.NewMockStub("smartContractTest", &cc)

	var response peer.Response

	if callType == initType {
		response = mockStub.MockInit(standardTxID, args)
	} else if callType == invokeType {
		response = mockStub.MockInvoke(standardTxID, args)
	} else {
		panic(fmt.Sprintf("Call type passed should be %s or %s. Value passed was %s", initType, invokeType, callType))
	}

	expectedResponse := shim.Success([]byte(expectedMessage))

	if expectedType == "error" {
		expectedResponse = shim.Error(expectedMessage)
	}

	assert.Equal(t, expectedResponse, response)
}

func testCallingContractFunctions(t *testing.T, callType string) {
	t.Helper()

	cc := convertC2CC()

	mc := myContract{}
	cc = convertC2CC(&mc)

	// Should error when name not known
	callContractFunctionAndCheckError(t, cc, []string{"somebadname:somebadfunctionname"}, callType, "Contract not found with name somebadname")

	// should return error when function not known and no unknown transaction specified
	mc.SetName("customname")
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckError(t, cc, []string{"customname:somebadfunctionname"}, callType, "Function somebadfunctionname not found in contract customname")

	// Should call default chaincode when name not passed
	callContractFunctionAndCheckError(t, cc, []string{"somebadfunctionname"}, callType, "Function somebadfunctionname not found in contract customname")

	mc = myContract{}
	cc = convertC2CC(&mc)

	// Should return success when function returns nothing
	callContractFunctionAndCheckSuccess(t, cc, []string{"myContract:ReturnsNothing"}, callType, "")

	// should return success when function returns no error
	callContractFunctionAndCheckSuccess(t, cc, []string{"myContract:ReturnsString"}, callType, mc.ReturnsString())

	// Should return error when function returns error
	callContractFunctionAndCheckError(t, cc, []string{"myContract:ReturnsError"}, callType, mc.ReturnsError().Error())

	// Should return error when function unknown and set unknown function returns error
	mc.SetUnknownTransaction(mc.ReturnsError)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckError(t, cc, []string{"myContract:somebadfunctionname"}, callType, mc.ReturnsError().Error())
	mc = myContract{}

	// Should return success when function unknown and set unknown function returns no error
	mc.SetUnknownTransaction(mc.ReturnsString)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"myContract:somebadfunctionname"}, callType, mc.ReturnsString())
	mc = myContract{}

	// Should return error when before function returns error and not call main function
	mc.SetBeforeTransaction(mc.ReturnsError)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckError(t, cc, []string{"myContract:ReturnsString"}, callType, mc.ReturnsError().Error())
	mc = myContract{}

	// Should return success from passed function when before function returns no error
	mc.SetBeforeTransaction(mc.ReturnsString)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"myContract:ReturnsString"}, callType, mc.ReturnsString())
	mc = myContract{}

	// Should return error when after function returns error
	mc.SetAfterTransaction(mc.ReturnsError)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckError(t, cc, []string{"myContract:ReturnsString"}, callType, mc.ReturnsError().Error())
	mc = myContract{}

	// Should return success from passed function when before function returns error
	mc.SetAfterTransaction(mc.ReturnsString)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"myContract:ReturnsString"}, callType, mc.ReturnsString())
	mc = myContract{}

	// Should call before, named then after functions in order
	mc.SetBeforeTransaction(mc.logBefore)
	mc.SetAfterTransaction(mc.logAfter)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"myContract:LogNamed"}, callType, "")
	assert.Equal(t, []string{"Before function called", "Named function called", "After function called"}, mc.called, "Expected called field of myContract to have logged in order before, named then after")
	mc = myContract{}

	// Should call before, unknown then after functions in order
	mc.SetBeforeTransaction(mc.logBefore)
	mc.SetAfterTransaction(mc.logAfter)
	mc.SetUnknownTransaction(mc.logUnknown)
	cc = convertC2CC(&mc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"myContract:somebadfunctionname"}, callType, "")
	assert.Equal(t, []string{"Before function called", "Unknown function called", "After function called"}, mc.called, "Expected called field of myContract to have logged in order before, named then after")
	mc = myContract{}

	// should pass the stub into transaction context as expected
	callContractFunctionAndCheckSuccess(t, cc, []string{"myContract:CheckContextStub"}, callType, "Stub as expected")

	sc := simpleTestContractWithCustomContext{}
	sc.SetTransactionContextHandler(new(customContext))
	cc = convertC2CC(&sc)

	//should use a custom transaction context when one is set
	callContractFunctionAndCheckSuccess(t, cc, []string{"simpleTestContractWithCustomContext:CheckCustomContext"}, callType, "I am custom context")

	//should use same ctx for all calls
	sc.SetBeforeTransaction(sc.SetValInCustomContext)
	cc = convertC2CC(&sc)
	callContractFunctionAndCheckSuccess(t, cc, []string{"simpleTestContractWithCustomContext:GetValInCustomContext", standardValue}, callType, standardValue)

	sc.SetAfterTransaction(sc.GetValInCustomContext)
	cc = convertC2CC(&sc)
	callContractFunctionAndCheckError(t, cc, []string{"simpleTestContractWithCustomContext:SetValInCustomContext", "some other value"}, callType, "I wanted a standard value")
}

// ================================
// Tests
// ================================

func TestReflectMetadata(t *testing.T) {
	cc := ContractChaincode{}
	cc.title = "some title"
	cc.version = "some version"

	complexType := reflect.TypeOf(complex64(1))

	var getSchemaErr error

	// Should panic if get schema panics
	someBadFunctionContractFunction := new(contractFunction)
	someBadFunctionContractFunction.params = contractFunctionParams{
		basicContextPtrType,
		[]reflect.Type{stringRefType, complexType},
	}
	bcFuncs := make(map[string]*contractFunction)
	bcFuncs["BadFunction"] = someBadFunctionContractFunction
	bcccn := contractChaincodeContract{
		"some version", bcFuncs, nil, nil, nil, nil, nil,
	}

	cc.contracts = map[string]contractChaincodeContract{
		"": bcccn,
	}

	_, getSchemaErr = getSchema(complexType, nil)

	assert.PanicsWithValue(t, fmt.Sprintf("Failed to generate metadata. Invalid function parameter type. %s", getSchemaErr), func() { cc.reflectMetadata() }, "should have panicked with bad contract function params")

	// Should panic if get schema panics
	anotherBadFunctionContractFunction := new(contractFunction)
	anotherBadFunctionContractFunction.params = contractFunctionParams{
		basicContextPtrType,
		[]reflect.Type{stringRefType},
	}
	anotherBadFunctionContractFunction.returns = contractFunctionReturns{}
	anotherBadFunctionContractFunction.returns.success = complexType
	abcFuncs := make(map[string]*contractFunction)
	abcFuncs["AnotherBadFunction"] = anotherBadFunctionContractFunction
	abcccn := contractChaincodeContract{
		"some version", abcFuncs, nil, nil, nil, nil, nil,
	}

	cc.contracts = map[string]contractChaincodeContract{
		"": abcccn,
	}

	_, getSchemaErr = getSchema(complexType, nil)

	assert.PanicsWithValue(t, fmt.Sprintf("Failed to generate metadata. Invalid function success return type. %s", getSchemaErr), func() { cc.reflectMetadata() }, "should have panicked with bad contract function success return")

	// setup for not panicking tests
	type SomeStruct struct {
		Prop1 string `json:"Prop1"`
	}

	someStructMetadata := ObjectMetadata{}
	someStructMetadata.ID = "SomeStruct"
	someStructMetadata.Properties = map[string]spec.Schema{"Prop1": *spec.StringProperty()}
	someStructMetadata.AdditionalProperties = false
	someStructMetadata.Required = []string{"Prop1"}

	errorSchema := spec.Schema{}
	errorSchema.Type = []string{"object"}
	errorSchema.Format = "error"

	someFunctionContractFunction := new(contractFunction)

	someFunctionMetadata := TransactionMetadata{}
	someFunctionMetadata.Name = "SomeFunction"
	someFunctionMetadata.Tag = []string{"submitTx"}

	someFunctionMetadataNoTag := TransactionMetadata{}
	someFunctionMetadataNoTag.Name = "SomeFunction"
	someFunctionMetadataNoTag.Tag = []string{}

	anotherFunctionContractFunction := new(contractFunction)
	anotherFunctionContractFunction.params = contractFunctionParams{
		basicContextPtrType,
		[]reflect.Type{stringRefType, reflect.TypeOf(SomeStruct{})},
	}
	anotherFunctionContractFunction.returns = contractFunctionReturns{
		reflect.TypeOf(SomeStruct{}),
		true,
	}

	param0AsParam := ParameterMetadata{}
	param0AsParam.Name = "param0"
	param0AsParam.Schema = *(stringTypeVar.getSchema())

	param1AsParam := ParameterMetadata{}
	param1AsParam.Name = "param1"
	param1AsParam.Schema = *spec.RefSchema("#/components/schemas/SomeStruct")

	anotherFunctionMetadata := TransactionMetadata{}
	anotherFunctionMetadata.Parameters = []ParameterMetadata{
		param0AsParam,
		param1AsParam,
	}

	successAsParam := ParameterMetadata{}
	successAsParam.Name = "success"
	successAsParam.Schema = *spec.RefSchema("#/components/schemas/SomeStruct")

	errorAsParam := ParameterMetadata{}
	errorAsParam.Name = "error"
	errorAsParam.Schema = errorSchema

	anotherFunctionMetadata.Returns = []ParameterMetadata{
		successAsParam,
		errorAsParam,
	}
	anotherFunctionMetadata.Name = "AnotherFunction"
	anotherFunctionMetadata.Tag = []string{"submitTx"}

	var expectedMetadata ContractChaincodeMetadata
	var contractInfo spec.Info

	chaincodeInfo := spec.Info{}
	chaincodeInfo.Title = "some title"
	chaincodeInfo.Version = "some version"

	scFuncs := make(map[string]*contractFunction)
	scFuncs["SomeFunction"] = someFunctionContractFunction
	scccn := contractChaincodeContract{
		"some version", scFuncs, nil, nil, nil, nil, nil,
	}

	cscFuncs := make(map[string]*contractFunction)
	cscFuncs["SomeFunction"] = someFunctionContractFunction

	cscFuncs["AnotherFunction"] = anotherFunctionContractFunction
	cscccn := contractChaincodeContract{
		"some other version", cscFuncs, nil, nil, nil, nil, nil,
	}

	// Should handle generating metadata for a single name
	cc.contracts = map[string]contractChaincodeContract{
		"SomeContract": scccn,
	}

	contractInfo = spec.Info{}
	contractInfo.Title = "SomeContract"
	contractInfo.Version = "some version"

	expectedMetadata = ContractChaincodeMetadata{}
	expectedMetadata.Info = chaincodeInfo
	expectedMetadata.Components = ComponentMetadata{}
	expectedMetadata.Components.Schemas = make(map[string]ObjectMetadata)
	expectedMetadata.Contracts = make(map[string]ContractMetadata)
	expectedMetadata.Contracts["SomeContract"] = ContractMetadata{
		Info: contractInfo,
		Name: "SomeContract",
		Transactions: []TransactionMetadata{
			someFunctionMetadata,
		},
	}

	testMetadata(t, cc.reflectMetadata(), expectedMetadata)

	// Should not add tag for system contract
	cc.contracts = map[string]contractChaincodeContract{
		"org.hyperledger.fabric": scccn,
	}

	contractInfo = spec.Info{}
	contractInfo.Title = "org.hyperledger.fabric"
	contractInfo.Version = "some version"

	expectedMetadata = ContractChaincodeMetadata{}
	expectedMetadata.Info = chaincodeInfo
	expectedMetadata.Components = ComponentMetadata{}
	expectedMetadata.Components.Schemas = make(map[string]ObjectMetadata)
	expectedMetadata.Contracts = make(map[string]ContractMetadata)
	expectedMetadata.Contracts["org.hyperledger.fabric"] = ContractMetadata{
		Info: contractInfo,
		Name: "org.hyperledger.fabric",
		Transactions: []TransactionMetadata{
			someFunctionMetadataNoTag,
		},
	}

	testMetadata(t, cc.reflectMetadata(), expectedMetadata)

	// Should handle generating metadata functions alphabetically on ID
	contractInfo = spec.Info{}
	contractInfo.Title = "customname"
	contractInfo.Version = "some other version"

	cc.contracts = map[string]contractChaincodeContract{
		"customname": cscccn,
	}

	expectedMetadata = ContractChaincodeMetadata{}
	expectedMetadata.Info = chaincodeInfo
	expectedMetadata.Components = ComponentMetadata{}
	expectedMetadata.Components.Schemas = make(map[string]ObjectMetadata)
	expectedMetadata.Components.Schemas["SomeStruct"] = someStructMetadata
	expectedMetadata.Contracts = make(map[string]ContractMetadata)
	expectedMetadata.Contracts["customname"] = ContractMetadata{
		Info: contractInfo,
		Name: "customname",
		Transactions: []TransactionMetadata{
			anotherFunctionMetadata,
			someFunctionMetadata,
		},
	}

	testMetadata(t, cc.reflectMetadata(), expectedMetadata)

	// should handle generating metadata for multiple names
	cc.contracts = map[string]contractChaincodeContract{
		"somename":   scccn,
		"customname": cscccn,
	}

	expectedMetadata = ContractChaincodeMetadata{}
	expectedMetadata.Info = chaincodeInfo
	expectedMetadata.Components = ComponentMetadata{}
	expectedMetadata.Components.Schemas = make(map[string]ObjectMetadata)
	expectedMetadata.Components.Schemas["SomeStruct"] = someStructMetadata

	expectedMetadata.Contracts = make(map[string]ContractMetadata)

	contractInfo = spec.Info{}
	contractInfo.Title = "somename"
	contractInfo.Version = "some version"

	expectedMetadata.Contracts["somename"] = ContractMetadata{
		Info: contractInfo,
		Name: "somename",
		Transactions: []TransactionMetadata{
			someFunctionMetadata,
		},
	}

	contractInfo = spec.Info{}
	contractInfo.Title = "customname"
	contractInfo.Version = "some other version"

	expectedMetadata.Contracts["customname"] = ContractMetadata{
		Info: contractInfo,
		Name: "customname",
		Transactions: []TransactionMetadata{
			anotherFunctionMetadata,
			someFunctionMetadata,
		},
	}

	testMetadata(t, cc.reflectMetadata(), expectedMetadata)
}

func TestAugmentMetadata(t *testing.T) {
	someFunctionContractFunction := new(contractFunction)

	scFuncs := make(map[string]*contractFunction)
	scFuncs["SomeFunction"] = someFunctionContractFunction
	scccn := contractChaincodeContract{
		"some version", scFuncs, nil, nil, nil, nil, nil,
	}

	cc := ContractChaincode{}
	cc.contracts = map[string]contractChaincodeContract{
		"somename": scccn,
	}

	// Should blend file and reflected metadata
	metadataBytes := []byte("{\"info\":{\"title\":\"my contract\",\"version\":\"0.0.1\"},\"contracts\":{},\"components\":{}}")

	createMetadataJSONFile(metadataBytes, os.ModePerm)
	cleanupMetadataJSONFile()

	fileMetadata := readMetadataFile()
	reflectedMetadata := cc.reflectMetadata()

	cc.augmentMetadata()
	fileMetadata.append(reflectedMetadata)

	testMetadata(t, cc.metadata, fileMetadata)
}

func TestAddContract(t *testing.T) {
	ciT := reflect.TypeOf((*ContractInterface)(nil)).Elem()
	var fullExclude []string
	for i := 0; i < ciT.NumMethod(); i++ {
		fullExclude = append(fullExclude, ciT.Method(i).Name)
	}

	cT := reflect.TypeOf(new(Contract))
	for i := 0; i < cT.NumMethod(); i++ {
		methodName := cT.Method(i).Name
		if !stringInSlice(methodName, fullExclude) {
			fullExclude = append(fullExclude, methodName)
		}
	}

	var cc *ContractChaincode
	sc := simpleTestContract{}
	csc := simpleTestContract{}
	csc.name = "customname"

	// Should panic when contract passed with non unique name
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	cc.contracts["customname"] = contractChaincodeContract{}
	sc = simpleTestContract{}
	sc.SetName("customname")
	assert.PanicsWithValue(t, "Multiple contracts being merged into chaincode with name customname", func() { cc.addContract(&sc, []string{}) }, "didn't panic when multiple contracts share same custom name")
	sc = simpleTestContract{}

	// Should add contract with default name to chaincode
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	cc.addContract(&sc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts["simpleTestContract"], sc)

	// Should add contract with custom name to chaincode
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	cc.addContract(&csc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts["customname"], csc)

	// Should add to map of chaincode not remove other chaincodes
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	cc.addContract(&sc, fullExclude)
	cc.addContract(&csc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts["simpleTestContract"], sc)
	testContractChaincodeContractRepresentsContract(t, cc.contracts["customname"], csc)

	// Should use contracts version
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	sc.version = "some version"
	cc.addContract(&sc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts["simpleTestContract"], sc)
	sc.version = ""

	// Should add contract to map with unknown transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	sc.unknownTransaction = sc.DoSomething
	cc.addContract(&sc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts["simpleTestContract"], sc)
	sc.unknownTransaction = nil

	// Should add contract to map with before transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	sc.beforeTransaction = sc.DoSomething
	cc.addContract(&sc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts["simpleTestContract"], sc)
	sc.beforeTransaction = nil

	// Should add contract to map with after transaction
	cc = new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)
	sc.afterTransaction = sc.DoSomething
	cc.addContract(&sc, fullExclude)
	testContractChaincodeContractRepresentsContract(t, cc.contracts["simpleTestContract"], sc)
	sc.afterTransaction = nil
}

func TestCreateNewChaincode(t *testing.T) {
	mc := new(myContract)

	// Should call convertC2CC
	actual := CreateNewChaincode(mc)
	expected := convertC2CC(mc)

	assert.Equal(t, expected.defaultContract, actual.defaultContract, "should return defaultContract same as convertC2CC")
	assert.Equal(t, len(expected.contracts), len(actual.contracts), "should return same as convertC2CC")

	for k := range actual.contracts {
		_, ok := expected.contracts[k]

		assert.True(t, ok, "should return same keys as convertC2CC")
	}
}

func TestStart(t *testing.T) {
	mc := new(myContract)

	cc := CreateNewChaincode(mc)

	assert.EqualError(t, cc.Start(), shim.Start(&cc).Error(), "should call shim.Start()")
}

func TestGetTitle(t *testing.T) {
	cc := ContractChaincode{}
	cc.title = "some title"

	assert.Equal(t, "some title", cc.GetTitle(), "should get the title when set")
}

func TestSetTitle(t *testing.T) {
	cc := ContractChaincode{}
	cc.SetTitle("some title")

	assert.Equal(t, "some title", cc.title, "should set the title")
}

func TestGetContractVersion(t *testing.T) {
	cc := ContractChaincode{}
	cc.version = "some version"

	assert.Equal(t, "some version", cc.GetVersion(), "should get the version when set")
}

func TestSetContractVersion(t *testing.T) {
	cc := ContractChaincode{}
	cc.SetVersion("some version")

	assert.Equal(t, "some version", cc.version, "should set the version")
}

func TestInit(t *testing.T) {
	// Should just return when no function name passed
	cc := convertC2CC()
	mockStub := shim.NewMockStub("blank fcn", &cc)
	assert.Equal(t, shim.Success([]byte("Default initiator successful.")), cc.Init(mockStub), "should just return success on init with no function passed")

	// Should call via invoke
	testCallingContractFunctions(t, initType)
}

func TestInvoke(t *testing.T) {
	testCallingContractFunctions(t, invokeType)
}
