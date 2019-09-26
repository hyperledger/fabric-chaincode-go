// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim

import (
	"crypto/tls"

	"github.com/hyperledger/fabric-chaincode-go/shim/internal"
	pb "github.com/hyperledger/fabric-protos-go/peer"

	"google.golang.org/grpc/keepalive"
)

// ChaincodeServer encapsulates basic properties needed for a chaincode server
type ChaincodeServer struct {
	Name   string
	CC     Chaincode
	Tls    *tls.Config
	KaOpts keepalive.ServerParameters
}

// Connect the bidi stream entry point called by chaincode to register with the Peer.
func (cs *ChaincodeServer) Connect(stream pb.Chaincode_ConnectServer) error {
	return chatWithPeer(cs.Name, stream, cs.CC)
}

// Start the server
func (cs *ChaincodeServer) Start(address string) error {
	// create listener and grpc server
	server, err := internal.NewServer(address, cs.Tls, cs.KaOpts)
	if err != nil {
		return err
	}

	// register the server with grpc ...
	pb.RegisterChaincodeServer(server.Server, cs)

	// ... and start
	return server.Start()
}
