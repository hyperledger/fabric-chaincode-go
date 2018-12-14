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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/go-openapi/spec"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

const metadataFolder = "contract-metadata"
const metadataFile = "metadata.json"

var logger = shim.NewLogger("contractapi/metadata.go")

// Helper for OS testing
type osHlp interface {
	Executable() (string, error)
	Stat(string) (os.FileInfo, error)
}

type osHlpStr struct{}

func (o osHlpStr) Executable() (string, error) {
	return os.Executable()
}

func (o osHlpStr) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

var osHelper osHlp = osHlpStr{}

// GetJSONSchema returns the JSON schema used for metadata
func GetJSONSchema() string {
	file, err := readLocalFile("schema/schema.json")

	if err != nil {
		panic(fmt.Sprintf("Unable to read JSON schema. Error: %s", err.Error()))
	}

	return string(file)
}

// ParameterMetadata details about a parameter used for a transaction
type ParameterMetadata struct {
	Description string      `json:"description,omitempty"`
	Name        string      `json:"name"`
	Required    bool        `json:"required,omitempty"`
	Schema      spec.Schema `json:"schema"`
}

// TransactionMetadata contains information on what makes up a transaction
type TransactionMetadata struct {
	Parameters []ParameterMetadata `json:"parameters,omitempty"`
	Returns    []ParameterMetadata `json:"returns,omitempty"`
	Tag        []string            `json:"tag,omitempty"`
	Name       string              `json:"name"`
}

// ContractMetadata contains information about what makes up a contract
type ContractMetadata struct {
	Info         spec.Info             `json:"info,omitempty"`
	Name         string                `json:"name"`
	Transactions []TransactionMetadata `json:"transactions"`
}

// ObjectMetadata description of an asset
type ObjectMetadata struct {
	ID         string              `json:"$id"`
	Properties []ParameterMetadata `json:"properties"`
}

// ComponentMetadata does something
type ComponentMetadata struct {
	Schemas map[string]ObjectMetadata `json:"schemas,omitempty"`
}

// ContractChaincodeMetadata describes a chaincode made using the contract api
type ContractChaincodeMetadata struct {
	Info       spec.Info                   `json:"info,omitempty"`
	Contracts  map[string]ContractMetadata `json:"contracts"`
	Components ComponentMetadata           `json:"components"`
}

func (ccm *ContractChaincodeMetadata) append(source ContractChaincodeMetadata) {
	if reflect.DeepEqual(ccm.Info, spec.Info{}) {
		ccm.Info = source.Info
	}

	if len(ccm.Contracts) == 0 {
		if ccm.Contracts == nil {
			ccm.Contracts = make(map[string]ContractMetadata)
		}

		for key, value := range source.Contracts {
			ccm.Contracts[key] = value
		}
	}

	if reflect.DeepEqual(ccm.Components, ComponentMetadata{}) {
		ccm.Components = source.Components
	}
}

func readMetadataFile() ContractChaincodeMetadata {
	fileMetadata := ContractChaincodeMetadata{}

	ex, execErr := osHelper.Executable()
	if execErr != nil {
		logger.Error(fmt.Sprintf("Error finding location of running executable. Defaulting to Reflected metadata. %s", execErr.Error()))

		return fileMetadata
	}
	exPath := filepath.Dir(ex)
	metadataPath := filepath.Join(exPath, metadataFolder, metadataFile)

	_, err := osHelper.Stat(metadataPath)

	logger.Error(err)

	if os.IsNotExist(err) {
		logger.Info("No metadata file supplied")
		return fileMetadata
	}

	fileMetadata.Contracts = make(map[string]ContractMetadata)

	metadataBytes, err := ioutil.ReadFile(metadataPath)

	if err != nil {
		panic(fmt.Sprintf("Failed to get existing metadata. Could not read file %s. %s", metadataPath, err))
	}

	schema := new(spec.Schema)
	json.Unmarshal([]byte(GetJSONSchema()), schema)

	metadata := map[string]interface{}{}
	json.Unmarshal(metadataBytes, &metadata)

	err = validate.AgainstSchema(schema, metadata, strfmt.Default)

	if err != nil {
		panic(fmt.Sprintf("Failed to get existing metadata. Given file did not match schema: %s", err.Error()))
	}

	json.Unmarshal(metadataBytes, &fileMetadata)

	return fileMetadata
}
