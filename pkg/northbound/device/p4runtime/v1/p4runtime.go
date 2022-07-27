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
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
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
func (s *Server) Capabilities(ctx context.Context, request *p4api.CapabilitiesRequest) (*p4api.CapabilitiesResponse, error) {
	log.Infof("Device %s: P4Runtime capabilities have been requested", s.deviceID)
	return &p4api.CapabilitiesResponse{P4RuntimeApiVersion: "1.1.0"}, nil
}

// Write :
func (s *Server) Write(ctx context.Context, request *p4api.WriteRequest) (*p4api.WriteResponse, error) {
	log.Infof("Device %s: Write received", s.deviceID)
	// TODO: implement this
	return &p4api.WriteResponse{}, nil
}

// Read :
func (s *Server) Read(request *p4api.ReadRequest, server p4api.P4Runtime_ReadServer) error {
	log.Infof("Device %s: Read received", s.deviceID)
	entities := make([]*p4api.Entity, 0, len(request.Entities))

	// TODO: implement this for real
	// Accumulate entities to respond with
	entities = append(entities, request.Entities...)

	// Send a response in one go
	err := server.Send(&p4api.ReadResponse{Entities: entities})
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

// SetForwardingPipelineConfig :
func (s *Server) SetForwardingPipelineConfig(ctx context.Context, request *p4api.SetForwardingPipelineConfigRequest) (*p4api.SetForwardingPipelineConfigResponse, error) {
	log.Infof("Device %s: Forwarding pipeline configuration has been set", s.deviceID)
	s.deviceSim.ForwardingPipelineConfig = request.Config
	return &p4api.SetForwardingPipelineConfigResponse{}, nil
}

// GetForwardingPipelineConfig :
func (s *Server) GetForwardingPipelineConfig(ctx context.Context, request *p4api.GetForwardingPipelineConfigRequest) (*p4api.GetForwardingPipelineConfigResponse, error) {
	log.Infof("Device %s: Getting pipeline configuration", s.deviceID)
	return &p4api.GetForwardingPipelineConfigResponse{
		Config: s.deviceSim.ForwardingPipelineConfig,
	}, nil
}

// State related to a single message stream
type streamState struct {
	arbitration     *p4api.MasterArbitrationUpdate
	streamResponses chan *p4api.StreamMessageResponse
}

// Send queues up the specified response to asynchronously send to the backing stream
func (state *streamState) Send(response *p4api.StreamMessageResponse) {
	state.streamResponses <- response
}

// RecordMastershipArbitration records the mastership arbitration
func (state *streamState) RecordMastershipArbitration(arbitration *p4api.MasterArbitrationUpdate) *p4api.MasterArbitrationUpdate {
	if arbitration != nil {
		state.arbitration = arbitration
	}
	return arbitration
}

// StreamChannel reads and handles incoming requests and emits any queued up outgoing responses
func (s *Server) StreamChannel(server p4api.P4Runtime_StreamChannelServer) error {
	// Create and register a new record to track the state of this stream
	responder := &streamState{
		streamResponses: make(chan *p4api.StreamMessageResponse, 128),
	}
	s.deviceSim.AddStreamResponder(responder)
	defer s.deviceSim.RemoveStreamResponder(responder)

	// Emit any queued-up messages in the background until we get an error or the context is closed
	go func() {
		for msg := range responder.streamResponses {
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

	// Read messages from the stream in the foreground (until we get an error or EOF) and process them
	for {
		msg, err := server.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		s.processRequest(responder, msg)
	}
	return nil
}

func (s *Server) processRequest(responder simulator.StreamResponder, msg *p4api.StreamMessageRequest) {
	log.Debugf("Device %s: Received message: %+v", s.deviceID, msg)

	// If the message is a packet out, process it
	if packet := msg.GetPacket(); packet != nil {
		s.deviceSim.ProcessPacketOut(packet, responder)
		return
	}

	// If the message is a mastership arbitration, record it and process it
	if arbitration := responder.RecordMastershipArbitration(msg.GetArbitration()); arbitration != nil {
		s.deviceSim.ProcessMastershipArbitration(arbitration, responder)
		return
	}

	// Process digest list ack
	if digestAck := msg.GetDigestAck(); digestAck != nil {
		s.deviceSim.ProcessDigestAck(digestAck, responder)
		return
	}
}
