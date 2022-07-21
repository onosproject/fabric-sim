// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package p4runtime implements the simulated P4Runtime service
package p4runtime

import (
	"context"
	"github.com/onosproject/fabric-sim/pkg/simulator"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	p4rtapi "github.com/p4lang/p4runtime/go/p4/v1"
	"io"
)

var log = logging.GetLogger("northbound", "device", "p4runtime")

// Server implements the P4Runtime API
type Server struct {
	deviceID   simapi.DeviceID
	simulation *simulator.Simulation
	deviceSim  *simulator.DeviceSimulator
}

// NewServer creates a new P4Runtime API server
func NewServer(deviceID simapi.DeviceID, simulation *simulator.Simulation) *Server {
	sim, err := simulation.GetDeviceSimulator(deviceID)
	if err != nil {
		return nil
	}
	return &Server{
		deviceID:   deviceID,
		simulation: simulation,
		deviceSim:  sim,
	}
}

// Write :
func (s *Server) Write(ctx context.Context, request *p4rtapi.WriteRequest) (*p4rtapi.WriteResponse, error) {
	log.Infof("Device %s: Write received", s.deviceID)
	return &p4rtapi.WriteResponse{}, nil
}

// Read :
func (s *Server) Read(request *p4rtapi.ReadRequest, server p4rtapi.P4Runtime_ReadServer) error {
	log.Infof("Device %s: Read received", s.deviceID)
	entities := make([]*p4rtapi.Entity, 0, len(request.Entities))

	// Accumulate entities to respond with
	// TODO: implement this, obviously
	entities = append(entities, request.Entities...)

	// Send a response in one go
	err := server.Send(&p4rtapi.ReadResponse{Entities: entities})
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

// SetForwardingPipelineConfig :
func (s *Server) SetForwardingPipelineConfig(ctx context.Context, request *p4rtapi.SetForwardingPipelineConfigRequest) (*p4rtapi.SetForwardingPipelineConfigResponse, error) {
	log.Infof("Device %s: Forwarding pipeline configuration has been set", s.deviceID)
	s.deviceSim.ForwardingPipelineConfig = request.Config
	return &p4rtapi.SetForwardingPipelineConfigResponse{}, nil
}

// GetForwardingPipelineConfig :
func (s *Server) GetForwardingPipelineConfig(ctx context.Context, request *p4rtapi.GetForwardingPipelineConfigRequest) (*p4rtapi.GetForwardingPipelineConfigResponse, error) {
	log.Infof("Device %s: Getting pipeline configuration", s.deviceID)
	return &p4rtapi.GetForwardingPipelineConfigResponse{
		Config: s.deviceSim.ForwardingPipelineConfig,
	}, nil
}

// StreamChannel :
func (s *Server) StreamChannel(server p4rtapi.P4Runtime_StreamChannelServer) error {
	for {
		msg, err := server.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		s.processRequest(server, msg)
		s.emitResponses(server)
	}
	return nil
}

// Capabilities :
func (s *Server) Capabilities(ctx context.Context, request *p4rtapi.CapabilitiesRequest) (*p4rtapi.CapabilitiesResponse, error) {
	log.Infof("Device %s: P4Runtime capabilities have been requested", s.deviceID)
	return &p4rtapi.CapabilitiesResponse{P4RuntimeApiVersion: "1.1.0"}, nil
}

func (s *Server) processRequest(server p4rtapi.P4Runtime_StreamChannelServer, msg *p4rtapi.StreamMessageRequest) {
	log.Infof("Device %s: Received message: %+v", s.deviceID, msg)

	// Process mastership arbitration update
	// Process packet out
	// Process digest list ack
}

func (s *Server) emitResponses(server p4rtapi.P4Runtime_StreamChannelServer) {
	// TODO: Implement this

	// Send mastership arbitration update
	// Send packet out
	// Send digest list
	// Send timeout notification
	// Send errors
}
