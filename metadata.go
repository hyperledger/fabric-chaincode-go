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
	"path/filepath"
	"sort"
	"strconv"

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
	return `{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"$id": "http://example.com/root.json",
		"type": "object",
		"title": "Hyperledger Fabric Contract Definition JSON Schema",
		"required": [
			"info",
			"contracts"
		],
		"properties": {
			"info": {
				"$ref": "#/definitions/info"
			},
			"contracts": {
				"type": "array",
				"items": {
					"$ref": "#/definitions/contract"
				}
			},
			"components": {
				"$ref": "#/definitions/components"
			}
		},
		"definitions": {
			"info": {
				"type": "object",
				"description": "General information about the API.",
				"required": [
					"version",
					"title"
				],
				"properties": {
					"title": {
						"type": "string",
						"description": "A unique and precise title of the API."
					},
					"version": {
						"type": "string",
						"description": "A semantic version number of the API."
					},
					"description": {
						"type": "string",
						"description": "A longer description of the API. Should be different from the title.  GitHub Flavored Markdown is allowed."
					},
					"termsOfService": {
						"type": "string",
						"description": "The terms of service for the API."
					},
					"contact": {
						"$ref": "#/definitions/contact"
					},
					"license": {
						"$ref": "#/definitions/license"
					}
				}
			},
			"contact": {
				"type": "object",
				"description": "Contact information for the owners of the API.",
				"properties": {
					"name": {
						"type": "string",
						"description": "The identifying name of the contact person/organization."
					},
					"url": {
						"type": "string",
						"description": "The URL pointing to the contact information.",
						"format": "uri"
					},
					"email": {
						"type": "string",
						"description": "The email address of the contact person/organization.",
						"format": "email"
					}
				}
			},
			"license": {
				"type": "object",
				"required": [
					"name"
				],
				"additionalProperties": false,
				"properties": {
					"name": {
						"type": "string",
						"description": "The name of the license type. It's encouraged to use an OSI compatible license."
					},
					"url": {
						"type": "string",
						"description": "The URL pointing to the license.",
						"format": "uri"
					}
				}
			},
			"contract": {
				"type": "object",
				"description": "",
				"required": [
					"namespace",
					"transactions"
				],
				"properties": {
					"info": {
						"$ref": "#/definitions/info"
					},
					"namespace": {
						"type": "string",
						"description": "A unique and precise title of the API."
					},
					"transactions": {
						"type": "array",
						"items": {
							"$ref": "#/definitions/transaction"
						}
					}
				}
			},
			"asset": {
				"type": "object",
				"description": "A complex type used in a domain",
				"required": [
					"name",
					"properties"
				],
				"properties": {
					"name": {
						"type": "string"
					},
					"properties": {
						"$ref": "#/definitions/parametersList"
					}
				}
			},
			"parametersList": {
				"type": "array",
				"description": "The parameters needed to send a valid API call.",
				"additionalItems": false,
				"items": {
					"oneOf": [
						{
							"$ref": "#/definitions/parameter"
						},
						{
							"$ref": "#/definitions/jsonReference"
						}
					]
				},
				"uniqueItems": true
			},
			"transaction": {
				"type": "object",
				"description": "single transaction specification",
				"required": [
					"transactionId"
				],
				"properties": {
					"transactionId": {
						"type": "string",
						"description": "name of the transaction "
					},
					"tag": {
						"type": "array",
						"items": {
							"type": "string",
							"desciprition": "free format tags"
						}
					},
					"parameters": {
						"$ref": "#/definitions/parametersList"
					},
					"returns": {
						"oneOf": [
							{
								"$ref": "#/definitions/parameter"
							},
							{
								"$ref": "#/definitions/parametersList"
							}
						]
					}
				}
			},
			"parameter": {
				"type": "object",
				"required": [
					"name",
					"schema"
				],
				"properties": {
					"description": {
						"type": "string",
						"description": "A brief description of the parameter. This could contain examples of use.  GitHub Flavored Markdown is allowed."
					},
					"name": {
						"type": "string",
						"description": "The name of the parameter."
					},
					"required": {
						"type": "boolean",
						"description": "Determines whether or not this parameter is required or optional.",
						"default": false
					},
					"schema": {
						"$ref": "#/definitions/schema"
					}
				},
				"additionalProperties": false
			},
			"jsonReference": {
				"type": "object",
				"required": [
					"$ref"
				],
				"additionalProperties": false,
				"properties": {
					"$ref": {
						"type": "string"
					}
				}
			},
			"schema": {
				"type": "object",
				"description": "A deterministic version of a JSON Schema object.",
				"properties": {
					"$ref": {
						"type": "string"
					},
					"format": {
						"type": "string"
					},
					"title": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/title"
					},
					"description": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/description"
					},
					"default": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/default"
					},
					"multipleOf": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/multipleOf"
					},
					"maximum": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/maximum"
					},
					"exclusiveMaximum": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/exclusiveMaximum"
					},
					"minimum": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/minimum"
					},
					"exclusiveMinimum": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/exclusiveMinimum"
					},
					"maxLength": {
						"$ref": "http://json-schema.org/draft-04/schema#/definitions/positiveInteger"
					},
					"minLength": {
						"$ref": "http://json-schema.org/draft-04/schema#/definitions/positiveIntegerDefault0"
					},
					"pattern": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/pattern"
					},
					"maxItems": {
						"$ref": "http://json-schema.org/draft-04/schema#/definitions/positiveInteger"
					},
					"minItems": {
						"$ref": "http://json-schema.org/draft-04/schema#/definitions/positiveIntegerDefault0"
					},
					"uniqueItems": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/uniqueItems"
					},
					"maxProperties": {
						"$ref": "http://json-schema.org/draft-04/schema#/definitions/positiveInteger"
					},
					"minProperties": {
						"$ref": "http://json-schema.org/draft-04/schema#/definitions/positiveIntegerDefault0"
					},
					"required": {
						"$ref": "http://json-schema.org/draft-04/schema#/definitions/stringArray"
					},
					"enum": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/enum"
					},
					"additionalProperties": {
						"anyOf": [
							{
								"$ref": "#/definitions/schema"
							},
							{
								"type": "boolean"
							}
						],
						"default": {}
					},
					"type": {
						"$ref": "http://json-schema.org/draft-04/schema#/properties/type"
					},
					"items": {
						"anyOf": [
							{
								"$ref": "#/definitions/schema"
							},
							{
								"type": "array",
								"minItems": 1,
								"items": {
									"$ref": "#/definitions/schema"
								}
							}
						],
						"default": {}
					},
					"allOf": {
						"type": "array",
						"minItems": 1,
						"items": {
							"$ref": "#/definitions/schema"
						}
					},
					"properties": {
						"type": "object",
						"additionalProperties": {
							"$ref": "#/definitions/schema"
						},
						"default": {}
					},
					"discriminator": {
						"type": "string"
					},
					"readOnly": {
						"type": "boolean",
						"default": false
					},
					"example": {}
				},
				"additionalProperties": false
			},
			"components": {
				"type": "object",
				"properties": {
					"schemas": {
						"type": "object",
						"patternProperties": {
							"^.*$": {
								"$ref": "#/definitions/asset"
							}
						}
					}
				}
			}
		}
	}`
}

