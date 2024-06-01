// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim

import (
	"fmt"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim/internal/mock"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"

	"github.com/stretchr/testify/assert"
)

//go:generate counterfeiter -o internal/mock/peer_chaincode_stream.go --fake-name PeerChaincodeStream . peerChaincodeStream

//lint:ignore U1000 Required to avoid circular dependency with mock
type peerChaincodeStream interface{ PeerChaincodeStream }

//go:generate counterfeiter -o internal/mock/client_stream.go --fake-name ClientStream . clientStream

//lint:ignore U1000 Required to avoid circular dependency with mock
type clientStream interface{ ClientStream }

type mockChaincode struct {
	initCalled   bool
	invokeCalled bool
}

func (mcc *mockChaincode) Init(stub ChaincodeStubInterface) *peer.Response {
	mcc.initCalled = true
	return Success(nil)
}

func (mcc *mockChaincode) Invoke(stub ChaincodeStubInterface) *peer.Response {
	mcc.invokeCalled = true
	return Success(nil)
}

func TestNewHandler_CreatedState(t *testing.T) {
	t.Parallel()

	chatStream := &mock.PeerChaincodeStream{}
	cc := &mockChaincode{}

	expected := &Handler{
		chatStream:       chatStream,
		cc:               cc,
		responseChannels: map[string]chan *peer.ChaincodeMessage{},
		state:            created,
	}

	handler := newChaincodeHandler(chatStream, cc)
	if handler == nil {
		t.Fatal("Handler should not be nil")
	}
	assert.Equal(t, expected, handler)
}

