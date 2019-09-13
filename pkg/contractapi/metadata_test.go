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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
)

// ================================
// Helpers
// ================================

type ioUtilReadFileTestStr struct{}

func (io ioUtilReadFileTestStr) ReadFile(filename string) ([]byte, error) {
	return nil, errors.New("some error")
}

type osExcTestStr struct{}

func (o osExcTestStr) Executable() (string, error) {
	return "", errors.New("some error")
}

func (o osExcTestStr) Stat(name string) (os.FileInfo, error) {
	return nil, nil
}

type osStatTestStr struct{}

func (o osStatTestStr) Executable() (string, error) {
	return "", nil
}

func (o osStatTestStr) Stat(name string) (os.FileInfo, error) {
	return os.Stat("some bad file")
}

// ================================
// Tests
// ================================

func TestGetJSONSchema(t *testing.T) {
	// should read file
	file, _ := readLocalFile("schema/schema.json")

	assert.Equal(t, GetJSONSchema(), string(file), "should retrieve schema file")

	// should panic when can't read file
	oldIoUtilHelper := ioutilHelper
	ioutilHelper = ioUtilReadFileTestStr{}

	assert.PanicsWithValue(t, "Unable to read JSON schema. Error: some error", func() { GetJSONSchema() }, "should panic when readLocalFile errors")

	ioutilHelper = oldIoUtilHelper
}

func TestAppend(t *testing.T) {
	var ccm ContractChaincodeMetadata

	source := ContractChaincodeMetadata{}
	source.Info = spec.Info{}
	source.Info.Title = "A title"
	source.Info.Version = "Some version"

	someContract := ContractMetadata{}
	someContract.Name = "some contract"

	source.Contracts = make(map[string]ContractMetadata)
	source.Contracts["some contract"] = someContract

	someComponent := ObjectMetadata{}

	source.Components = ComponentMetadata{}
	source.Components.Schemas = make(map[string]ObjectMetadata)
	source.Components.Schemas["some component"] = someComponent

	// should use the source info when info is blank
	ccm = ContractChaincodeMetadata{}
	ccm.append(source)

	assert.Equal(t, ccm.Info, source.Info, ccm.Info, "should have used source info when info blank")

	// should use own info when info set
	ccm = ContractChaincodeMetadata{}
	ccm.Info = spec.Info{}
	ccm.Info.Title = "An existing title"
	ccm.Info.Version = "Some existing version"

	someInfo := ccm.Info

	ccm.append(source)

	assert.Equal(t, someInfo, ccm.Info, "should have used own info when info existing")
	assert.NotEqual(t, source.Info, ccm.Info, "should not use source info when info exists")

	// should use the source contract when contract is 0 length and nil
	ccm = ContractChaincodeMetadata{}
	ccm.append(source)

	assert.Equal(t, source.Contracts, ccm.Contracts, "should have used source info when contract 0 length map")

	// should use the source contract when contract is 0 length and not nil
	ccm = ContractChaincodeMetadata{}
	ccm.Contracts = make(map[string]ContractMetadata)
	ccm.append(source)

	assert.Equal(t, source.Contracts, ccm.Contracts, "should have used source info when contract 0 length map")

	// should use own contract when contract greater than 1
	anotherContract := ContractMetadata{}
	anotherContract.Name = "some contract"

	ccm = ContractChaincodeMetadata{}
	ccm.Contracts = make(map[string]ContractMetadata)
	ccm.Contracts["another contract"] = anotherContract

	contractMap := ccm.Contracts

	assert.Equal(t, contractMap, ccm.Contracts, "should have used own contracts when contracts existing")
	assert.NotEqual(t, source.Contracts, ccm.Contracts, "should not have used source contracts when existing contracts")

	// should use source components when components is empty
	ccm = ContractChaincodeMetadata{}
	ccm.append(source)

	assert.Equal(t, ccm.Components, source.Components, "should use sources components")

	// should use own components when components is empty
	anotherComponent := ObjectMetadata{}

	ccm = ContractChaincodeMetadata{}
	ccm.Components = ComponentMetadata{}
	ccm.Components.Schemas = make(map[string]ObjectMetadata)
	ccm.Components.Schemas["another component"] = anotherComponent

	ccmComponent := ccm.Components

	ccm.append(source)

	assert.Equal(t, ccmComponent, ccm.Components, "should have used own components")
	assert.NotEqual(t, source.Components, ccm.Components, "should not be same as source components")
}

func TestReadMetadataFile(t *testing.T) {
	var filepath string
	var metadataBytes []byte

	oldOsHelper := osHelper

	// Should return empty metadata when execute not found
	osHelper = osExcTestStr{}

	assert.Equal(t, ContractChaincodeMetadata{}, readMetadataFile(), "should return blank metadata when cannot read file due to exec error")

	osHelper = oldOsHelper

	// Should return empty metadata when file does not exist
	osHelper = osStatTestStr{}

	assert.Equal(t, ContractChaincodeMetadata{}, readMetadataFile(), "should return blank metadata when cannot read file as does not exist")

	osHelper = oldOsHelper

	// Should panic when cannot read file but it exists
	filepath = createMetadataJSONFile([]byte("some file contents"), 0000)
	_, readfileErr := ioutil.ReadFile(filepath)
	assert.PanicsWithValue(t, fmt.Sprintf("Failed to get existing metadata. Could not read file %s. %s", filepath, readfileErr), func() { readMetadataFile() }, "should panic when cannot read file but it exists")
	cleanupMetadataJSONFile()

	// should panic when file does not match schema
	metadataBytes = []byte("{\"some\":\"json\"}")

	createMetadataJSONFile(metadataBytes, os.ModePerm)

	assert.PanicsWithValue(t, fmt.Sprintf("Failed to get existing metadata. Given file did not match schema: 1. (root): info is required\n2. (root): contracts is required"), func() { readMetadataFile() }, "should panic when file does not meet schema")
	cleanupMetadataJSONFile()

	// should use metadata file data
	metadataBytes = []byte("{\"info\":{\"title\":\"my contract\",\"version\":\"0.0.1\"},\"contracts\":{},\"components\":{}}")
	contractChaincodeMetadata := ContractChaincodeMetadata{}

	json.Unmarshal(metadataBytes, &contractChaincodeMetadata)

	createMetadataJSONFile(metadataBytes, os.ModePerm)
	assert.Equal(t, contractChaincodeMetadata, readMetadataFile(), "should return metadata from file")
	cleanupMetadataJSONFile()
}
