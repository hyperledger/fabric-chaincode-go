// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim/internal/mock"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/queryresult"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"google.golang.org/protobuf/proto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toChaincodeArgs(args ...string) [][]byte {
	ccArgs := make([][]byte, len(args))
	for i, a := range args {
		ccArgs[i] = []byte(a)
	}
	return ccArgs
}

// requireProtoEqual ensures an expected protobuf message matches an actual message
func requireProtoEqual(t *testing.T, expected proto.Message, actual proto.Message) {
	require.True(t, proto.Equal(expected, actual), "Expected %v, got %v", expected, actual)
}

func TestNewChaincodeStub(t *testing.T) {
	expectedArgs := toChaincodeArgs("function", "arg1", "arg2")
	expectedDecorations := map[string][]byte{"decoration-key": []byte("decoration-value")}
	expectedCreator := []byte("signature-header-creator")
	expectedTransient := map[string][]byte{"key": []byte("value")}
	expectedEpoch := uint64(999)

	validSignedProposal := &peer.SignedProposal{
		ProposalBytes: marshalOrPanic(&peer.Proposal{
			Header: marshalOrPanic(&common.Header{
				ChannelHeader: marshalOrPanic(&common.ChannelHeader{
					Type:  int32(common.HeaderType_ENDORSER_TRANSACTION),
					Epoch: expectedEpoch,
				}),
				SignatureHeader: marshalOrPanic(&common.SignatureHeader{
					Creator: expectedCreator,
				}),
			}),
			Payload: marshalOrPanic(&peer.ChaincodeProposalPayload{
				Input:        []byte("chaincode-proposal-input"),
				TransientMap: expectedTransient,
			}),
		}),
	}

	tests := []struct {
		signedProposal *peer.SignedProposal
		expectedErr    string
	}{
		{signedProposal: nil},
		{signedProposal: proto.Clone(validSignedProposal).(*peer.SignedProposal)},
		{
			signedProposal: &peer.SignedProposal{ProposalBytes: []byte("garbage")},
			expectedErr:    "failed to extract Proposal from SignedProposal",
		},
		{
			signedProposal: &peer.SignedProposal{},
			expectedErr:    "failed to extract Proposal fields: proposal header is nil",
		},
		{
			signedProposal: &peer.SignedProposal{},
			expectedErr:    "failed to extract Proposal fields: proposal header is nil",
		},
		{
			signedProposal: &peer.SignedProposal{
				ProposalBytes: marshalOrPanic(&peer.Proposal{
					Header: marshalOrPanic(&common.Header{
						ChannelHeader: marshalOrPanic(&common.ChannelHeader{
							Type:  int32(common.HeaderType_CONFIG_UPDATE),
							Epoch: expectedEpoch,
						}),
					}),
				}),
			},
			expectedErr: "invalid channel header type. Expected ENDORSER_TRANSACTION or CONFIG, received CONFIG_UPDATE",
		},
	}

	for _, tt := range tests {
		stub, err := newChaincodeStub(
			&Handler{},
			"channel-id",
			"transaction-id",
			&peer.ChaincodeInput{Args: expectedArgs[:], Decorations: expectedDecorations},
			tt.signedProposal,
		)
		if tt.expectedErr != "" {
			assert.Error(t, err)
			assert.ErrorContains(t, err, tt.expectedErr)
			continue
		}
		assert.NoError(t, err)
		assert.NotNil(t, stub)

		assert.Equal(t, &Handler{}, stub.handler, "expected empty handler")
		assert.Equal(t, "channel-id", stub.ChannelID)
		assert.Equal(t, "transaction-id", stub.TxID)
		assert.Equal(t, expectedArgs, stub.args)
		assert.Equal(t, expectedDecorations, stub.decorations)
		assert.Equal(t, "VALIDATION_PARAMETER", stub.validationParameterMetakey)
		if tt.signedProposal == nil {
			assert.Nil(t, stub.proposal, "expected nil proposal")
			assert.Nil(t, stub.creator, "expected nil creator")
			assert.Nil(t, stub.transient, "expected nil transient")
			assert.Nil(t, stub.binding, "expected nil binding")
			continue
		}

		prop := &peer.Proposal{}
		err = proto.Unmarshal(tt.signedProposal.ProposalBytes, prop)
		assert.NoError(t, err)
		assert.Equal(t, prop, stub.proposal)

		assert.Equal(t, expectedCreator, stub.creator)
		assert.Equal(t, expectedTransient, stub.transient)

		epoch := make([]byte, 8)
		binary.LittleEndian.PutUint64(epoch, expectedEpoch)
		shdr := &common.SignatureHeader{}
		digest := sha256.Sum256(append(append(shdr.GetNonce(), expectedCreator...), epoch...))
		assert.Equal(t, digest[:], stub.binding)
	}
}

