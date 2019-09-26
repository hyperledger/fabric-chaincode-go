/*
Copyright State Street Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package internal_test

import (
	"net"
	"testing"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim/internal"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/keepalive"
)

func TestBadServer(t *testing.T) {
	srv := &internal.Server{}
	err := srv.Start()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "nil listener")

	l, err := net.Listen("tcp", ":0")
	assert.NotNil(t, l)
	assert.Nil(t, err)
	srv = &internal.Server{Listener: l}
	err = srv.Start()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "nil server")
}

func TestServerAddressNotProvided(t *testing.T) {
	kaOpts := &keepalive.ServerParameters{
		Time:    1 * time.Minute,
		Timeout: 20 * time.Second,
	}
	srv, err := internal.NewServer("", nil, kaOpts)
	assert.Nil(t, srv)
	assert.NotNil(t, err, "server listen address not provided")
}

func TestBadServerAddress(t *testing.T) {
	kaOpts := &keepalive.ServerParameters{
		Time:    1 * time.Minute,
		Timeout: 20 * time.Second,
	}
	srv, err := internal.NewServer("__badhost__:0", nil, kaOpts)
	assert.Nil(t, srv)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "listen tcp: lookup __badhost__")

	srv, err = internal.NewServer("host", nil, kaOpts)
	assert.Nil(t, srv)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "listen tcp: address host: missing port in address")
}