// LicenseMetadata details for the license of the chaincode
type LicenseMetadata struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

// ContactMetadata information for owners of the chaincode
type ContactMetadata struct {
	Email string `json:"email,omitempty"`
	Name  string `json:"name,omitempty"`
	URL   string `json:"url,omitempty"`
}

// InfoMetadata general information about the API.
type InfoMetadata struct {
	Contact        *ContactMetadata `json:"contact,omitempty"`
	Description    string           `json:"description,omitempty"`
	License        *LicenseMetadata `json:"license,omitempty"`
	TermsOfService string           `json:"termsOfService,omitempty"`
	Title          string           `json:"title"`
	Version        string           `json:"version"`
}

// SchemaOrBoolean represents a schema or boolean value
type SchemaOrBoolean struct {
	Boolean bool
	Schema  *Schema
}

// MarshalJSON if Schema not nil returns boolean value as bytes else JSON bytes for Schema
func (s SchemaOrBoolean) MarshalJSON() ([]byte, error) {
	if s.Schema == nil {
		return json.Marshal(s.Boolean)
	}

	return json.Marshal(s.Schema)
}

// UnmarshalJSON converts JSON back into SchemaOrBoolean
func (s *SchemaOrBoolean) UnmarshalJSON(data []byte) error {
	var schema Schema

	err := json.Unmarshal(data, &schema)

	if err == nil {
		s.Schema = &schema
		return nil
	}

	var iface interface{}

	err = json.Unmarshal(data, &iface)

	if err != nil {
		return err
	}

	boo, ok := iface.(bool)

	if !ok {
		return errors.New("Can only unmarshal to SchemaOrBoolean if value is boolean or Schema format")
	}

	s.Boolean = boo

	return nil
}

