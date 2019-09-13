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

// ContractInterface defines functions a valid contract should have. Contracts to
// be used in chaincode must implement this interface.
type ContractInterface interface {
	// GetVersion returns the the version of the contract. If the function returns a
	// blank string then "latest" is used for the version in the metadata.
	GetVersion() string

	// GetUnknownTransaction returns the unknown function to be used for a contract.
	// When the contract is used in creating a new chaincode this function is called
	// and the unknown transaction returned is stored. The unknown function is then
	// called in cases where an unknown function name is passed for a call to the
	// contract via Init/Invoke of the chaincode. If nil is returned the
	// chaincode uses its default handling for unknown function names
	GetUnknownTransaction() interface{}

	// GetBeforeTransaction returns the before function to be used for a contract.
	// When the contract is used in creating a new chaincode this function is called
	// and the before transaction returned is stored. The before function is then
	// called before the named function on each Init/Invoke of that contract via the
	// chaincode. When called the before function is passed no extra args, only the
	// the transaction context (if specified to take it). If nil is returned
	// then no before function is called on Init/Invoke.
	GetBeforeTransaction() interface{}

	// GetAfterTransaction returns the after function to be used for a contract.
	// When the contract is used in creating a new chaincode this function is called
	// and the after transaction returned is stored. The after function is then
	// called after the named function on each Init/Invoke of that contract via the
	// chaincode. When called the after function is passed the returned value of the
	// named function and the transaction context (if the function takes the transaction
	// context). If nil is returned then no after function is called on Init/
	// Invoke.
	GetAfterTransaction() interface{}

	// GetName returns the name of the contract. When the contract is used
	// in creating a new chaincode this function is called and the name returned
	// is then used to identify the contract within the chaincode on Init/Invoke calls.
	// This function can return a blank string but this is undefined behaviour.
	GetName() string

	// GetTransactionContextHandler returns the TransactionContextInterface that is
	// used by the functions of the contract. When the contract is used in creating
	// a new chaincode this function is called and the transaction context returned
	// is stored. When the chaincode is called via Init/Invoke a transaction context
	// of the stored type is created and sent as a parameter to the named contract
	// function (and before/after and unknown functions) if the function requires the
	// context in its list of parameters.
	GetTransactionContextHandler() TransactionContextInterface
}

// Contract defines functions for setting and getting before, after and unknown transactions
// and name. Can be embedded in user structs to quickly ensure their definition meets
// the ContractInterface.
type Contract struct {
	version            string
	unknownTransaction interface{}
	beforeTransaction  interface{}
	afterTransaction   interface{}
	contextHandler     TransactionContextInterface
	name               string
}

// SetVersion sets the version of the contract
func (c *Contract) SetVersion(version string) {
	c.version = version
}

// GetVersion returns the version of the contract
func (c *Contract) GetVersion() string {
	return c.version
}

// SetUnknownTransaction sets function for contract's unknownTransaction.
func (c *Contract) SetUnknownTransaction(fn interface{}) {
	c.unknownTransaction = fn
}

// GetUnknownTransaction returns the current set unknownTransaction, may be nil
func (c *Contract) GetUnknownTransaction() interface{} {
	return c.unknownTransaction
}

// SetBeforeTransaction sets function for contract's beforeTransaction.
func (c *Contract) SetBeforeTransaction(fn interface{}) {
	c.beforeTransaction = fn
}

// GetBeforeTransaction returns the current set beforeTransaction, may be nil
func (c *Contract) GetBeforeTransaction() interface{} {
	return c.beforeTransaction
}

// SetAfterTransaction sets function for contract's afterTransaction.
func (c *Contract) SetAfterTransaction(fn interface{}) {
	c.afterTransaction = fn
}

// GetAfterTransaction returns the current set afterTransaction, may be nil
func (c *Contract) GetAfterTransaction() interface{} {
	return c.afterTransaction
}

// SetName sets the name for the contract.
func (c *Contract) SetName(name string) {
	c.name = name
}

// GetName returns the current set name for
// the contract.
func (c *Contract) GetName() string {
	return c.name
}

// SetTransactionContextHandler sets the transaction context type to be used for
// the contract.
func (c *Contract) SetTransactionContextHandler(ctx TransactionContextInterface) {
	c.contextHandler = ctx
}

// GetTransactionContextHandler returns the current transaction context set for
// the contract.
func (c *Contract) GetTransactionContextHandler() TransactionContextInterface {
	if c.contextHandler == nil {
		return new(TransactionContext)
	}

	return c.contextHandler
}
