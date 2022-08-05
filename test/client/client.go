// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package client contains number of integration test utilities
package client

import (
	"crypto/tls"
	"fmt"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/grpc/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GetClientCredentials returns client credentials
func GetClientCredentials() (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(certs.DefaultClientCrt), []byte(certs.DefaultClientKey))
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}, nil
}

// CreateConnection creates gRPC connection to the fabric simulator
func CreateConnection() (*grpc.ClientConn, error) {
	tlsConfig, err := GetClientCredentials()
	if err != nil {
		return nil, err
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor()),
	}

	conn, err := grpc.Dial("fabric-sim:5150", opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// CreateDeviceConnection creates connection to the device agent.
func CreateDeviceConnection(device *simapi.Device) (*grpc.ClientConn, error) {
	tlsConfig, err := GetClientCredentials()
	if err != nil {
		return nil, err
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
	}

	conn, err := grpc.Dial(fmt.Sprintf("fabric-sim:%d", device.ControlPort), opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
