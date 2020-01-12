// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"io/ioutil"
	"os"
	"testing"
	"time"

	. "github.com/hyperledger/fabric-chaincode-go/shim/internal"
	peerpb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// TLS <key, cert, cacert> tuples for client and server were created
// using cryptogen tool. Of course, any standard tool such as openssl
// could have been used as well
var keyPEM = `-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgKg8jpiNIB5LXLull
IRoYMsQximSiU7XvGCYLslx4GauhRANCAARBGdslxalpg0dxk9GwVhi+Qw9oKZPE
n1hWPFmusDKtNbDLsHd9k1lU+SWnJKYlg7hmaUvxC1lR2M6KmvAwSUfN
-----END PRIVATE KEY-----
`
var certPEM = `-----BEGIN CERTIFICATE-----
MIICaTCCAhCgAwIBAgIQS46wcUDY2nJ2gQ/7fp/ptzAKBggqhkjOPQQDAjB2MQsw
CQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZy
YW5jaXNjbzEZMBcGA1UEChMQb3JnMS5leGFtcGxlLmNvbTEfMB0GA1UEAxMWdGxz
Y2Eub3JnMS5leGFtcGxlLmNvbTAeFw0xOTEyMTIwMTA1NTBaFw0yOTEyMDkwMTA1
NTBaMFoxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQH
Ew1TYW4gRnJhbmNpc2NvMR4wHAYDVQQDExVteWNjLm9yZzEuZXhhbXBsZS5jb20w
WTATBgcqhkjOPQIBBggqhkjOPQMBBwNCAARBGdslxalpg0dxk9GwVhi+Qw9oKZPE
n1hWPFmusDKtNbDLsHd9k1lU+SWnJKYlg7hmaUvxC1lR2M6KmvAwSUfNo4GbMIGY
MA4GA1UdDwEB/wQEAwIFoDAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIw
DAYDVR0TAQH/BAIwADArBgNVHSMEJDAigCBxQqUF6hEsSgXTc47WT4U58SOdgX8n
8RlMuxFg0wRtjjAsBgNVHREEJTAjghVteWNjLm9yZzEuZXhhbXBsZS5jb22CBG15
Y2OHBH8AAAEwCgYIKoZIzj0EAwIDRwAwRAIgWgxAuGibD+Da/qCLBryJMDGlyIrx
HV+tI33lEy1B9qoCIEJD4xipI2WYp1sHmK2nxYPcoTb9WLFdNZ6twKZyw9c8
-----END CERTIFICATE-----
`
var rootPEM = `-----BEGIN CERTIFICATE-----
MIICSTCCAe+gAwIBAgIQWpamEC5/D2N5JKS8FEpgTzAKBggqhkjOPQQDAjB2MQsw
CQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZy
YW5jaXNjbzEZMBcGA1UEChMQb3JnMS5leGFtcGxlLmNvbTEfMB0GA1UEAxMWdGxz
Y2Eub3JnMS5leGFtcGxlLmNvbTAeFw0xOTEyMTIwMTA1NTBaFw0yOTEyMDkwMTA1
NTBaMHYxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYwFAYDVQQH
Ew1TYW4gRnJhbmNpc2NvMRkwFwYDVQQKExBvcmcxLmV4YW1wbGUuY29tMR8wHQYD
VQQDExZ0bHNjYS5vcmcxLmV4YW1wbGUuY29tMFkwEwYHKoZIzj0CAQYIKoZIzj0D
AQcDQgAE2eFjoZkB/ozmheZZ9P05kUXAQAG+j0oTmRr9vX2qJa+tyrbS/i4UKrXo
82dqcDmmL16l2ukBXt7/aBre5WbVEaNfMF0wDgYDVR0PAQH/BAQDAgGmMA8GA1Ud
JQQIMAYGBFUdJQAwDwYDVR0TAQH/BAUwAwEB/zApBgNVHQ4EIgQgcUKlBeoRLEoF
03OO1k+FOfEjnYF/J/EZTLsRYNMEbY4wCgYIKoZIzj0EAwIDSAAwRQIhANmPRnJi
p7amrl9rF5xWtW0rR+y9uSCi6cy/T8bJl1JTAiATHlHcuNhHFeGb+Vl512FC3sGM
bHHlP/A/QkbGqJL4HQ==
-----END CERTIFICATE-----
`