func TestChaincodeStubSetEvent(t *testing.T) {
	stub := &ChaincodeStub{}
	err := stub.SetEvent("", []byte("event payload"))
	assert.EqualError(t, err, "event name can not be empty string")
	assert.Nil(t, stub.chaincodeEvent)

	stub = &ChaincodeStub{}
	err = stub.SetEvent("name", []byte("payload"))
	assert.NoError(t, err)
	assert.Equal(t, &peer.ChaincodeEvent{EventName: "name", Payload: []byte("payload")}, stub.chaincodeEvent)
}

func TestChaincodeStubAccessors(t *testing.T) {
	stub := &ChaincodeStub{TxID: "transaction-id"}
	assert.Equal(t, "transaction-id", stub.GetTxID())

	stub = &ChaincodeStub{ChannelID: "channel-id"}
	assert.Equal(t, "channel-id", stub.GetChannelID())

	stub = &ChaincodeStub{decorations: map[string][]byte{"key": []byte("value")}}
	assert.Equal(t, map[string][]byte{"key": []byte("value")}, stub.GetDecorations())

	stub = &ChaincodeStub{args: [][]byte{[]byte("function"), []byte("arg1"), []byte("arg2")}}
	assert.Equal(t, [][]byte{[]byte("function"), []byte("arg1"), []byte("arg2")}, stub.GetArgs())
	assert.Equal(t, []string{"function", "arg1", "arg2"}, stub.GetStringArgs())

	f, a := stub.GetFunctionAndParameters()
	assert.Equal(t, "function", f)
	assert.Equal(t, []string{"arg1", "arg2"}, a)

	as, err := stub.GetArgsSlice()
	assert.NoError(t, err)
	assert.Equal(t, []byte("functionarg1arg2"), as)

	stub = &ChaincodeStub{}
	f, a = stub.GetFunctionAndParameters()
	assert.Equal(t, "", f)
	assert.Empty(t, a)

	stub = &ChaincodeStub{creator: []byte("creator")}
	creator, err := stub.GetCreator()
	assert.NoError(t, err)
	assert.Equal(t, []byte("creator"), creator)

	stub = &ChaincodeStub{transient: map[string][]byte{"key": []byte("value")}}
	transient, err := stub.GetTransient()
	assert.NoError(t, err)
	assert.Equal(t, map[string][]byte{"key": []byte("value")}, transient)

	stub = &ChaincodeStub{binding: []byte("binding")}
	binding, err := stub.GetBinding()
	assert.NoError(t, err)
	assert.Equal(t, []byte("binding"), binding)

	stub = &ChaincodeStub{signedProposal: &peer.SignedProposal{ProposalBytes: []byte("proposal-bytes")}}
	sp, err := stub.GetSignedProposal()
	assert.NoError(t, err)
	assert.Equal(t, &peer.SignedProposal{ProposalBytes: []byte("proposal-bytes")}, sp)
}

func TestChaincodeStubGetTxTimestamp(t *testing.T) {
	now := timestamppb.Now()
	tests := []struct {
		proposal    *peer.Proposal
		ts          *timestamppb.Timestamp
		expectedErr string
	}{
		{
			ts: now,
			proposal: &peer.Proposal{
				Header: marshalOrPanic(&common.Header{
					ChannelHeader: marshalOrPanic(&common.ChannelHeader{
						Timestamp: now,
					}),
				}),
			},
		},
		{
			proposal: &peer.Proposal{
				Header: marshalOrPanic(&common.Header{
					ChannelHeader: []byte("garbage-channel-header"),
				}),
			},
			expectedErr: "error unmarshaling ChannelHeader",
		},
		{
			proposal:    &peer.Proposal{Header: []byte("garbage-header")},
			expectedErr: "error unmarshaling Header",
		},
	}

	for _, tt := range tests {
		stub := &ChaincodeStub{proposal: tt.proposal}
		ts, err := stub.GetTxTimestamp()
		if tt.expectedErr != "" {
			assert.ErrorContains(t, err, tt.expectedErr)
			continue
		}

		assert.NoError(t, err)
		assert.True(t, proto.Equal(ts, tt.ts))
	}
}