func TestHandlerState(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name        string
		state       state
		msg         *peer.ChaincodeMessage
		expectedErr string
	}{
		{
			name:  "created",
			state: created,
			msg: &peer.ChaincodeMessage{
				Type: peer.ChaincodeMessage_REGISTERED,
			},
		},
		{
			name:  "wrong message type in created state",
			state: created,
			msg: &peer.ChaincodeMessage{
				Type: peer.ChaincodeMessage_READY,
			},
			expectedErr: fmt.Sprintf("cannot handle message (%s)", peer.ChaincodeMessage_READY),
		},
		{
			name:  "established",
			state: established,
			msg: &peer.ChaincodeMessage{
				Type: peer.ChaincodeMessage_READY,
			},
		},
		{
			name:  "wrong message type in  established state",
			state: established,
			msg: &peer.ChaincodeMessage{
				Type: peer.ChaincodeMessage_REGISTERED,
			},
			expectedErr: fmt.Sprintf("cannot handle message (%s)", peer.ChaincodeMessage_REGISTERED),
		},
		{
			name:  "wrong message type in ready state",
			state: ready,
			msg: &peer.ChaincodeMessage{
				Type: peer.ChaincodeMessage_REGISTERED,
			},
			expectedErr: fmt.Sprintf("cannot handle message (%s)", peer.ChaincodeMessage_REGISTERED),
		},
		{
			name:  "keepalive",
			state: established,
			msg: &peer.ChaincodeMessage{
				Type: peer.ChaincodeMessage_KEEPALIVE,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			handler := &Handler{
				chatStream: &mock.PeerChaincodeStream{},
				cc:         &mockChaincode{},
				state:      test.state,
			}
			err := handler.handleMessage(test.msg, nil)
			if test.expectedErr != "" {
				assert.Contains(t, err.Error(), test.expectedErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandleMessage(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name         string
		msg          *peer.ChaincodeMessage
		msgType      peer.ChaincodeMessage_Type
		expectedErr  string
		invokeCalled bool
		initCalled   bool
	}{
		{
			name: "INIT",
			msg: &peer.ChaincodeMessage{
				Type: peer.ChaincodeMessage_INIT,
			},
			msgType:      peer.ChaincodeMessage_COMPLETED,
			initCalled:   true,
			invokeCalled: false,
		},
		{
			name: "INIT with bad payload",
			msg: &peer.ChaincodeMessage{
				Type:    peer.ChaincodeMessage_INIT,
				Payload: []byte{1},
			},
			msgType:      peer.ChaincodeMessage_ERROR,
			initCalled:   false,
			invokeCalled: false,
		},
		{
			name: "INVOKE",
			msg: &peer.ChaincodeMessage{
				Type: peer.ChaincodeMessage_TRANSACTION,
			},
			msgType:      peer.ChaincodeMessage_COMPLETED,
			initCalled:   false,
			invokeCalled: true,
		},
		{
			name: "INVOKE with bad payload",
			msg: &peer.ChaincodeMessage{
				Type:    peer.ChaincodeMessage_TRANSACTION,
				Payload: []byte{1},
			},
			msgType:      peer.ChaincodeMessage_ERROR,
			initCalled:   false,
			invokeCalled: false,
		},
		{
			name: "RESPONSE with no responseChannel",
			msg: &peer.ChaincodeMessage{
				Type: peer.ChaincodeMessage_RESPONSE,
			},
			expectedErr: "responseChannel does not exist",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			chatStream := &mock.PeerChaincodeStream{}
			cc := &mockChaincode{}

			msgChan := make(chan *peer.ChaincodeMessage)
			chatStream.SendStub = func(msg *peer.ChaincodeMessage) error {
				go func() {
					msgChan <- msg
				}()
				return nil
			}

			// create handler in ready state
			handler := &Handler{
				chatStream:       chatStream,
				cc:               cc,
				responseChannels: map[string]chan *peer.ChaincodeMessage{},
				state:            ready,
			}

			err := handler.handleMessage(test.msg, nil)
			if test.expectedErr != "" {
				assert.Contains(t, err.Error(), test.expectedErr)
			} else {
				if err != nil {
					t.Fatalf("Unexpected error for '%s': %s", test.name, err)
				}
				resp := <-msgChan
				assert.Equal(t, test.msgType, resp.GetType())
				assert.Equal(t, test.initCalled, cc.initCalled)
				assert.Equal(t, test.invokeCalled, cc.invokeCalled)
			}
		})
	}
}

func TestHandlePeerCalls(t *testing.T) {
	payload := []byte("error")
	h := &Handler{
		cc:               &mockChaincode{},
		responseChannels: map[string]chan *peer.ChaincodeMessage{},
		state:            ready,
	}
	chatStream := &mock.PeerChaincodeStream{}
	chatStream.SendStub = func(msg *peer.ChaincodeMessage) error {
		go func() {
			err := h.handleResponse(
				&peer.ChaincodeMessage{
					Type:      peer.ChaincodeMessage_ERROR,
					ChannelId: msg.GetChannelId(),
					Txid:      msg.GetTxid(),
					Payload:   payload,
				},
			)
			assert.NoError(t, err, "handleResponse")
		}()
		return nil
	}
	h.chatStream = chatStream

	_, err := h.handleQueryStateNext("id", "channel", "txid")
	assert.EqualError(t, err, string(payload))

	_, err = h.handleQueryStateClose("id", "channel", "txid")
	assert.EqualError(t, err, string(payload))

	// force error by removing responseChannels
	h.responseChannels = nil
	_, err = h.handleGetState("col", "key", "channel", "txid")
	assert.Contains(t, err.Error(), "[txid] error sending GET_STATE")

	_, err = h.handleGetPrivateDataHash("col", "key", "channel", "txid")
	assert.Contains(t, err.Error(), "[txid] error sending GET_PRIVATE_DATA_HASH")

	_, err = h.handleGetStateMetadata("col", "key", "channel", "txid")
	assert.Contains(t, err.Error(), "[txid] error sending GET_STATE_METADATA")

	err = h.handlePutState("col", "key", []byte{}, "channel", "txid")
	assert.Contains(t, err.Error(), "[txid] error sending PUT_STATE")

	err = h.handlePutStateMetadataEntry("col", "key", "mkey", []byte{}, "channel", "txid")
	assert.Contains(t, err.Error(), "[txid] error sending PUT_STATE_METADATA")

	err = h.handleDelState("col", "key", "channel", "txid")
	assert.Contains(t, err.Error(), "[txid] error sending DEL_STATE")

	_, err = h.handleGetStateByRange("col", "start", "end", []byte{}, "channel", "txid")
	assert.Contains(t, err.Error(), "[txid] error sending GET_STATE_BY_RANGE")

	_, err = h.handleQueryStateNext("id", "channel", "txid")
	assert.Contains(t, err.Error(), "cannot create response channel")

	_, err = h.handleQueryStateClose("id", "channel", "txid")
	assert.Contains(t, err.Error(), "cannot create response channel")

	_, err = h.handleGetQueryResult("col", "query", []byte{}, "channel", "txid")
	assert.Contains(t, err.Error(), "[txid] error sending GET_QUERY_RESULT")

	_, err = h.handleGetHistoryForKey("key", "channel", "txid")
	assert.Contains(t, err.Error(), "cannot create response channel")

}
