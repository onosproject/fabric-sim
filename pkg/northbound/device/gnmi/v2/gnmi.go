// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package gnmi implements the simulated gNMI service
package gnmi

import (
	"context"
	"github.com/onosproject/fabric-sim/pkg/simulator"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
	"io"
)

var log = logging.GetLogger("northbound", "device", "gnmi")

// Server implements the P4Runtime API
type Server struct {
	deviceID   simapi.DeviceID
	simulation *simulator.Simulation
	deviceSim  *simulator.DeviceSimulator
}

// NewServer creates a new gNMI API server
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

// Capabilities allows the client to retrieve the set of capabilities that
// is supported by the target. This allows the target to validate the
// service version that is implemented and retrieve the set of models that
// the target supports. The models can then be specified in subsequent RPCs
// to restrict the set of data that is utilized.
// Reference: gNMI Specification Section 3.2
func (s *Server) Capabilities(ctx context.Context, request *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	log.Infof("Device %s: gNMI capabilities have been requested", s.deviceID)
	// TODO: populate appropriately with supported models; for now, this is not required
	modelData := make([]*gnmi.ModelData, 0)
	return &gnmi.CapabilityResponse{
		SupportedModels:    modelData,
		SupportedEncodings: []gnmi.Encoding{gnmi.Encoding_PROTO, gnmi.Encoding_JSON_IETF},
		GNMIVersion:        "0.8.0",
	}, nil
}

// Get retrieves a snapshot of data from the target. A Get RPC requests that the
// target snapshots a subset of the data tree as specified by the paths
// included in the message and serializes this to be returned to the
// client using the specified encoding.
// Reference: gNMI Specification Section 3.3
func (s *Server) Get(ctx context.Context, request *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	log.Infof("Device %s: gNMI get request received", s.deviceID)
	notifications, err := s.deviceSim.ProcessConfigGet(request.Prefix, request.Path)
	if err != nil {
		return nil, errors.Status(err).Err()
	}
	return &gnmi.GetResponse{
		Notification: notifications,
	}, nil
}

// Set allows the client to modify the state of data on the target. The
// paths to modified along with the new values that the client wishes
// to set the value to.
// Reference: gNMI Specification Section 3.4
func (s *Server) Set(ctx context.Context, request *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	log.Infof("Device %s: gNMI set request received", s.deviceID)
	results, err := s.deviceSim.ProcessConfigSet(request.Prefix, request.Update, request.Replace, request.Delete)
	if err != nil {
		return nil, errors.Status(err).Err()
	}
	return &gnmi.SetResponse{
		Prefix:    request.Prefix,
		Response:  results,
		Timestamp: 0,
	}, nil
}

type subContext struct {
	stream gnmi.GNMI_SubscribeServer
	req    *gnmi.SubscribeRequest
}

// Subscribe allows a client to request the target to send it values
// of particular paths within the data tree. These values may be streamed
// at a particular cadence (STREAM), sent one off on a long-lived channel
// (POLL), or sent as a one-off retrieval (ONCE).
// Reference: gNMI Specification Section 3.5
func (s *Server) Subscribe(server gnmi.GNMI_SubscribeServer) error {
	log.Infof("Device %s: gNMI subscribe request received", s.deviceID)

	sctx := &subContext{stream: server}

	log.Info("Waiting for subscription messages")
	for {
		req, err := server.Recv()
		if err != nil {
			if err != io.EOF {
				log.Info("Client closed the subscription stream")
				return nil
			}
			// Cancel SB requests and exit with error
			log.Warn(err)
			return err
		}

		log.Infof("Received gNMI Subscribe Request: %+v", req)
		err = s.processSubscribeRequest(server.Context(), sctx, req)
		if err != nil {
			log.Warn(err)
			return err
		}
	}
}

func (s *Server) processSubscribeRequest(ctx context.Context, sctx *subContext, request *gnmi.SubscribeRequest) error {
	if request.GetSubscribe() != nil && sctx.req != nil {
		return errors.NewInvalid("duplicate subscription message detected")
	} else if request.GetPoll() != nil && sctx.req == nil {
		return errors.NewInvalid("subscription request not received yet")

	} else if request.GetSubscribe() != nil {
		// If the request is the subscription, remember it
		sctx.req = request
		subscribe := request.GetSubscribe()
		// TODO: Implement various modes of retrieval
		switch subscribe.Mode {
		case gnmi.SubscriptionList_ONCE:
			return s.processSubscribeOnce(ctx, sctx, subscribe)
		case gnmi.SubscriptionList_STREAM:
			return s.processSubscribeStream(ctx, sctx, subscribe)
		case gnmi.SubscriptionList_POLL:
			return s.processSubscribePoll(ctx, sctx, subscribe)
		}

	} else if request.GetPoll() != nil {
		// TODO: If the request is a poll, go fetch the source device

	} else {
		return errors.NewInvalid("unknown subscription message type")
	}

	return nil
}

func (s *Server) processSubscribeOnce(ctx context.Context, sctx *subContext, subscribe *gnmi.SubscriptionList) error {
	paths := make([]*gnmi.Path, 0, len(subscribe.Subscription))
	for _, s := range subscribe.Subscription {
		paths = append(paths, s.Path)
	}
	notifications, err := s.deviceSim.ProcessConfigGet(subscribe.Prefix, paths)
	// TODO: implement proper error handling; for now, just return what we got back
	for _, notification := range notifications {
		err = sctx.stream.Send(&gnmi.SubscribeResponse{
			Response: &gnmi.SubscribeResponse_Update{Update: notification},
		})
		if err != nil {
			return err
		}
	}
	return err
}

func (s *Server) processSubscribePoll(ctx context.Context, sctx *subContext, subscribe *gnmi.SubscriptionList) error {
	return nil
}

func (s *Server) processSubscribeStream(ctx context.Context, sctx *subContext, subscribe *gnmi.SubscriptionList) error {
	return nil
}
