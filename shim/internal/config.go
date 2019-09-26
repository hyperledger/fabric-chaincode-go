// Copyright the Hyperledger Fabric contributors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc/keepalive"
)

// Config ...
type Config struct {
	ChaincodeName string
	TLS           *tls.Config
	KaOpts        keepalive.ClientParameters
	ServerKaOpts  keepalive.ServerParameters
}

// LoadConfig ...
func LoadConfig(isserver bool) (Config, error) {
	tlsEnabled, err := strconv.ParseBool(os.Getenv("CORE_PEER_TLS_ENABLED"))
	if err != nil {
		return Config{}, errors.New("'CORE_PEER_TLS_ENABLED' must be set to 'true' or 'false'")
	}

	conf := Config{
		ChaincodeName: os.Getenv("CORE_CHAINCODE_ID_NAME"),
		// if client, use this ... hardcode to match peer
		KaOpts: keepalive.ClientParameters{
			Time:                1 * time.Minute,
			Timeout:             20 * time.Second,
			PermitWithoutStream: true,
		},
		// if server, use this... hardcode to match peer
		ServerKaOpts: keepalive.ServerParameters{
			Time:    1 * time.Minute,
			Timeout: 20 * time.Second,
		},
	}

	if !tlsEnabled {
		return conf, nil
	}

	data, err := ioutil.ReadFile(os.Getenv("CORE_TLS_CLIENT_KEY_PATH"))
	if err != nil {
		return Config{}, fmt.Errorf("failed to read private key file: %s", err)
	}
	key, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return Config{}, fmt.Errorf("failed to decode private key file: %s", err)
	}
	data, err = ioutil.ReadFile(os.Getenv("CORE_TLS_CLIENT_CERT_PATH"))
	if err != nil {
		return Config{}, fmt.Errorf("failed to read public key file: %s", err)
	}
	cert, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return Config{}, fmt.Errorf("failed to decode public key file: %s", err)
	}

	var rootCertPool *x509.CertPool
	rootCAs := os.Getenv("CORE_PEER_TLS_ROOTCERT_FILE")
	if rootCAs == "" {
		//as a client, Peer CA must be provided
		if !isserver {
			return Config{}, fmt.Errorf("root cert file not provided for chaincode")
		}
	} else {
		root, err := ioutil.ReadFile(os.Getenv("CORE_PEER_TLS_ROOTCERT_FILE"))
		if err != nil {
			return Config{}, fmt.Errorf("failed to read root cert file: %s", err)
		}
		rootCertPool = x509.NewCertPool()
		if ok := rootCertPool.AppendCertsFromPEM(root); !ok {
			return Config{}, errors.New("failed to load root cert file")
		}
	}

	clientCert, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return Config{}, errors.New("failed to parse client key pair")
	}

	conf.TLS = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      rootCertPool,
	}

	//follow Peer's server default config properties
	if isserver {
		conf.TLS.SessionTicketsDisabled = true
		conf.TLS.CipherSuites = []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		}
		if rootCertPool != nil {
			conf.TLS.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}

	return conf, nil
}
