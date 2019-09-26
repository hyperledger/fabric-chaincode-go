// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/shim/internal/mock"
	peerpb "github.com/hyperledger/fabric-protos-go/peer"

	"github.com/stretchr/testify/assert"
)

func TestStart(t *testing.T) {

	var tests = []struct {
		name             string
		envVars          map[string]string
		peerAddress      string
		chaincodeAddress string
		streamGetter     func(name string) (ClientStream, error)
		cc               Chaincode
		expectedErr      string
	}{
		{
			name:        "Missing Chaincode ID",
			expectedErr: "'CORE_CHAINCODE_ID_NAME' must be set",
		},
		{
			name: "Missing Peer Address",
			envVars: map[string]string{
				"CORE_CHAINCODE_ID_NAME": "cc",
			},
			expectedErr: "flag 'peer.address' must be set",
		},
		{
			name: "TLS Not Set",
			envVars: map[string]string{
				"CORE_CHAINCODE_ID_NAME": "cc",
			},
			peerAddress: "127.0.0.1:12345",
			expectedErr: "'CORE_PEER_TLS_ENABLED' must be set to 'true' or 'false'",
		},
		{
			name: "Connection Error",
			envVars: map[string]string{
				"CORE_CHAINCODE_ID_NAME": "cc",
				"CORE_PEER_TLS_ENABLED":  "false",
			},
			peerAddress: "127.0.0.1:12345",
			expectedErr: `connection error: desc = "transport: error while dialing: dial tcp 127.0.0.1:12345: connect: connection refused"`,
		},
		{
			name: "Chat - Nil Message",
			envVars: map[string]string{
				"CORE_CHAINCODE_ID_NAME": "cc",
				"CORE_PEER_TLS_ENABLED":  "false",
			},
			peerAddress: "127.0.0.1:12345",
			streamGetter: func(name string) (ClientStream, error) {
				stream := &mock.ClientStream{}
				return stream, nil
			},
			expectedErr: "received nil message, ending chaincode stream",
		},
		{
			name: "Chat - EOF",
			envVars: map[string]string{
				"CORE_CHAINCODE_ID_NAME": "cc",
				"CORE_PEER_TLS_ENABLED":  "false",
			},
			peerAddress: "127.0.0.1:12345",
			streamGetter: func(name string) (ClientStream, error) {
				stream := &mock.ClientStream{}
				stream.RecvReturns(nil, io.EOF)
				return stream, nil
			},
			expectedErr: "received EOF, ending chaincode stream",
		},
		{
			name: "Chat - Recv Error",
			envVars: map[string]string{
				"CORE_CHAINCODE_ID_NAME": "cc",
				"CORE_PEER_TLS_ENABLED":  "false",
			},
			peerAddress: "127.0.0.1:12345",
			streamGetter: func(name string) (ClientStream, error) {
				stream := &mock.ClientStream{}
				stream.RecvReturns(nil, errors.New("recvError"))
				return stream, nil
			},
			expectedErr: "receive failed: recvError",
		},
		{
			name: "Chat - Not Ready",
			envVars: map[string]string{
				"CORE_CHAINCODE_ID_NAME": "cc",
				"CORE_PEER_TLS_ENABLED":  "false",
			},
			peerAddress: "127.0.0.1:12345",
			streamGetter: func(name string) (ClientStream, error) {
				stream := &mock.ClientStream{}
				stream.RecvReturnsOnCall(
					0,
					&peerpb.ChaincodeMessage{
						Type: peerpb.ChaincodeMessage_READY,
						Txid: "txid",
					},
					nil,
				)
				return stream, nil
			},
			expectedErr: "error handling message: [txid] Chaincode h cannot handle message (READY) while in state: created",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}
			peerAddress = &test.peerAddress
			streamGetter = test.streamGetter
			err := Start(test.cc)
			assert.EqualError(t, err, test.expectedErr)
		})
	}

}

func TestChaincodeServerStart(t *testing.T) {

	var tests = []struct {
		name         string
		ccsrv        ChaincodeServer
		streamGetter func(name string) (ClientStream, error)
		expectedErr  string
		containsErr  string
	}{
		{
			name:        "Missing Chaincode ID",
			ccsrv:       ChaincodeServer{},
			expectedErr: "ccid must be specified",
		},
		{
			name:        "Missing Peer Address",
			ccsrv:       ChaincodeServer{CCID: "cc"},
			expectedErr: "address must be specified",
		},
		{
			name:        "Missing Peer Address and Chaincode Address",
			ccsrv:       ChaincodeServer{CCID: "cc", Address: "127.0.0.1:12345"},
			expectedErr: "chaincode must be specified",
		},
		{
			name:        "Badly formed chaincode server address",
			ccsrv:       ChaincodeServer{CCID: "cc", Address: "127.0.0.1", CC: &mockChaincode{}, TLSProps: TLSProperties{Disabled: true}},
			expectedErr: "listen tcp: address 127.0.0.1: missing port in address",
		},
		{
			name:        "Bad host in chaincode server address",
			ccsrv:       ChaincodeServer{CCID: "cc", Address: "__badhost__:12345", CC: &mockChaincode{}, TLSProps: TLSProperties{Disabled: true}},
			containsErr: "listen tcp: lookup __badhost__",
		},
		// Basic TLS tests, path tests
		{
			name:        "TLS enabled but key path not provided",
			ccsrv:       ChaincodeServer{CCID: "cc", Address: "host:12345", CC: &mockChaincode{}, TLSProps: TLSProperties{Disabled: false}},
			containsErr: "key not provided",
		},
		{
			name:        "TLS enabled but cert path not provided",
			ccsrv:       ChaincodeServer{CCID: "cc", Address: "host:12345", CC: &mockChaincode{}, TLSProps: TLSProperties{Disabled: false, Key: []byte("key")}},
			containsErr: "cert not provided",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			err := test.ccsrv.Start()
			if test.expectedErr != "" {
				assert.EqualError(t, err, test.expectedErr)
			} else if test.containsErr != "" {
				assert.Contains(t, err.Error(), test.containsErr)
			}
		})
	}

}