// SchemaOrArray represents a schema or an array of schemas
type SchemaOrArray struct {
	Schema      *Schema
	SchemaArray []*Schema
}

// MarshalJSON if Schema not nil returns Schema value as bytes else JSON bytes for SchemaArray
func (s SchemaOrArray) MarshalJSON() ([]byte, error) {
	if s.Schema == nil {
		return json.Marshal(s.SchemaArray)
	}

	return json.Marshal(s.Schema)
}

// UnmarshalJSON converts JSON back into SchemaOrArray
func (s *SchemaOrArray) UnmarshalJSON(data []byte) error {
	var schema Schema

	err := json.Unmarshal(data, &schema)

	if err == nil {
		s.Schema = &schema
		return nil
	}

	var iface []*Schema

	err = json.Unmarshal(data, &iface)

	if err != nil {
		return errors.New("Can only unmarshal to SchemaOrArray if value is Schema format or array of Schema formats")
	}

	s.SchemaArray = iface

	return nil
}

// StringOrArray represents a string or an array of strings
type StringOrArray []string

// MarshalJSON converts single string in array to just string as JSON else JSON array for more than 1
func (s StringOrArray) MarshalJSON() ([]byte, error) {
	if len(s) == 1 {
		return json.Marshal([]string(s)[0])
	}
	return json.Marshal([]string(s))
}

// UnmarshalJSON converts JSON array/string to StringOrArray object
func (s *StringOrArray) UnmarshalJSON(data []byte) error {

	var arr []string

	err := json.Unmarshal(data, &arr)

	if err == nil {
		*s = StringOrArray(arr)
		return nil
	}

	var iface interface{}

	err = json.Unmarshal(data, &iface)

	if err != nil {
		return err
	}

	str, ok := iface.(string)

	if !ok {
		return errors.New("Can only unmarshal to StringOrArray if value is []string or string")
	}

	*s = []string{str}

	return nil
}

// Schema a deterministic version of a JSON Schema object.
type Schema struct {
	AdditionalProperties *SchemaOrBoolean   `json:"additionalProperties,omitempty"`
	AllOf                []*Schema          `json:"allOf,omitempty"`
	Default              interface{}        `json:"default,omitempty"`
	Description          string             `json:"description,omitempty"`
	Discriminator        string             `json:"discriminator,omitempty"`
	Enum                 []interface{}      `json:"enum,omitempty"`
	Example              interface{}        `json:"example,omitempty"`
	ExclusiveMaximum     bool               `json:"exclusiveMaximum,omitempty"`
	ExclusiveMinimum     bool               `json:"exclusiveMinimum,omitempty"`
	Format               string             `json:"format,omitempty"`
	Items                *SchemaOrArray     `json:"items,omitempty"`
	MaxItems             uint               `json:"maxItems,omitempty"`
	MaxLength            uint               `json:"maxLength,omitempty"`
	MaxProperties        uint               `json:"maxProperties,omitempty"`
	Maximum              float64            `json:"maximum,omitempty"`
	MinItems             uint               `json:"minItems,omitempty"`
	MinLength            uint               `json:"minLength,omitempty"`
	MinProperties        uint               `json:"minProperties,omitempty"`
	Minimum              float64            `json:"minimum,omitempty"`
	MultipleOf           float64            `json:"multipleOf,omitempty"`
	Pattern              string             `json:"pattern,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	ReadOnly             bool               `json:"readOnly,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Title                string             `json:"title,omitempty"`
	Type                 StringOrArray      `json:"type,omitempty"`
	UniqueItems          bool               `json:"uniqueItems,omitempty"`
}