var clientKeyPEM = `-----BEGIN EC PRIVATE KEY-----
MHcCAQEEINVHep4/z6iPa151Ipp4MmCb1l/VKkY3vuMfUQf3LhQboAoGCCqGSM49
AwEHoUQDQgAEcE6hZ7muszSi5wXIVKPdIuLYPTIxQxj+jekPRfFnJF/RJKM0Nj3T
Bk9spwCHwu1t3REyobjaZcFQk0y32Pje5A==
-----END EC PRIVATE KEY-----
`

var clientCertPEM = `-----BEGIN CERTIFICATE-----
MIICAzCCAaqgAwIBAgIQe/ZUgn+/dH6FGrx+dr/PfjAKBggqhkjOPQQDAjBYMQsw
CQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZy
YW5jaXNjbzENMAsGA1UEChMET3JnMTENMAsGA1UEAxMET3JnMTAeFw0xODA4MjEw
ODI1MzNaFw0yODA4MTgwODI1MzNaMGgxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpD
YWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMRUwEwYDVQQKEwxPcmcx
LWNsaWVudDExFTATBgNVBAMTDE9yZzEtY2xpZW50MTBZMBMGByqGSM49AgEGCCqG
SM49AwEHA0IABHBOoWe5rrM0oucFyFSj3SLi2D0yMUMY/o3pD0XxZyRf0SSjNDY9
0wZPbKcAh8Ltbd0RMqG42mXBUJNMt9j43uSjRjBEMA4GA1UdDwEB/wQEAwIFoDAT
BgNVHSUEDDAKBggrBgEFBQcDAjAMBgNVHRMBAf8EAjAAMA8GA1UdIwQIMAaABAEC
AwQwCgYIKoZIzj0EAwIDRwAwRAIgaK/prRkZS6zctxwBUl2QApUrH7pMmab30Nn9
ER8f3m0CICBZ9XoxKXEFFcSRpfiA2/vzoOPg76lRXcCklxzGSJYu
-----END CERTIFICATE-----
`

var clientRootPEM = `-----BEGIN CERTIFICATE-----
MIIB8TCCAZegAwIBAgIQUigdJy6IudO7sVOXsKVrtzAKBggqhkjOPQQDAjBYMQsw
CQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMNU2FuIEZy
YW5jaXNjbzENMAsGA1UEChMET3JnMTENMAsGA1UEAxMET3JnMTAeFw0xODA4MjEw
ODI1MzNaFw0yODA4MTgwODI1MzNaMFgxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpD
YWxpZm9ybmlhMRYwFAYDVQQHEw1TYW4gRnJhbmNpc2NvMQ0wCwYDVQQKEwRPcmcx
MQ0wCwYDVQQDEwRPcmcxMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEVOI+oAAB
Pl+iRsCcGq81WbXap2L1r432T5gbzUNKYRvVsyFFYmdO8ql8uDi4UxSY64eaeRFT
uxdcsTG7M5K2yaNDMEEwDgYDVR0PAQH/BAQDAgGmMA8GA1UdJQQIMAYGBFUdJQAw
DwYDVR0TAQH/BAUwAwEB/zANBgNVHQ4EBgQEAQIDBDAKBggqhkjOPQQDAgNIADBF
AiEA6U7IRGf+S7e9U2+jSI2eFiBsVEBIi35LgYoKqjELj5oCIAD7DfVMaMHzzjiQ
XIlJQdS/9afDi32qZWZfe3kAUAs0
-----END CERTIFICATE-----
`

