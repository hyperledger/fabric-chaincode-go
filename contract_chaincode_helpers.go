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
	"reflect"
)

var contractStringType = reflect.TypeOf(Contract{}).String()

func convertC2CC(contracts ...ContractInterface) *ContractChaincode {
	ciT := reflect.TypeOf((*ContractInterface)(nil)).Elem()
	var ciMethods []string
	for i := 0; i < ciT.NumMethod(); i++ {
		ciMethods = append(ciMethods, ciT.Method(i).Name)
	}

	cT := reflect.TypeOf(new(Contract))
	var contractMethods []string
	for i := 0; i < cT.NumMethod(); i++ {
		methodName := cT.Method(i).Name
		if !stringInSlice(methodName, ciMethods) {
			contractMethods = append(contractMethods, methodName)
		}
	}

	cc := new(ContractChaincode)
	cc.contracts = make(map[string]contractChaincodeContract)

	for _, contract := range contracts {
		additionalExcludes := []string{}
		if embedsStruct(contract, "contractapi.Contract") {
			additionalExcludes = contractMethods
		}
		cc.addContract(contract, append(ciMethods, additionalExcludes...))
	}

	sysC := new(systemContract)
	sysC.SetName(SystemContractName)

	cc.addContract(sysC, append(ciMethods, contractMethods...))

	sccnStore := []contractChaincodeContract{}

	for k := range cc.contracts {
		sccnStore = append(sccnStore, cc.contracts[k])
	}

	cc.augmentMetadata()

	metadataJSON, _ := json.Marshal(cc.metadata)

	sysC.setMetadata(string(metadataJSON))

	return cc
}