// ParameterMetadata details about a parameter used for a transaction
type ParameterMetadata struct {
	Description string `json:"description,omitempty"`
	Name        string `json:"name"`
	Required    bool   `json:"required,omitempty"`
	Schema      Schema `json:"schema"`
}

// TransactionMetadata contains information on what makes up a transaction
type TransactionMetadata struct {
	Parameters    []ParameterMetadata `json:"parameters,omitempty"`
	Returns       []ParameterMetadata `json:"returns,omitempty"`
	Tag           []string            `json:"tag,omitempty"`
	TransactionID string              `json:"transactionId"`
}

// ContractMetadata contains information about what makes up a contract
type ContractMetadata struct {
	Info         InfoMetadata          `json:"info,omitempty"`
	Namespace    string                `json:"namespace"`
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
	Info       InfoMetadata       `json:"info,omitempty"`
	Contracts  []ContractMetadata `json:"contracts"`
	Components ComponentMetadata  `json:"components"`
}

func generateMetadata(cc contractChaincode) string {
	ccMetadata := new(ContractChaincodeMetadata)

	ex, execErr := osHelper.Executable()
	if execErr != nil {
		logger.Error(fmt.Sprintf("Error finding location of running executable. Defaulting to Reflected metadata. %s", execErr.Error()))

	}
	exPath := filepath.Dir(ex)

	metadataPath := filepath.Join(exPath, metadataFolder, metadataFile)

	_, err := os.Stat(metadataPath)

	if execErr != nil || os.IsNotExist(err) {
		for key, contract := range cc.contracts {
			contractMetadata := ContractMetadata{}
			contractMetadata.Namespace = key

			for key, fn := range contract.functions {
				transactionMetadata := TransactionMetadata{}
				transactionMetadata.TransactionID = key

				if fn.params.context != nil {
					schema := Schema{}
					schema.Type = []string{"object"}
					schema.Format = fn.params.context.String()

					param := ParameterMetadata{}
					param.Name = "ctx"
					param.Required = true
					param.Schema = schema

					transactionMetadata.Parameters = append(transactionMetadata.Parameters, param)
				}

				for index, field := range fn.params.fields {
					schema, err := getSchema(field)

					if err != nil {
						panic(fmt.Sprintf("Failed to generate metadata. Invalid function parameter type. %s", err))
					}

					param := ParameterMetadata{}
					param.Name = fmt.Sprintf("param%d", index)
					param.Required = true
					param.Schema = *schema

					transactionMetadata.Parameters = append(transactionMetadata.Parameters, param)
				}

				if fn.returns.success != nil {
					schema, err := getSchema(fn.returns.success)

					if err != nil {
						panic(fmt.Sprintf("Failed to generate metadata. Invalid function success return type. %s", err))
					}

					param := ParameterMetadata{}
					param.Name = "success"
					param.Schema = *schema

					transactionMetadata.Returns = append(transactionMetadata.Returns, param)
				}

				if fn.returns.error {
					schema := Schema{}
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
				return contractMetadata.Transactions[i].TransactionID < contractMetadata.Transactions[j].TransactionID
			})

			ccMetadata.Contracts = append(ccMetadata.Contracts, contractMetadata)
		}

		sort.Slice(ccMetadata.Contracts, func(i, j int) bool {
			return ccMetadata.Contracts[i].Namespace < ccMetadata.Contracts[j].Namespace
		})
	} else {
		metadataBytes, err := ioutil.ReadFile(metadataPath)

		if err != nil {
			panic(fmt.Sprintf("Failed to generate metadata. Could not read file %s. %s", metadataPath, err))
		}

		schemaLoader := gojsonschema.NewBytesLoader([]byte(GetJSONSchema()))
		metadataLoader := gojsonschema.NewBytesLoader(metadataBytes)

		result, _ := gojsonschema.Validate(schemaLoader, metadataLoader)

		if !result.Valid() {
			var errors string

			for index, desc := range result.Errors() {
				errors = errors + "\n" + strconv.Itoa(index+1) + ".\t" + desc.String()
			}

			panic(fmt.Sprintf("Failed to generate metadata. Given file did not match schema: %s", errors))
		}

		json.Unmarshal(metadataBytes, ccMetadata)
	}
	ccMetadataJSON, _ := json.Marshal(ccMetadata)

	return string(ccMetadataJSON)
}