func TestGetMSPID(t *testing.T) {
	_, err := GetMSPID()
	assert.EqualError(t, err, "'CORE_PEER_LOCALMSPID' is not set")

	os.Setenv("CORE_PEER_LOCALMSPID", "mspid")

	mspid, err := GetMSPID()
	assert.NoError(t, err)
	assert.Equal(t, "mspid", mspid)

	os.Unsetenv("CORE_PEER_LOCALMSPID")
}

func TestChaincodeStubHandlers(t *testing.T) {
	var tests = []struct {
		name     string
		resType  peer.ChaincodeMessage_Type
		payload  []byte
		testFunc func(*ChaincodeStub, *Handler, *testing.T, []byte)
	}{
		{
			name:    "Simple Response",
			resType: peer.ChaincodeMessage_RESPONSE,
			payload: []byte("myvalue"),
			testFunc: func(s *ChaincodeStub, h *Handler, t *testing.T, payload []byte) {
				resp, err := s.GetState("key")
				if err != nil {
					t.Fatalf("Unexpected error for GetState: %s", err)
				}
				assert.Equal(t, payload, resp)

				resp, err = s.GetPrivateData("col", "key")
				if err != nil {
					t.Fatalf("Unexpected error for GetState: %s", err)
				}
				assert.Equal(t, payload, resp)
				_, err = s.GetPrivateData("", "key")
				assert.EqualError(t, err, "collection must not be an empty string")

				resp, err = s.GetPrivateDataHash("col", "key")
				if err != nil {
					t.Fatalf("Unexpected error for GetPrivateDataHash: %s", err)
				}
				assert.Equal(t, payload, resp)
				_, err = s.GetPrivateDataHash("", "key")
				assert.EqualError(t, err, "collection must not be an empty string")

				err = s.PutState("key", payload)
				assert.NoError(t, err)

				err = s.PutPrivateData("col", "key", payload)
				assert.NoError(t, err)
				err = s.PutPrivateData("", "key", payload)
				assert.EqualError(t, err, "collection must not be an empty string")
				err = s.PutPrivateData("col", "", payload)
				assert.EqualError(t, err, "key must not be an empty string")

				err = s.SetStateValidationParameter("key", payload)
				assert.NoError(t, err)

				err = s.SetPrivateDataValidationParameter("col", "key", payload)
				assert.NoError(t, err)

				err = s.DelState("key")
				assert.NoError(t, err)

				err = s.DelPrivateData("col", "key")
				assert.NoError(t, err)
				err = s.DelPrivateData("", "key")
				assert.EqualError(t, err, "collection must not be an empty string")

				err = s.PurgePrivateData("col", "key")
				assert.NoError(t, err)
				err = s.PurgePrivateData("", "key")
				assert.EqualError(t, err, "collection must not be an empty string")

			},
		},
		{
			name:    "ValidationParameter",
			resType: peer.ChaincodeMessage_RESPONSE,
			payload: marshalOrPanic(
				&peer.StateMetadataResult{
					Entries: []*peer.StateMetadata{
						{
							Metakey: "mkey",
							Value:   []byte("metavalue"),
						},
					},
				},
			),
			testFunc: func(s *ChaincodeStub, h *Handler, t *testing.T, payload []byte) {
				resp, err := s.GetStateValidationParameter("key")
				if err != nil {
					t.Fatalf("Unexpected error for GetStateValidationParameter: %s", err)
				}
				assert.Equal(t, []byte("metavalue"), resp)

				resp, err = s.GetPrivateDataValidationParameter("col", "key")
				if err != nil {
					t.Fatalf("Unexpected error for GetPrivateDataValidationParameter: %s", err)
				}
				assert.Equal(t, []byte("metavalue"), resp)
			},
		},
		{
			name:    "InvokeChaincode",
			resType: peer.ChaincodeMessage_RESPONSE,
			payload: marshalOrPanic(
				&peer.ChaincodeMessage{
					Type: peer.ChaincodeMessage_COMPLETED,
					Payload: marshalOrPanic(
						&peer.Response{
							Status:  OK,
							Payload: []byte("invokechaincode"),
						},
					),
				},
			),
			testFunc: func(s *ChaincodeStub, h *Handler, t *testing.T, payload []byte) {
				resp := s.InvokeChaincode("cc", [][]byte{}, "channel")
				assert.Equal(t, resp.Payload, []byte("invokechaincode"))
			},
		},
		{
			name:    "QueryResponse",
			resType: peer.ChaincodeMessage_RESPONSE,
			payload: marshalOrPanic(
				&peer.QueryResponse{
					Results: []*peer.QueryResultBytes{
						{
							ResultBytes: marshalOrPanic(
								&queryresult.KV{
									Key:   "querykey",
									Value: []byte("queryvalue"),
								},
							),
						},
					},
					Metadata: marshalOrPanic(
						&peer.QueryResponseMetadata{
							Bookmark:            "book",
							FetchedRecordsCount: 1,
						},
					),
					HasMore: true,
				},
			),
			testFunc: func(s *ChaincodeStub, h *Handler, t *testing.T, payload []byte) {
				expectedResult := &queryresult.KV{
					Key:   "querykey",
					Value: []byte("queryvalue"),
				}

				// stub stuff
				sqi, err := s.GetQueryResult("query")
				if err != nil {
					t.Fatalf("Unexpected error for GetQueryResult: %s", err)
				}
				kv, err := sqi.Next()
				if err != nil {
					t.Fatalf("Unexpected error for GetQueryResult: %s", err)
				}
				requireProtoEqual(t, expectedResult, kv)

				sqi, err = s.GetPrivateDataQueryResult("col", "query")
				if err != nil {
					t.Fatalf("Unexpected error for GetPrivateDataQueryResult: %s", err)
				}
				kv, err = sqi.Next()
				if err != nil {
					t.Fatalf("Unexpected error for GetPrivateDataQueryResult: %s", err)
				}
				requireProtoEqual(t, expectedResult, kv)

				_, err = s.GetPrivateDataQueryResult("", "query")
				assert.EqualError(t, err, "collection must not be an empty string")

				sqi, err = s.GetStateByRange("", "end")
				if err != nil {
					t.Fatalf("Unexpected error for GetStateByRange: %s", err)
				}
				// first result
				kv, err = sqi.Next()
				if err != nil {
					t.Fatalf("Unexpected error for GetStateByRange: %s", err)
				}
				requireProtoEqual(t, expectedResult, kv)
				// second result
				assert.True(t, sqi.HasNext())
				kv, err = sqi.Next()
				if err != nil {
					t.Fatalf("Unexpected error for GetStateByRange: %s", err)
				}
				requireProtoEqual(t, expectedResult, kv)
				err = sqi.Close()
				assert.NoError(t, err)

				sqi, qrm, err := s.GetStateByRangeWithPagination("", "end", 1, "book")
				assert.NoError(t, err)
				kv, err = sqi.Next()
				if err != nil {
					t.Fatalf("Unexpected error for GetStateByRangeWithPagination: %s", err)
				}
				requireProtoEqual(t, expectedResult, kv)
				assert.Equal(t, "book", qrm.GetBookmark())
				assert.Equal(t, int32(1), qrm.GetFetchedRecordsCount())

				sqi, err = s.GetPrivateDataByRange("col", "", "end")
				if err != nil {
					t.Fatalf("Unexpected error for GetPrivateDataByRange: %s", err)
				}
				kv, err = sqi.Next()
				if err != nil {
					t.Fatalf("Unexpected error for GetPrivateDataByRange: %s", err)
				}
				requireProtoEqual(t, expectedResult, kv)

				_, err = s.GetPrivateDataByRange("", "", "end")
				assert.EqualError(t, err, "collection must not be an empty string")

				sqi, err = s.GetStateByPartialCompositeKey("object", []string{"attr1", "attr2"})
				assert.NoError(t, err)
				kv, err = sqi.Next()
				if err != nil {
					t.Fatalf("Unexpected error for GetStateByPartialCompositeKey: %s", err)
				}
				requireProtoEqual(t, expectedResult, kv)

				sqi, err = s.GetPrivateDataByPartialCompositeKey("col", "object", []string{"attr1", "attr2"})
				assert.NoError(t, err)
				kv, err = sqi.Next()
				if err != nil {
					t.Fatalf("Unexpected error for GetPrivateDataByPartialCompositeKey: %s", err)
				}
				requireProtoEqual(t, expectedResult, kv)

				_, err = s.GetPrivateDataByPartialCompositeKey("", "object", []string{"attr1", "attr2"})
				assert.EqualError(t, err, "collection must not be an empty string")

				sqi, qrm, err = s.GetStateByPartialCompositeKeyWithPagination(
					"object",
					[]string{"key1", "key2"},
					1,
					"book",
				)
				assert.NoError(t, err)
				kv, err = sqi.Next()
				if err != nil {
					t.Fatalf("Unexpected error for GetStateByPartialCompositeKeyWithPagination: %s", err)
				}
				requireProtoEqual(t, expectedResult, kv)
				assert.Equal(t, "book", qrm.GetBookmark())
				assert.Equal(t, int32(1), qrm.GetFetchedRecordsCount())

				sqi, qrm, err = s.GetQueryResultWithPagination("query", 1, "book")
				assert.NoError(t, err)
				kv, err = sqi.Next()
				if err != nil {
					t.Fatalf("Unexpected error forGetQueryResultWithPagination: %s", err)
				}
				requireProtoEqual(t, expectedResult, kv)
				assert.Equal(t, "book", qrm.GetBookmark())
				assert.Equal(t, int32(1), qrm.GetFetchedRecordsCount())
			},
		},
		{
			name:    "GetHistoryForKey",
			resType: peer.ChaincodeMessage_RESPONSE,
			payload: marshalOrPanic(
				&peer.QueryResponse{
					Results: []*peer.QueryResultBytes{
						{
							ResultBytes: marshalOrPanic(
								&queryresult.KeyModification{
									TxId:  "txid",
									Value: []byte("historyforkey"),
								},
							),
						},
					},
					HasMore: false,
				},
			),
			testFunc: func(s *ChaincodeStub, h *Handler, t *testing.T, payload []byte) {
				expectedResult := &queryresult.KeyModification{
					TxId:  "txid",
					Value: []byte("historyforkey"),
				}
				hqi, err := s.GetHistoryForKey("key")
				if err != nil {
					t.Fatalf("Unexpected error for GetHistoryForKey: %s", err)
				}
				km, err := hqi.Next()
				if err != nil {
					t.Fatalf("Unexpected error for GetPrivateDataByRangee: %s", err)
				}
				requireProtoEqual(t, expectedResult, km)
				assert.False(t, hqi.HasNext())
			},
		},
		{
			name:    "Error Conditions",
			resType: peer.ChaincodeMessage_ERROR,
			payload: []byte("error"),
			testFunc: func(s *ChaincodeStub, h *Handler, t *testing.T, payload []byte) {
				_, err := s.GetState("key")
				assert.EqualError(t, err, string(payload))

				_, err = s.GetPrivateDataHash("col", "key")
				assert.EqualError(t, err, string(payload))

				_, err = s.GetStateValidationParameter("key")
				assert.EqualError(t, err, string(payload))

				err = s.PutState("key", payload)
				assert.EqualError(t, err, string(payload))

				err = s.SetPrivateDataValidationParameter("col", "key", payload)
				assert.EqualError(t, err, string(payload))

				err = s.DelState("key")
				assert.EqualError(t, err, string(payload))

				_, err = s.GetStateByRange("start", "end")
				assert.EqualError(t, err, string(payload))

				_, err = s.GetQueryResult("query")
				assert.EqualError(t, err, string(payload))

				_, err = s.GetHistoryForKey("key")
				assert.EqualError(t, err, string(payload))

				resp := s.InvokeChaincode("cc", [][]byte{}, "channel")
				assert.Equal(t, payload, resp.GetPayload())

			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			handler := &Handler{
				cc:               &mockChaincode{},
				responseChannels: map[string]chan *peer.ChaincodeMessage{},
				state:            ready,
			}
			stub := &ChaincodeStub{
				ChannelID:                  "channel",
				TxID:                       "txid",
				handler:                    handler,
				validationParameterMetakey: "mkey",
			}
			chatStream := &mock.PeerChaincodeStream{}
			chatStream.SendStub = func(msg *peer.ChaincodeMessage) error {
				go func() {
					err := handler.handleResponse(
						&peer.ChaincodeMessage{
							Type:      test.resType,
							ChannelId: msg.GetChannelId(),
							Txid:      msg.GetTxid(),
							Payload:   test.payload,
						},
					)
					assert.NoError(t, err, "handleResponse")
				}()
				return nil
			}
			handler.chatStream = chatStream
			test.testFunc(stub, handler, t, test.payload)
		})
	}
}
