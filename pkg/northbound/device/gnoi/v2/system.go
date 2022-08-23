// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package gnoi implements the simulated gNOI System service
package gnoi

import (
	"context"
	"github.com/onosproject/fabric-sim/pkg/simulator"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	gnoiapi "github.com/openconfig/gnoi/system"
	"time"
)

var log = logging.GetLogger("northbound", "device", "gnoi")

// Server implements the P4Runtime API
type Server struct {
	deviceID   simapi.DeviceID
	simulation *simulator.Simulation
	deviceSim  *simulator.DeviceSimulator
	gnoiapi.UnimplementedSystemServer
}

// NewServer creates a new gNOI System API server
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

func notImplemented() error {
	return errors.Status(errors.NewNotSupported("method not supported")).Err()
}

// Ping is not implemented
func (s Server) Ping(request *gnoiapi.PingRequest, server gnoiapi.System_PingServer) error {
	return notImplemented()
}

// Traceroute is not implemented
func (s Server) Traceroute(request *gnoiapi.TracerouteRequest, server gnoiapi.System_TracerouteServer) error {
	return notImplemented()
}

// Time returns device's time since start of epoch, expressed in nanoseconds
func (s Server) Time(ctx context.Context, request *gnoiapi.TimeRequest) (*gnoiapi.TimeResponse, error) {
	log.Debugf("Device %s: Received time request", s.deviceID)
	return &gnoiapi.TimeResponse{Time: uint64(time.Now().UnixNano())}, nil
}

// SetPackage is not implemented
func (s Server) SetPackage(server gnoiapi.System_SetPackageServer) error {
	return notImplemented()
}

// SwitchControlProcessor is not implemented
func (s Server) SwitchControlProcessor(ctx context.Context, request *gnoiapi.SwitchControlProcessorRequest) (*gnoiapi.SwitchControlProcessorResponse, error) {
	return nil, notImplemented()
}

// Reboot is not implemented
func (s Server) Reboot(ctx context.Context, request *gnoiapi.RebootRequest) (*gnoiapi.RebootResponse, error) {
	return nil, notImplemented()
}

// RebootStatus is not implemented
func (s Server) RebootStatus(ctx context.Context, request *gnoiapi.RebootStatusRequest) (*gnoiapi.RebootStatusResponse, error) {
	return nil, notImplemented()
}

// CancelReboot is not implemented
func (s Server) CancelReboot(ctx context.Context, request *gnoiapi.CancelRebootRequest) (*gnoiapi.CancelRebootResponse, error) {
	return nil, notImplemented()
}

// KillProcess is not implemented
func (s Server) KillProcess(ctx context.Context, request *gnoiapi.KillProcessRequest) (*gnoiapi.KillProcessResponse, error) {
	return nil, notImplemented()
}