func TestLoadBase64EncodedConfig(t *testing.T) {
	// setup key/cert files
	testDir, err := ioutil.TempDir("", "shiminternal")
	if err != nil {
		t.Fatalf("Failed to test directory: %s", err)
	}
	defer os.RemoveAll(testDir)

	keyFile, err := ioutil.TempFile(testDir, "testKey")
	if err != nil {
		t.Fatalf("Failed to create key file: %s", err)
	}
	b64Key := base64.StdEncoding.EncodeToString([]byte(keyPEM))
	if _, err := keyFile.WriteString(b64Key); err != nil {
		t.Fatalf("Failed to write to key file: %s", err)
	}

	certFile, err := ioutil.TempFile(testDir, "testCert")
	if err != nil {
		t.Fatalf("Failed to create cert file: %s", err)
	}
	b64Cert := base64.StdEncoding.EncodeToString([]byte(certPEM))
	if _, err := certFile.WriteString(b64Cert); err != nil {
		t.Fatalf("Failed to write to cert file: %s", err)
	}

	rootFile, err := ioutil.TempFile(testDir, "testRoot")
	if err != nil {
		t.Fatalf("Failed to create root file: %s", err)
	}
	if _, err := rootFile.WriteString(rootPEM); err != nil {
		t.Fatalf("Failed to write to root file: %s", err)
	}

	notb64File, err := ioutil.TempFile(testDir, "testNotb64")
	if err != nil {
		t.Fatalf("Failed to create notb64 file: %s", err)
	}
	if _, err := notb64File.WriteString("#####"); err != nil {
		t.Fatalf("Failed to write to notb64 file: %s", err)
	}

	notPEMFile, err := ioutil.TempFile(testDir, "testNotPEM")
	if err != nil {
		t.Fatalf("Failed to create notPEM file: %s", err)
	}
	b64 := base64.StdEncoding.EncodeToString([]byte("not pem"))
	if _, err := notPEMFile.WriteString(b64); err != nil {
		t.Fatalf("Failed to write to notPEM file: %s", err)
	}

	defer cleanupEnv()

	// expected TLS config
	rootPool := x509.NewCertPool()
	rootPool.AppendCertsFromPEM([]byte(rootPEM))
	clientCert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		t.Fatalf("Failed to load client cert pair: %s", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      rootPool,
	}

	kaOpts := keepalive.ClientParameters{
		Time:                1 * time.Minute,
		Timeout:             20 * time.Second,
		PermitWithoutStream: true,
	}

	var tests = []struct {
		name     string
		env      map[string]string
		expected Config
		errMsg   string
	}{
		{
			name: "TLS disabled",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME": "testCC",
				"CORE_PEER_TLS_ENABLED":  "false",
			},
			expected: Config{
				ChaincodeName: "testCC",
				KaOpts:        kaOpts,
			},
		},
		{
			name: "TLS Enabled",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":      "testCC",
				"CORE_PEER_TLS_ENABLED":       "true",
				"CORE_TLS_CLIENT_KEY_PATH":    keyFile.Name(),
				"CORE_TLS_CLIENT_CERT_PATH":   certFile.Name(),
				"CORE_PEER_TLS_ROOTCERT_FILE": rootFile.Name(),
			},
			expected: Config{
				ChaincodeName: "testCC",
				TLS:           tlsConfig,
				KaOpts:        kaOpts,
			},
		},
		{
			name: "Bad TLS_ENABLED",
			env: map[string]string{
				"CORE_PEER_TLS_ENABLED": "nottruthy",
			},
			errMsg: "'CORE_PEER_TLS_ENABLED' must be set to 'true' or 'false'",
		},
		{
			name: "Missing key file",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":   "testCC",
				"CORE_PEER_TLS_ENABLED":    "true",
				"CORE_TLS_CLIENT_KEY_PATH": "missingkey",
			},
			errMsg: "failed to read private key file",
		},
		{
			name: "Bad key file",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":   "testCC",
				"CORE_PEER_TLS_ENABLED":    "true",
				"CORE_TLS_CLIENT_KEY_PATH": notb64File.Name(),
			},
			errMsg: "failed to decode private key file",
		},
		{
			name: "Missing cert file",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":    "testCC",
				"CORE_PEER_TLS_ENABLED":     "true",
				"CORE_TLS_CLIENT_KEY_PATH":  keyFile.Name(),
				"CORE_TLS_CLIENT_CERT_PATH": "missingkey",
			},
			errMsg: "failed to read public key file",
		},
		{
			name: "Bad cert file",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":    "testCC",
				"CORE_PEER_TLS_ENABLED":     "true",
				"CORE_TLS_CLIENT_KEY_PATH":  keyFile.Name(),
				"CORE_TLS_CLIENT_CERT_PATH": notb64File.Name(),
			},
			errMsg: "failed to decode public key file",
		},
		{
			name: "Missing root file",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":      "testCC",
				"CORE_PEER_TLS_ENABLED":       "true",
				"CORE_TLS_CLIENT_KEY_PATH":    keyFile.Name(),
				"CORE_TLS_CLIENT_CERT_PATH":   certFile.Name(),
				"CORE_PEER_TLS_ROOTCERT_FILE": "missingkey",
			},
			errMsg: "failed to read root cert file",
		},
		{
			name: "Bad root file",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":      "testCC",
				"CORE_PEER_TLS_ENABLED":       "true",
				"CORE_TLS_CLIENT_KEY_PATH":    keyFile.Name(),
				"CORE_TLS_CLIENT_CERT_PATH":   certFile.Name(),
				"CORE_PEER_TLS_ROOTCERT_FILE": notb64File.Name(),
			},
			errMsg: "failed to load root cert file",
		},
		{
			name: "Key not PEM",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":      "testCC",
				"CORE_PEER_TLS_ENABLED":       "true",
				"CORE_TLS_CLIENT_KEY_PATH":    notPEMFile.Name(),
				"CORE_TLS_CLIENT_CERT_PATH":   certFile.Name(),
				"CORE_PEER_TLS_ROOTCERT_FILE": rootFile.Name(),
			},
			errMsg: "failed to parse client key pair",
		},
		{
			name: "Cert not PEM",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":      "testCC",
				"CORE_PEER_TLS_ENABLED":       "true",
				"CORE_TLS_CLIENT_KEY_PATH":    keyFile.Name(),
				"CORE_TLS_CLIENT_CERT_PATH":   notPEMFile.Name(),
				"CORE_PEER_TLS_ROOTCERT_FILE": rootFile.Name(),
			},
			errMsg: "failed to parse client key pair",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.env {
				os.Setenv(k, v)
			}
			conf, err := LoadConfig()
			if test.errMsg == "" {
				assert.Equal(t, test.expected, conf)
			} else {
				assert.Contains(t, err.Error(), test.errMsg)
			}
		})
	}

	tlsServerConfig := &tls.Config{
		MinVersion:             tls.VersionTLS12,
		Certificates:           []tls.Certificate{clientCert},
		ClientCAs:              rootPool,
		SessionTicketsDisabled: true,
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	tlsServerNonMutualConfig := &tls.Config{
		MinVersion:             tls.VersionTLS12,
		Certificates:           []tls.Certificate{clientCert},
		RootCAs:                nil,
		SessionTicketsDisabled: true,
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		},
		ClientAuth: tls.NoClientCert,
	}

	// additional tests to differentiate client vs server
	var tlsTests = []struct {
		name     string
		issrv    bool
		key      []byte
		cert     []byte
		rootCert []byte
		expected *tls.Config
		errMsg   string
	}{
		{
			name:     "Server TLS",
			issrv:    true,
			key:      []byte(keyPEM),
			cert:     []byte(certPEM),
			rootCert: []byte(rootPEM),
			expected: tlsServerConfig,
		},
		{
			name:     "Server non-mutual TLS",
			issrv:    true,
			key:      []byte(keyPEM),
			cert:     []byte(certPEM),
			rootCert: nil,
			expected: tlsServerNonMutualConfig,
		},
		{
			name:     "Server key unspecified",
			issrv:    true,
			key:      nil,
			cert:     []byte(certPEM),
			rootCert: []byte(rootPEM),
			errMsg:   "key not provided",
		},
		{
			name:     "Server cert unspecified",
			issrv:    true,
			key:      []byte(keyPEM),
			cert:     nil,
			rootCert: []byte(rootPEM),
			errMsg:   "cert not provided",
		},
		{
			name:     "Client TLS root CA unspecified",
			issrv:    false,
			key:      []byte(keyPEM),
			cert:     []byte(certPEM),
			rootCert: nil,
			errMsg:   "root cert not provided",
		},
	}

	for _, test := range tlsTests {
		t.Run(test.name, func(t *testing.T) {
			tlsCfg, err := LoadTLSConfig(test.issrv, test.key, test.cert, test.rootCert)
			if test.errMsg == "" {
				assert.Equal(t, test.expected, tlsCfg)
			} else {
				assert.Contains(t, err.Error(), test.errMsg)
			}
		})
	}
}

