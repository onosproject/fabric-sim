// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package device implements the simulated device agent NB
package device

import (
	"context"
	"fmt"
	gnoisim "github.com/onosproject/fabric-sim/pkg/northbound/device/gnoi/v2"
	"github.com/onosproject/fabric-sim/pkg/northbound/device/p4runtime/v1"
	"github.com/onosproject/fabric-sim/pkg/simulator"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"github.com/onosproject/onos-net-lib/pkg/gnmiserver"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	gnoiapi "github.com/openconfig/gnoi/system"
	p4rtapi "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

var log = logging.GetLogger("northbound", "device")

// Service implements gNMI and P4Runtime services for a specified device
type Service struct {
	northbound.Service
	deviceID   simapi.DeviceID
	simulation *simulator.Simulation
	deviceSim  *simulator.DeviceSimulator
}

// Register registers the gNMI and P4Runtime with the given gRPC server
func (s Service) Register(r *grpc.Server) {
	gnmiServer := gnmiserver.NewGNMIServer(&s.deviceSim.GNMIConfigurable, fmt.Sprintf("Device %s", s.deviceID))
	gnmiapi.RegisterGNMIServer(r, gnmiServer)
	gnoiapi.RegisterSystemServer(r, gnoisim.NewServer(s.deviceID, s.simulation))
	p4rtapi.RegisterP4RuntimeServer(r, p4runtime.NewServer(s.deviceID, s.simulation))
	log.Debugf("Device %s: P4Runtime and gNMI registered", s.deviceID)
}

// NewAgent creates a new simulated device agent
func NewAgent() simulator.DeviceAgent {
	return &agent{}
}

// Implementation of DeviceAgent interface
type agent struct {
	server *northbound.Server
}

// Start starts the simulated device agent
func (a *agent) Start(simulation *simulator.Simulation, deviceSim *simulator.DeviceSimulator) error {
	a.server = northbound.NewServer(northbound.NewInsecureServerConfig(int16(deviceSim.Device.ControlPort)))
	a.server.AddService(Service{
		deviceID:   deviceSim.Device.ID,
		simulation: simulation,
		deviceSim:  deviceSim,
	})

	doneCh := make(chan error)
	go func() {
		const maxMessageSize = 16 * 1024 * 1024
		grpcOpts := []grpc.ServerOption{
			grpc.MaxRecvMsgSize(maxMessageSize),
			grpc.MaxSendMsgSize(maxMessageSize),
			grpc.StatsHandler(&statsHandler{deviceSim: deviceSim}),
		}
		err := a.server.Serve(func(started string) {
			log.Infof("Device %s: Started simulated device NBI on %s", deviceSim.Device.ID, started)
			close(doneCh)
		}, grpcOpts...)
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}

// Stop stops the simulated device agent
func (a *agent) Stop(mode simapi.StopMode) error {
	if mode == simapi.StopMode_ORDERLY_STOP {
		a.server.GracefulStop()
	} else {
		// FIXME: This is not sufficiently chaotic
		a.server.Stop()
	}
	return nil
}

// Internal handler of RPC server stats
type statsHandler struct {
	deviceSim *simulator.DeviceSimulator
}

// ConnCtxKey is a connection context key
type ConnCtxKey struct{}

// RPCCtxKey is an RPC context key
type RPCCtxKey struct{}

// TagConn tags the connection context
func (h *statsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return context.WithValue(ctx, ConnCtxKey{}, info)
}

// TagRPC tags the RPC context
func (h *statsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return context.WithValue(ctx, RPCCtxKey{}, info)
}

// HandleConn handle the connection stats
func (h *statsHandler) HandleConn(ctx context.Context, s stats.ConnStats) {
}

// HandleRPC handle RPC stats
func (h *statsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	if ih, ok := s.(*stats.InHeader); ok {
		h.deviceSim.UpdateIOStats(ih.WireLength, true)
	} else if ip, ok := s.(*stats.InPayload); ok {
		h.deviceSim.UpdateIOStats(ip.WireLength, true)
	} else if op, ok := s.(*stats.OutPayload); ok {
		h.deviceSim.UpdateIOStats(op.WireLength, false)
	} else if it, ok := s.(*stats.InTrailer); ok {
		h.deviceSim.UpdateIOStats(it.WireLength, true)
	}
}
