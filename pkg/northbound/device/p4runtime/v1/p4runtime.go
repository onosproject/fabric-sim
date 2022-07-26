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
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
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

// Capabilities responds with the device P4Runtime capabilities
func (s *Server) Capabilities(ctx context.Context, request *p4rtapi.CapabilitiesRequest) (*p4rtapi.CapabilitiesResponse, error) {
	log.Infof("Device %s: P4Runtime capabilities have been requested", s.deviceID)
	return &p4rtapi.CapabilitiesResponse{P4RuntimeApiVersion: "1.1.0"}, nil
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

type channelState struct {
	arbitration     *p4rtapi.MasterArbitrationUpdate
	streamResponses chan *p4rtapi.StreamMessageResponse
}

// StreamChannel reads and handles incoming requests and emits any queued up outgoing responses
func (s *Server) StreamChannel(server p4rtapi.P4Runtime_StreamChannelServer) error {
	state := &channelState{
		streamResponses: make(chan *p4rtapi.StreamMessageResponse, 128),
	}

	// Emit any queued-up messages in the background until we get an error or the context is closed
	go func() {
		for msg := range state.streamResponses {
			if err := server.Send(msg); err != nil {
				return
			}
			select {
			case <-server.Context().Done():
				return
			default:
			}
		}
	}()

	for {
		msg, err := server.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		s.processRequest(state, msg)
	}
	return nil
}

func (s *Server) processRequest(state *channelState, msg *p4rtapi.StreamMessageRequest) {
	log.Infof("Device %s: Received message: %+v", s.deviceID, msg)

	// Process mastership arbitration update
	if state.arbitration == nil {
		if arbitration := msg.GetArbitration(); arbitration != nil {
			// Record the arbitration in our channel state and respond to it
			state.arbitration = arbitration

			electionStatus := &status.Status{Code: int32(code.Code_OK)}
			maxElectionID, err := s.deviceSim.RecordRoleElection(arbitration.Role, arbitration.ElectionId)
			if err != nil {
				electionStatus.Code = int32(code.Code_PERMISSION_DENIED)
				electionStatus.Message = err.Error()
			}
			state.streamResponses <- &p4rtapi.StreamMessageResponse{
				Update: &p4rtapi.StreamMessageResponse_Arbitration{
					Arbitration: &p4rtapi.MasterArbitrationUpdate{
						DeviceId:   arbitration.DeviceId,
						Role:       arbitration.Role,
						ElectionId: maxElectionID,
						Status:     electionStatus,
					},
				},
			}
		}
		return
	}

	// Process packet out
	if packet := msg.GetPacket(); packet != nil {
		// TODO: Handle the packet outs
		log.Infof("Device %s: packet out: %+v", s.deviceID, msg.GetPacket())
	}

	// Process digest list ack
	if digestAck := msg.GetDigestAck(); digestAck != nil {
		// TODO: Handle the digest list acks
		log.Infof("Device %s: digest ack: %+v", s.deviceID, msg.GetDigestAck())
	}
}