func TestLoadPEMEncodedConfig(t *testing.T) {
	// setup key/cert files
	testDir, err := ioutil.TempDir("", "shiminternal")
	if err != nil {
		t.Fatalf("Failed to test directory: %s", err)
	}
	defer os.RemoveAll(testDir)

	keyFile, err := ioutil.TempFile(testDir, "testKey")
	if err != nil {
		t.Fatalf("Failed to create key file: %s", err)
	}
	if _, err := keyFile.WriteString(keyPEM); err != nil {
		t.Fatalf("Failed to write to key file: %s", err)
	}

	certFile, err := ioutil.TempFile(testDir, "testCert")
	if err != nil {
		t.Fatalf("Failed to create cert file: %s", err)
	}
	if _, err := certFile.WriteString(certPEM); err != nil {
		t.Fatalf("Failed to write to cert file: %s", err)
	}

	rootFile, err := ioutil.TempFile(testDir, "testRoot")
	if err != nil {
		t.Fatalf("Failed to create root file: %s", err)
	}
	if _, err := rootFile.WriteString(rootPEM); err != nil {
		t.Fatalf("Failed to write to root file: %s", err)
	}

	keyFile64, err := ioutil.TempFile(testDir, "testKey64")
	if err != nil {
		t.Fatalf("Failed to create key file: %s", err)
	}
	b64Key := base64.StdEncoding.EncodeToString([]byte(keyPEM))
	if _, err := keyFile64.WriteString(b64Key); err != nil {
		t.Fatalf("Failed to write to key file: %s", err)
	}

	certFile64, err := ioutil.TempFile(testDir, "testCert64")
	if err != nil {
		t.Fatalf("Failed to create cert file: %s", err)
	}
	b64Cert := base64.StdEncoding.EncodeToString([]byte(certPEM))
	if _, err := certFile64.WriteString(b64Cert); err != nil {
		t.Fatalf("Failed to write to cert file: %s", err)
	}

	defer cleanupEnv()

	// expected TLS config
	rootPool := x509.NewCertPool()
	rootPool.AppendCertsFromPEM([]byte(rootPEM))
	clientCert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		t.Fatalf("Failed to load client cert pair: %s", err)
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      rootPool,
	}

	kaOpts := keepalive.ClientParameters{
		Time:                1 * time.Minute,
		Timeout:             20 * time.Second,
		PermitWithoutStream: true,
	}

	var tests = []struct {
		name     string
		env      map[string]string
		expected Config
		errMsg   string
	}{
		{
			name: "TLS Enabled with PEM-encoded variables",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":      "testCC",
				"CORE_PEER_TLS_ENABLED":       "true",
				"CORE_TLS_CLIENT_KEY_FILE":    keyFile.Name(),
				"CORE_TLS_CLIENT_CERT_FILE":   certFile.Name(),
				"CORE_PEER_TLS_ROOTCERT_FILE": rootFile.Name(),
			},
			expected: Config{
				ChaincodeName: "testCC",
				TLS:           tlsConfig,
				KaOpts:        kaOpts,
			},
		},
		{
			name: "Client cert uses base64 encoding",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":      "testCC",
				"CORE_PEER_TLS_ENABLED":       "true",
				"CORE_TLS_CLIENT_KEY_FILE":    keyFile.Name(),
				"CORE_TLS_CLIENT_CERT_PATH":   certFile64.Name(),
				"CORE_PEER_TLS_ROOTCERT_FILE": rootFile.Name(),
			},
			expected: Config{
				ChaincodeName: "testCC",
				TLS:           tlsConfig,
				KaOpts:        kaOpts,
			},
		},
		{
			name: "Client key uses base64 encoding",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":      "testCC",
				"CORE_PEER_TLS_ENABLED":       "true",
				"CORE_TLS_CLIENT_KEY_PATH":    keyFile64.Name(),
				"CORE_TLS_CLIENT_CERT_FILE":   certFile.Name(),
				"CORE_PEER_TLS_ROOTCERT_FILE": rootFile.Name(),
			},
			expected: Config{
				ChaincodeName: "testCC",
				TLS:           tlsConfig,
				KaOpts:        kaOpts,
			},
		},
		{
			name: "Client cert uses base64 encoding with PEM variable",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":      "testCC",
				"CORE_PEER_TLS_ENABLED":       "true",
				"CORE_TLS_CLIENT_KEY_FILE":    keyFile.Name(),
				"CORE_TLS_CLIENT_CERT_FILE":   certFile64.Name(),
				"CORE_PEER_TLS_ROOTCERT_FILE": rootFile.Name(),
			},
			errMsg: "failed to parse client key pair",
		},
		{
			name: "Client key uses base64 encoding with PEM variable",
			env: map[string]string{
				"CORE_CHAINCODE_ID_NAME":      "testCC",
				"CORE_PEER_TLS_ENABLED":       "true",
				"CORE_TLS_CLIENT_KEY_FILE":    keyFile64.Name(),
				"CORE_TLS_CLIENT_CERT_FILE":   certFile.Name(),
				"CORE_PEER_TLS_ROOTCERT_FILE": rootFile.Name(),
			},
			errMsg: "failed to parse client key pair",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.env {
				os.Setenv(k, v)
			}
			conf, err := LoadConfig()
			if test.errMsg == "" {
				assert.Equal(t, test.expected, conf)
			} else {
				assert.Contains(t, err.Error(), test.errMsg)
			}
		})
	}
}

