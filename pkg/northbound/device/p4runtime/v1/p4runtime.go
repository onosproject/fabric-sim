// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package p4runtime implements the simulated P4Runtime service
package p4runtime

import (
	"context"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	p4rtapi "github.com/p4lang/p4runtime/go/p4/v1"
)

var log = logging.GetLogger("northbound", "device", "p4runtime")

// Server implements the P4Runtime API
type Server struct {
	deviceID simapi.DeviceID
}

// Write :
func (s *Server) Write(ctx context.Context, request *p4rtapi.WriteRequest) (*p4rtapi.WriteResponse, error) {
	//TODO implement me
	panic("implement me")
}

// Read :
func (s *Server) Read(request *p4rtapi.ReadRequest, server p4rtapi.P4Runtime_ReadServer) error {
	//TODO implement me
	panic("implement me")
}

// SetForwardingPipelineConfig :
func (s *Server) SetForwardingPipelineConfig(ctx context.Context, request *p4rtapi.SetForwardingPipelineConfigRequest) (*p4rtapi.SetForwardingPipelineConfigResponse, error) {
	log.Infof("Device %s: Forwarding pipeline configuration has been set", s.deviceID)
	panic("implement me")
}

// GetForwardingPipelineConfig :
func (s *Server) GetForwardingPipelineConfig(ctx context.Context, request *p4rtapi.GetForwardingPipelineConfigRequest) (*p4rtapi.GetForwardingPipelineConfigResponse, error) {
	//TODO implement me
	panic("implement me")
}

// StreamChannel :
func (s *Server) StreamChannel(server p4rtapi.P4Runtime_StreamChannelServer) error {
	//TODO implement me
	panic("implement me")
}

// Capabilities :
func (s *Server) Capabilities(ctx context.Context, request *p4rtapi.CapabilitiesRequest) (*p4rtapi.CapabilitiesResponse, error) {
	log.Infof("Device %s: P4Runtime capabilities have been requested", s.deviceID)
	panic("implement me")
}
