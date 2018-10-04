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
)

// MetadataParam contains details about a parameter for a function
// storing the param name and type
type MetadataParam struct {
	Name     string `json:"name"`
	DataType string `json:"dataType"`
}

// MetadataFunction contains details on the parameters a
// function takes and the types it returns
type MetadataFunction struct {
	Params  []MetadataParam `json:"params"`
	Returns []string        `json:"returns"`
}

// MetadataNamespace stores details of the transactions that exist
// for a namespace
type MetadataNamespace struct {
	Transactions map[string]MetadataFunction `json:"transactions"`
}

// MetadataContractChaincode stores details for a chaincode. Contains
// all namespaces of the chaincode and details of those namespaces
type MetadataContractChaincode struct {
	Namespaces map[string]MetadataNamespace `json:"namespaces"`
}

func generateMetadata(cc contractChaincode) string {

	sscc := new(MetadataContractChaincode)
	sscc.Namespaces = make(map[string]MetadataNamespace)

	for key := range cc.contracts {
		metadata := cc.contracts[key]
		simpNS := MetadataNamespace{}
		simpNS.Transactions = make(map[string]MetadataFunction)

		for key := range metadata.functions {
			fn := metadata.functions[key]
			metaFunc := MetadataFunction{}
			metaFunc.Params = []MetadataParam{}
			metaFunc.Returns = []string{}

			if fn.params.context != nil {
				param := MetadataParam{
					"ctx",
					fn.params.context.String(),
				}
				metaFunc.Params = append(metaFunc.Params, param)
			}

			for index, field := range fn.params.fields {
				param := MetadataParam{
					fmt.Sprintf("param%d", index),
					field.String(),
				}
				metaFunc.Params = append(metaFunc.Params, param)
			}

			if fn.returns.success != nil {
				metaFunc.Returns = append(metaFunc.Returns, fn.returns.success.String())
			}

			if fn.returns.error {
				metaFunc.Returns = append(metaFunc.Returns, "error")
			}

			simpNS.Transactions[key] = metaFunc

		}

		sscc.Namespaces[key] = simpNS
	}

	ssccJSON, _ := json.Marshal(sscc)

	return string(ssccJSON)
}
