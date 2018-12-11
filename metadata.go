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
	"strconv"

	"github.com/go-openapi/spec"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/xeipuuv/gojsonschema"
)

const metadataFolder = "contract-metadata"
const metadataFile = "metadata.json"

var logger = shim.NewLogger("contractapi/metadata.go")

// Helper for OS testing
type osExc interface {
	Executable() (string, error)
}

type osExcStr struct{}

func (o osExcStr) Executable() (string, error) {
	return os.Executable()
}

var osHelper osExc = osExcStr{}

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
	Description string           `json:"description,omitempty"`
	Name        string           `json:"name"`
	Required    bool             `json:"required,omitempty"`
	Schema      spec.SchemaProps `json:"schema"`
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
	Info         spec.InfoProps        `json:"info,omitempty"`
	Name         string                `json:"name"`
	Transactions []TransactionMetadata `json:"transactions"`
}

// AssetMetadata description of an asset
type AssetMetadata struct {
	Name       string              `json:"name"`
	Properties []ParameterMetadata `json:"properties"`
}

// ComponentMetadata does something
type ComponentMetadata struct {
	Schemas map[string]AssetMetadata `json:"schemas,omitempty"`
}

// ContractChaincodeMetadata describes a chaincode made using the contractapi
type ContractChaincodeMetadata struct {
	Info       spec.InfoProps              `json:"info,omitempty"`
	Contracts  map[string]ContractMetadata `json:"contracts"`
	Components ComponentMetadata           `json:"components"`
}

func (ccm *ContractChaincodeMetadata) append(source ContractChaincodeMetadata) {
	if reflect.DeepEqual(ccm.Info, spec.InfoProps{}) {
		ccm.Info = source.Info
	}

	if len(ccm.Contracts) == 0 {
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
	fileMetadata.Contracts = make(map[string]ContractMetadata)

	ex, execErr := osHelper.Executable()
	if execErr != nil {
		logger.Error(fmt.Sprintf("Error finding location of running executable. Defaulting to Reflected metadata. %s", execErr.Error()))

		return fileMetadata
	}
	exPath := filepath.Dir(ex)
	metadataPath := filepath.Join(exPath, metadataFolder, metadataFile)

	_, err := os.Stat(metadataPath)

	if os.IsNotExist(err) {
		logger.Info("No metadata file supplied")
		return fileMetadata
	}

	metadataBytes, err := ioutil.ReadFile(metadataPath)

	if err != nil {
		panic(fmt.Sprintf("Failed to generate metadata. Could not read file %s. %s", metadataPath, err))
	}

	schemaLoader := gojsonschema.NewBytesLoader([]byte(GetJSONSchema()))
	metadataLoader := gojsonschema.NewBytesLoader(metadataBytes)

	result, err := gojsonschema.Validate(schemaLoader, metadataLoader)

	if result == nil {
		panic(fmt.Sprintf("Error validating metadata file against schema. Is the file valid JSON?"))
	}

	if !result.Valid() {
		var errors string

		for index, desc := range result.Errors() {
			errors = errors + "\n" + strconv.Itoa(index+1) + ".\t" + desc.String()
		}

		panic(fmt.Sprintf("Failed to generate metadata. Given file did not match schema: %s", errors))
	}

	json.Unmarshal(metadataBytes, &fileMetadata)

	return fileMetadata
}