func newTLSConnection(t *testing.T, address string, crt, key, rootCert []byte) *grpc.ClientConn {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	tlsConfig.RootCAs = x509.NewCertPool()
	tlsConfig.RootCAs.AppendCertsFromPEM(rootCert)
	if crt != nil && key != nil {
		cert, err := tls.X509KeyPair(crt, key)
		assert.NoError(t, err)
		assert.NotNil(t, cert)

		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	}

	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

	kap := keepalive.ClientParameters{
		Time:                time.Duration(1) * time.Minute,
		Timeout:             time.Duration(20) * time.Second,
		PermitWithoutStream: true,
	}

	dialOpts = append(dialOpts, grpc.WithKeepaliveParams(kap))

	ctx, cancel := context.WithTimeout(context.Background(), (5 * time.Second))
	defer cancel()
	conn, err := grpc.DialContext(ctx, address, dialOpts...)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	return conn
}

func TestTLSClientWithChaincodeServer(t *testing.T) {
	rootPool := x509.NewCertPool()
	ok := rootPool.AppendCertsFromPEM([]byte(clientRootPEM))
	if !ok {
		t.Fatal("failed to create test root cert pool")
	}

	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		t.Fatalf("Failed to load client cert pair: %s", err)
	}

	tlsServerConfig := &tls.Config{
		MinVersion:             tls.VersionTLS12,
		Certificates:           []tls.Certificate{cert},
		ClientCAs:              rootPool,
		SessionTicketsDisabled: true,
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
	}

	// given server is good and expects valid TLS connection, test good and invalid scenarios
	var tlsTests = []struct {
		name           string
		issrv          bool
		clientKey      []byte
		clientCert     []byte
		clientRootCert []byte
		expected       *tls.Config
		errMsg         string
		success        bool
		address        string
	}{
		{
			name:           "Good TLS",
			issrv:          true,
			clientKey:      []byte(clientKeyPEM),
			clientCert:     []byte(clientCertPEM),
			clientRootCert: []byte(rootPEM),
			success:        true,
			address:        "127.0.0.1:0",
		},
		{
			name:           "Bad server RootCA",
			issrv:          true,
			clientKey:      []byte(clientKeyPEM),
			clientCert:     []byte(clientCertPEM),
			clientRootCert: []byte(clientRootPEM),
			success:        false,
			errMsg:         "transport: authentication handshake failed: x509: certificate signed by unknown authority",
			address:        "127.0.0.1:0",
		},
		{
			name:           "Bad client cert",
			issrv:          true,
			clientKey:      []byte(keyPEM),
			clientCert:     []byte(certPEM),
			clientRootCert: []byte(rootPEM),
			success:        false,
			errMsg:         "all SubConns are in TransientFailure",
			address:        "127.0.0.1:0",
		},
		{
			name:           "No client cert",
			issrv:          true,
			clientRootCert: []byte(rootPEM),
			success:        false,
			errMsg:         "all SubConns are in TransientFailure",
			address:        "127.0.0.1:0",
		},
	}

	for _, test := range tlsTests {
		t.Run(test.name, func(t *testing.T) {
			srv, err := NewServer(test.address, tlsServerConfig, nil)
			if err != nil {
				t.Fatalf("error creating server for test: %v", err)
			}
			defer srv.Stop()
			go srv.Start()

			conn := newTLSConnection(t, srv.Listener.Addr().String(), test.clientCert, test.clientKey, test.clientRootCert)
			assert.NotNil(t, conn)

			ccclient := peerpb.NewChaincodeClient(conn)
			assert.NotNil(t, ccclient)

			stream, err := ccclient.Connect(context.Background())
			if test.success {
				assert.NoError(t, err)
				assert.NotNil(t, stream)
			} else {
				assert.Error(t, err)
				assert.Regexp(t, test.errMsg, err.Error())
			}
		})
	}
}

func cleanupEnv() {
	os.Unsetenv("CORE_PEER_TLS_ENABLED")
	os.Unsetenv("CORE_TLS_CLIENT_KEY_PATH")
	os.Unsetenv("CORE_TLS_CLIENT_CERT_PATH")
	os.Unsetenv("CORE_PEER_TLS_ROOTCERT_FILE")
	os.Unsetenv("CORE_CHAINCODE_ID_NAME")
}
