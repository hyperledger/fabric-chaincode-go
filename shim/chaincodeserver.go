// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim

import (
	"errors"

	"github.com/hyperledger/fabric-chaincode-go/shim/internal"
	pb "github.com/hyperledger/fabric-protos-go/peer"
)

// ChaincodeServer encapsulates basic properties needed for a chaincode server
type ChaincodeServer struct {
	Name    string
	Address string
	CC      Chaincode
}

// Connect the bidi stream entry point called by chaincode to register with the Peer.
func (cs *ChaincodeServer) Connect(stream pb.Chaincode_ConnectServer) error {
	return chatWithPeer(cs.Name, stream, cs.CC)
}

// Start the server
func (cs *ChaincodeServer) Start() error {
	if cs.Name == "" {
		return errors.New("name must be specified")
	}

	if cs.Address == "" {
		return errors.New("address must be specified")
	}

	if cs.CC == nil {
		return errors.New("chaincode must be specified")
	}

	// create listener and grpc server
	server, err := internal.NewServer(cs.Address, nil)
	if err != nil {
		return err
	}

	// register the server with grpc ...
	pb.RegisterChaincodeServer(server.Server, cs)

	// ... and start
	return server.Start()
}
