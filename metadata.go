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
	"sort"

	"github.com/go-openapi/spec"

	"encoding/json"
)

// MetadataFunction stores details of a contract function
type MetadataFunction struct {
	spec.OperationProps
	TransactionID string           `json:"transactionId"`
	Return        []spec.Parameter `json:"return"`
}

// MarshalJSON custom handling of JSON marshall as OperationProps
// removes added fields
func (mf MetadataFunction) MarshalJSON() ([]byte, error) {
	mfMap := make(map[string]interface{})

	bytes, _ := json.Marshal(mf.OperationProps)

	json.Unmarshal(bytes, &mfMap)

	if len(mf.Return) != 0 {
		mfMap["return"] = mf.Return
	}

	mfMap["transactionId"] = mf.TransactionID

	return json.Marshal(mfMap)
}

// MetadataContract stores details of a contract
type MetadataContract struct {
	spec.InfoProps
	Namespace    string             `json:"namespace"`
	Transactions []MetadataFunction `json:"transactions"`
}

// MetadataContractChaincode stores details for a chaincode. Contains
// all contracts of the chaincode and details of those contracts
type MetadataContractChaincode struct {
	Contracts []MetadataContract `json:"contracts"`
}

func generateMetadata(cc contractChaincode) string {

	sscc := new(MetadataContractChaincode)
	sscc.Contracts = []MetadataContract{}

	for key := range cc.contracts {
		metadata := cc.contracts[key]
		simpNS := MetadataContract{}
		simpNS.Namespace = key
		simpNS.Transactions = []MetadataFunction{}

		for key := range metadata.functions {
			fn := metadata.functions[key]
			metaFunc := MetadataFunction{}
			metaFunc.TransactionID = key

			if fn.params.context != nil {
				schema := new(spec.Schema)
				schema.Typed("object", fn.params.context.String())

				param := spec.BodyParam("ctx", schema)
				metaFunc.Parameters = append(metaFunc.Parameters, *param)
			}

			for index, field := range fn.params.fields {
				schema, err := getSchema(field)

				if err != nil {
					panic(fmt.Sprintf("Failed to generate metadata. Invalid function parameter type. %s", err))
				}

				param := spec.BodyParam(fmt.Sprintf("param%d", index), schema)
				metaFunc.Parameters = append(metaFunc.Parameters, *param)
			}

			if fn.returns.success != nil {
				schema, err := getSchema(fn.returns.success)

				if err != nil {
					panic(fmt.Sprintf("Failed to generate metadata. Invalid function success return type. %s", err))
				}

				param := spec.BodyParam("success", schema)
				metaFunc.Return = append(metaFunc.Return, *param)
			}

			if fn.returns.error {
				schema := new(spec.Schema)
				schema.Typed("object", "error")
				param := spec.BodyParam("error", schema)

				metaFunc.Return = append(metaFunc.Return, *param)
			}

			simpNS.Transactions = append(simpNS.Transactions, metaFunc)
		}

		sort.Slice(simpNS.Transactions, func(i, j int) bool {
			return simpNS.Transactions[i].TransactionID < simpNS.Transactions[j].TransactionID
		})

		sscc.Contracts = append(sscc.Contracts, simpNS)
	}

	sort.Slice(sscc.Contracts, func(i, j int) bool {
		return sscc.Contracts[i].Namespace < sscc.Contracts[j].Namespace
	})

	ssccJSON, _ := json.Marshal(sscc)

	return string(ssccJSON)
}
