// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package shim

import (
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
)

type writeBatch struct {
	writes map[string]*peer.WriteRecord
}

func newWriteBatch() *writeBatch {
	return &writeBatch{
		writes: make(map[string]*peer.WriteRecord),
	}
}

func (b *writeBatch) Writes() []*peer.WriteRecord {
	if b == nil {
		return nil
	}

	var results []*peer.WriteRecord
	for _, value := range b.writes {
		results = append(results, value)
	}

	return results
}

func (b *writeBatch) PutState(collection string, key string, value []byte) {
	b.writes[batchLedgerKey(collection, key)] = &peer.WriteRecord{
		Key:        key,
		Value:      value,
		Collection: collection,
		Type:       peer.WriteRecord_PUT_STATE,
	}
}

func (b *writeBatch) PutStateMetadataEntry(collection string, key string, metakey string, metadata []byte) {
	b.writes[batchLedgerKey(collection, key)] = &peer.WriteRecord{
		Key:        key,
		Collection: collection,
		Metadata:   &peer.StateMetadata{Metakey: metakey, Value: metadata},
		Type:       peer.WriteRecord_PUT_STATE_METADATA,
	}
}

func (b *writeBatch) DelState(collection string, key string) {
	b.writes[batchLedgerKey(collection, key)] = &peer.WriteRecord{
		Key:        key,
		Collection: collection,
		Type:       peer.WriteRecord_DEL_STATE,
	}
}

func (b *writeBatch) PurgeState(collection string, key string) {
	b.writes[batchLedgerKey(collection, key)] = &peer.WriteRecord{
		Key:        key,
		Collection: collection,
		Type:       peer.WriteRecord_PURGE_PRIVATE_DATA,
	}
}

func batchLedgerKey(collection string, key string) string {
	return prefixStateDataWriteBatch + collection + key
}
