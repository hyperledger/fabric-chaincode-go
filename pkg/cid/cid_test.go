// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package cid_test

import (
	"encoding/base64"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/v2/pkg/cid"
	"github.com/hyperledger/fabric-protos-go-apiv2/msp"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

const certWithOutAttrs = `-----BEGIN CERTIFICATE-----
MIICXTCCAgSgAwIBAgIUeLy6uQnq8wwyElU/jCKRYz3tJiQwCgYIKoZIzj0EAwIw
eTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcTDVNh
biBGcmFuY2lzY28xGTAXBgNVBAoTEEludGVybmV0IFdpZGdldHMxDDAKBgNVBAsT
A1dXVzEUMBIGA1UEAxMLZXhhbXBsZS5jb20wHhcNMTcwOTA4MDAxNTAwWhcNMTgw
OTA4MDAxNTAwWjBdMQswCQYDVQQGEwJVUzEXMBUGA1UECBMOTm9ydGggQ2Fyb2xp
bmExFDASBgNVBAoTC0h5cGVybGVkZ2VyMQ8wDQYDVQQLEwZGYWJyaWMxDjAMBgNV
BAMTBWFkbWluMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEFq/90YMuH4tWugHa
oyZtt4Mbwgv6CkBSDfYulVO1CVInw1i/k16DocQ/KSDTeTfgJxrX1Ree1tjpaodG
1wWyM6OBhTCBgjAOBgNVHQ8BAf8EBAMCB4AwDAYDVR0TAQH/BAIwADAdBgNVHQ4E
FgQUhKs/VJ9IWJd+wer6sgsgtZmxZNwwHwYDVR0jBBgwFoAUIUd4i/sLTwYWvpVr
TApzcT8zv/kwIgYDVR0RBBswGYIXQW5pbHMtTWFjQm9vay1Qcm8ubG9jYWwwCgYI
KoZIzj0EAwIDRwAwRAIgCoXaCdU8ZiRKkai0QiXJM/GL5fysLnmG2oZ6XOIdwtsC
IEmCsI8Mhrvx1doTbEOm7kmIrhQwUVDBNXCWX1t3kJVN
-----END CERTIFICATE-----
`
const certWithAttrs = `-----BEGIN CERTIFICATE-----
MIIB6TCCAY+gAwIBAgIUHkmY6fRP0ANTvzaBwKCkMZZPUnUwCgYIKoZIzj0EAwIw
GzEZMBcGA1UEAxMQZmFicmljLWNhLXNlcnZlcjAeFw0xNzA5MDgwMzQyMDBaFw0x
ODA5MDgwMzQyMDBaMB4xHDAaBgNVBAMTE015VGVzdFVzZXJXaXRoQXR0cnMwWTAT
BgcqhkjOPQIBBggqhkjOPQMBBwNCAATmB1r3CdWvOOP3opB3DjJnW3CnN8q1ydiR
dzmuA6A2rXKzPIltHvYbbSqISZJubsy8gVL6GYgYXNdu69RzzFF5o4GtMIGqMA4G
A1UdDwEB/wQEAwICBDAMBgNVHRMBAf8EAjAAMB0GA1UdDgQWBBTYKLTAvJJK08OM
VGwIhjMQpo2DrjAfBgNVHSMEGDAWgBTEs/52DeLePPx1+65VhgTwu3/2ATAiBgNV
HREEGzAZghdBbmlscy1NYWNCb29rLVByby5sb2NhbDAmBggqAwQFBgcIAQQaeyJh
dHRycyI6eyJhdHRyMSI6InZhbDEifX0wCgYIKoZIzj0EAwIDSAAwRQIhAPuEqWUp
svTTvBqLR5JeQSctJuz3zaqGRqSs2iW+QB3FAiAIP0mGWKcgSGRMMBvaqaLytBYo
9v3hRt1r8j8vN0pMcg==
-----END CERTIFICATE-----
`

// #nosec G101
const idemixCred = `CiAGTPr9iYBj7gqHeU90RgXXklBhgIKbtVtUzMDnhJ9bCRIggb1mWLzNEO1ac+PfDNSpbCC02eY5OqRooSH1mnhlftEaQgoMaWRlbWl4TVNQSUQyEhBvcmcxLmRlcGFydG1lbnQxGiCEOl1qWZ+TRhDD3X6Cl59utnDnW8oio0y2vuJ1xHNMMiIOCgxpZGVtaXhNU1BJRDIq5gYKRAog1MgGWhcd/jgQpbpdO17LktSoelSsQJKfAmFhspaM6+QSICMcxQLs4JRPeSbyWG81KNepmLIi8C1AOyrgytJYMmKIEkQKICCwfD7vfGKqzGFEa7H7cBbR81kImbXcECJnDbj4QQNUEiB64Bi0jRahh18QuZzqnw6sksn8GBCi2sVsrjtLTKsvMRpECiBD9miof2CyCjHOr6s/JAiALRzjdogv0xQHEyqNAfIDABIgx56y7lUllGc0XtYsFIdq7CulDE55Re5xT1wvzRNhITUiIPSaozvr294lNGF3Wy5Yd7wlNW/IZBpBcXda/dQfGci9KiBbO2o2bWD1P4HOMfI//ebo8WrwTNgmPfmlqNBxzKuQMjIg5AwmQGEYnKN/pOVDFMjm/3a9hJDv9R2svI42aVBms0M6IHMSFIZ8j/yZH5nHtCkwpQMCuBFmI6krD2CfTjCiOUfoQiDO7cyRnCt9uEGIhQsBiwnSEXH+G9Il9qvfkUrAiZlbrkogv6dmb1xijfB3gsyVWxgfKlRNRtf78dMwjSf76jEnSrBSIJTkD7lSBwBepMFROxYneTHuG6JcSZpdoeOGqFl0drJWUiC8ndC2y9LsFJLKs2ddFqsFW7kNg+vROXuSLQdglSBffVog/eDzc90wTBEZu2T6LhWEbcP5oZ5TYdE/o+cOUfPgV4RiRAogBkz6/YmAY+4Kh3lPdEYF15JQYYCCm7VbVMzA54SfWwkSIIG9Zli8zRDtWnPj3wzUqWwgtNnmOTqkaKEh9Zp4ZX7RaiCXNeWrQz2UPkuAEZrt++TP/DbmAFF7cBQlYkb81jrn/nKIAQog/gwzULTJbCAoVg9XfCiROs4cU5oSv4Q80iYWtonAnvsSIE6mYFdzisBU21rhxjfYE7kk3Xjih9A1idJp7TSjfmorGiBwIEbnxUKjs3Z3DXUSTj5R78skdY1hWEjpCbSBvtwn/yIgBVTjvNOIwpBC7qZJKX6yn4tMvoCCGpiz4BKBEUqtBJt6ZzBlAjEAoBaHzX1HjvrnPMDXajqcLeHR5//AIIGDDcGQ+4GNqJu9Wawlw6Zs58Nnkpmh29ivAjBJNHeGNvX9sQb9lyzLAtCa5Il4xKNGGpGZ+uhQAjtNpRAZLtv2hgSqJAy0X6HwNXeAAQGKAQA=`

func TestClient(t *testing.T) {
	stub, err := getMockStub()
	assert.NoError(t, err, "Failed to get mock submitter")
	sinfo, err := cid.New(stub)
	assert.NoError(t, err, "Error getting submitter of the transaction")
	id, err := cid.GetID(stub)
	assert.NoError(t, err, "Error getting ID of the submitter of the transaction")
	assert.NotEmpty(t, id, "Transaction submitter ID should not be empty")
	t.Logf("The client's ID is: %s", id)
	cert, err := cid.GetX509Certificate(stub)
	assert.NoError(t, err, "Error getting X509 certificate of the submitter of the transaction")
	assert.NotNil(t, cert, "Transaction submitter certificate should not be nil")
	mspid, err := cid.GetMSPID(stub)
	assert.NoError(t, err, "Error getting MSP ID of the submitter of the transaction")
	assert.NotEmpty(t, mspid, "Transaction submitter MSP ID should not be empty")
	_, found, err := sinfo.GetAttributeValue("foo")
	assert.NoError(t, err, "Error getting Unique ID of the submitter of the transaction")
	assert.False(t, found, "Attribute 'foo' should not be found in the submitter cert")
	err = cid.AssertAttributeValue(stub, "foo", "")
	assert.Error(t, err, "AssertAttributeValue should have returned an error with no attribute")
	found, err = cid.HasOUValue(stub, "Fabric")
	assert.NoError(t, err, "Error getting X509 cert of the submitter of the transaction")
	assert.True(t, found)
	found, err = cid.HasOUValue(stub, "foo")
	assert.NoError(t, err, "HasOUValue")
	assert.False(t, found, "OU 'foo' should not be found in the submitter cert")

	stub, err = getMockStubWithAttrs()
	assert.NoError(t, err, "Failed to get mock submitter")
	sinfo, err = cid.New(stub)
	assert.NoError(t, err, "Failed to new client")
	attrVal, found, err := sinfo.GetAttributeValue("attr1")
	assert.NoError(t, err, "Error getting Unique ID of the submitter of the transaction")
	assert.True(t, found, "Attribute 'attr1' should be found in the submitter cert")
	assert.Equal(t, attrVal, "val1", "Value of attribute 'attr1' should be 'val1'")
	attrVal, found, err = cid.GetAttributeValue(stub, "attr1")
	assert.NoError(t, err, "Error getting Unique ID of the submitter of the transaction")
	assert.True(t, found, "Attribute 'attr1' should be found in the submitter cert")
	assert.Equal(t, attrVal, "val1", "Value of attribute 'attr1' should be 'val1'")
	err = cid.AssertAttributeValue(stub, "attr1", "val1")
	assert.NoError(t, err, "Error in AssertAttributeValue")
	err = cid.AssertAttributeValue(stub, "attr1", "val2")
	assert.Error(t, err, "Assert should have failed; value was val1, not val2")
	found, err = cid.HasOUValue(stub, "foo")
	assert.NoError(t, err, "Error getting X509 cert of the submitter of the transaction")
	assert.False(t, found, "HasOUValue")

	// Error case1
	stub, err = getMockStubWithNilCreator()
	assert.NoError(t, err, "Failed to get mock submitter")
	_, err = cid.New(stub)
	assert.Error(t, err, "NewSubmitterInfo should have returned an error when submitter with nil creator is passed")

	// Error case2
	stub, err = getMockStubWithFakeCreator()
	assert.NoError(t, err, "Failed to get mock submitter")
	_, err = cid.New(stub)
	assert.Error(t, err, "NewSubmitterInfo should have returned an error when submitter with fake creator is passed")
}

func TestIdemix(t *testing.T) {
	stub, err := getIdemixMockStubWithAttrs()
	assert.NoError(t, err, "Failed to get mock idemix stub")
	sinfo, err := cid.New(stub)
	assert.NoError(t, err, "Failed to new client")
	cert, err := sinfo.GetX509Certificate()
	assert.Nil(t, cert, "Idemix can't get x509 type of cert")
	assert.NoError(t, err, "Err for this func is nil")
	id, err := cid.GetID(stub)
	assert.Error(t, err, "Cannot determine identity")
	assert.Equal(t, id, "", "Id should be empty when Idemix")
	attrVal, found, err := sinfo.GetAttributeValue("ou")
	assert.NoError(t, err, "Error getting 'ou' of the submitter of the transaction")
	assert.True(t, found, "Attribute 'ou' should be found in the submitter cert")
	assert.Equal(t, attrVal, "org1.department1", "Value of attribute 'attr1' should be 'val1'")
	attrVal, found, err = sinfo.GetAttributeValue("role")
	assert.NoError(t, err, "Error getting 'role' of the submitter of the transaction")
	assert.True(t, found, "Attribute 'role' should be found in the submitter cert")
	assert.Equal(t, attrVal, "member", "Value of attribute 'attr1' should be 'val1'")
	_, found, err = sinfo.GetAttributeValue("id")
	assert.NoError(t, err, "GetAttributeValue")
	assert.False(t, found, "Attribute 'id' should not be found in the submitter cert")
}

func getMockStub() (cid.ChaincodeStubInterface, error) {
	stub := &mockStub{}
	sid := &msp.SerializedIdentity{Mspid: "SampleOrg",
		IdBytes: []byte(certWithOutAttrs)}
	b, err := proto.Marshal(sid)
	if err != nil {
		return nil, err
	}
	stub.creator = b
	return stub, nil
}

func getMockStubWithAttrs() (cid.ChaincodeStubInterface, error) {
	stub := &mockStub{}
	sid := &msp.SerializedIdentity{Mspid: "SampleOrg",
		IdBytes: []byte(certWithAttrs)}
	b, err := proto.Marshal(sid)
	if err != nil {
		return nil, err
	}
	stub.creator = b
	return stub, nil
}

func getIdemixMockStubWithAttrs() (cid.ChaincodeStubInterface, error) {
	stub := &mockStub{}
	idBytes, err := base64.StdEncoding.DecodeString(idemixCred)
	if err != nil {
		return nil, err
	}
	sid := &msp.SerializedIdentity{Mspid: "idemixOrg",
		IdBytes: idBytes,
	}
	b, err := proto.Marshal(sid)
	if err != nil {
		return nil, err
	}
	stub.creator = b
	return stub, nil
}

func getMockStubWithNilCreator() (cid.ChaincodeStubInterface, error) {
	c := &mockStub{}
	c.creator = nil
	return c, nil
}

func getMockStubWithFakeCreator() (cid.ChaincodeStubInterface, error) {
	c := &mockStub{}
	c.creator = []byte("foo")
	return c, nil
}

type mockStub struct {
	creator []byte
}

func (s *mockStub) GetCreator() ([]byte, error) {
	return s.creator, nil
}
