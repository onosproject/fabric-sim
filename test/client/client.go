// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package client contains number of integration test utilities
package client

import (
	"fmt"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	testutils "github.com/onosproject/onos-lib-go/pkg/test"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CreateConnection creates gRPC connection to the fabric simulator
func CreateConnection() (*grpc.ClientConn, error) {
	return testutils.CreateConnection("fabric-sim:5150", true)
}

// CreateDeviceConnection creates connection to the device agent.
func CreateDeviceConnection(device *simapi.Device) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.Dial(fmt.Sprintf("fabric-sim:%d", device.ControlPort), opts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
