// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package p4runtime implements the simulated P4Runtime service
package p4runtime

import (
	"context"
	gogo "github.com/gogo/protobuf/types"
	"github.com/onosproject/fabric-sim/pkg/simulator"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-api/go/onos/stratum"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/peer"
	"io"
	"time"
)

var log = logging.GetLogger("northbound", "device", "p4runtime")

// Server implements the P4Runtime API
type Server struct {
	deviceID   simapi.DeviceID
	simulation *simulator.Simulation
	deviceSim  *simulator.DeviceSimulator
	p4api.UnimplementedP4RuntimeServer
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

// Write applies a set of updates to the device
func (s *Server) Write(ctx context.Context, request *p4api.WriteRequest) (*p4api.WriteResponse, error) {
	log.Infof("Device %s: Write received", s.deviceID)
	if err := s.checkMastership(request.DeviceId, request.Role, request.ElectionId); err != nil {
		return nil, errors.Status(err).Err()
	}
	if err := s.checkForwardingPipeline(); err != nil {
		return nil, errors.Status(err).Err()
	}
	if err := s.deviceSim.ProcessWrite(request.Atomicity, request.Updates); err != nil {
		return nil, errors.Status(err).Err()
	}
	return &p4api.WriteResponse{}, nil
}

// Makes sure that the specified role and election ID have mastership over the given device; returns error if not
func (s *Server) checkMastership(deviceID uint64, role string, electionID *p4api.Uint128) error {
	return s.deviceSim.IsMaster(deviceID, role, electionID)
}

// Makes sure that the forwarding pipeline has been set fo the device
func (s *Server) checkForwardingPipeline() error {
	if s.deviceSim.GetPipelineConfig() == nil {
		return errors.NewConflict("forwarding pipeline not set for %s", s.deviceID)
	}
	return nil
}

// Read receives a query and stream back all requested entities
func (s *Server) Read(request *p4api.ReadRequest, server p4api.P4Runtime_ReadServer) error {
	log.Infof("Device %s: Read received", s.deviceID)

	// Process the read, sending results using the supplied batch sender function
	_ = s.deviceSim.ProcessRead(request.Entities, func(entities []*p4api.Entity) error {
		return server.Send(&p4api.ReadResponse{Entities: entities})
	})

	// TODO: accumulate batch errors into details
	return errors.Status(nil).Err()
}

// SetForwardingPipelineConfig sets the forwarding pipeline configuration
func (s *Server) SetForwardingPipelineConfig(ctx context.Context, request *p4api.SetForwardingPipelineConfigRequest) (*p4api.SetForwardingPipelineConfigResponse, error) {
	log.Infof("Device %s: Forwarding pipeline configuration has been set", s.deviceID)
	if err := s.checkMastership(request.DeviceId, request.Role, request.ElectionId); err != nil {
		return nil, errors.Status(err).Err()
	}
	if err := s.deviceSim.SetPipelineConfig(request.Config); err != nil {
		return nil, errors.Status(err).Err()
	}
	return &p4api.SetForwardingPipelineConfigResponse{}, nil
}

// GetForwardingPipelineConfig retrieves the current forwarding pipeline configuration
func (s *Server) GetForwardingPipelineConfig(ctx context.Context, request *p4api.GetForwardingPipelineConfigRequest) (*p4api.GetForwardingPipelineConfigResponse, error) {
	log.Infof("Device %s: Getting pipeline configuration", s.deviceID)
	config := s.deviceSim.GetPipelineConfig()
	switch request.ResponseType {
	case p4api.GetForwardingPipelineConfigRequest_COOKIE_ONLY:
		config.P4Info = nil
		config.P4DeviceConfig = nil
	case p4api.GetForwardingPipelineConfigRequest_P4INFO_AND_COOKIE:
		config.P4DeviceConfig = nil
	case p4api.GetForwardingPipelineConfigRequest_DEVICE_CONFIG_AND_COOKIE:
		config.P4Info = nil
	}
	return &p4api.GetForwardingPipelineConfigResponse{
		Config: config,
	}, nil
}

// State related to a single message stream
type streamState struct {
	deviceID        uint64
	role            *p4api.Role
	roleConfig      *stratum.P4RoleConfig
	electionID      *p4api.Uint128
	sentCode        *int32
	streamResponses chan *p4api.StreamMessageResponse
	connection      *simapi.Connection
}

// Send queues up the specified response to asynchronously send to the backing stream
func (state *streamState) Send(response *p4api.StreamMessageResponse) {
	state.streamResponses <- response
}

func (state *streamState) SendMastershipArbitration(role *p4api.Role, masterElectionID *p4api.Uint128, failCode code.Code) {
	if role != state.role {
		return
	}

	// Send failed election status code unless we are the master
	electionStatus := &status.Status{Code: int32(failCode)}
	if state.electionID == masterElectionID && state.role == role {
		electionStatus.Code = int32(code.Code_OK)
	}

	// Send only if we haven't sent this code previously
	if state.sentCode == nil || *state.sentCode != electionStatus.Code {
		state.Send(&p4api.StreamMessageResponse{
			Update: &p4api.StreamMessageResponse_Arbitration{
				Arbitration: &p4api.MasterArbitrationUpdate{
					DeviceId:   state.deviceID,
					Role:       state.role,
					ElectionId: masterElectionID,
					Status:     electionStatus,
				},
			},
		})
		state.sentCode = &electionStatus.Code
	}
}

// LatchMastershipArbitration record the mastership arbitration role and election ID if the arbitration update is not nil
func (state *streamState) LatchMastershipArbitration(arbitration *p4api.MasterArbitrationUpdate) *p4api.MasterArbitrationUpdate {
	if arbitration != nil {
		state.deviceID = arbitration.DeviceId
		state.role = arbitration.Role
		state.electionID = arbitration.ElectionId

		if arbitration.Role != nil && arbitration.Role.Config != nil {
			state.roleConfig = &stratum.P4RoleConfig{ReceivesPacketIns: true}
			any := &gogo.Any{TypeUrl: state.role.Config.TypeUrl, Value: state.role.Config.Value}
			_ = gogo.UnmarshalAny(any, state.roleConfig)
		}
	}
	return arbitration
}

// IsMaster returns true if the responder is the current master, i.e. has the master election ID, for the given role.
func (state *streamState) IsMaster(role *p4api.Role, masterElectionID *p4api.Uint128) bool {
	return (state.role == role || (state.role != nil && role != nil && state.role.Name == role.Name)) &&
		state.electionID.High == masterElectionID.High && state.electionID.Low == masterElectionID.Low
}

// GetRoleConfig returns the stratum role configuration received during role arbitration; nil if none
func (state *streamState) GetRoleConfig() *stratum.P4RoleConfig {
	return state.roleConfig
}

// GetConnection returns the peer connection info for the stream channel
func (state *streamState) GetConnection() *simapi.Connection {
	return state.connection
}

// StreamChannel reads and handles incoming requests and emits any queued up outgoing responses
func (s *Server) StreamChannel(server p4api.P4Runtime_StreamChannelServer) error {
	log.Infof("Device %s: Received stream channel request", s.deviceID)

	// Create and register a new record to track the state of this stream
	responder := &streamState{
		streamResponses: make(chan *p4api.StreamMessageResponse, 128),
	}
	if p, ok := peer.FromContext(server.Context()); ok {
		responder.connection = &simapi.Connection{
			FromAddress: p.Addr.String(),
			Protocol:    "p4rt",
			Time:        time.Now().Unix(),
		}
	}
	s.deviceSim.AddStreamResponder(responder)

	// On stream closure, remove the responder and run mastership arbitration
	defer func() {
		s.deviceSim.RemoveStreamResponder(responder)
		_ = s.deviceSim.RunMastershipArbitration(responder.role, responder.electionID)
	}()

	// Emit any queued-up messages in the background until we get an error or the context is closed
	go func() {
		for msg := range responder.streamResponses {
			log.Infof("Device %s NB: Sending message to %s: %+v",
				s.deviceID, responder.connection.FromAddress, msg)
			if err := server.Send(msg); err != nil {
				log.Warnf("Device %s NB: Unable to send message... closing connection", s.deviceID)
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
			return errors.Status(err).Err()
		}
		if err = s.processRequest(responder, msg); err != nil {
			return errors.Status(err).Err()
		}
	}

	return nil
}

func (s *Server) processRequest(responder simulator.StreamResponder, msg *p4api.StreamMessageRequest) error {
	log.Debugf("Device %s: Received message: %+v", s.deviceID, msg)

	// If the message is a packet out, process it
	if packet := msg.GetPacket(); packet != nil {
		return s.deviceSim.ProcessPacketOut(packet, responder)
	}

	// If the message is a mastership arbitration, record it and process it
	if arbitration := responder.LatchMastershipArbitration(msg.GetArbitration()); arbitration != nil {
		return s.deviceSim.RunMastershipArbitration(arbitration.Role, arbitration.ElectionId)
	}

	// Process digest list ack
	if digestAck := msg.GetDigestAck(); digestAck != nil {
		return s.deviceSim.ProcessDigestAck(digestAck, responder)
	}

	return nil
}
