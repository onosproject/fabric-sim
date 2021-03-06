// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package device implements the simulated device agent NB
package device

import (
	gnmisim "github.com/onosproject/fabric-sim/pkg/northbound/device/gnmi/v2"
	p4rtsim "github.com/onosproject/fabric-sim/pkg/northbound/device/p4runtime/v1"
	"github.com/onosproject/fabric-sim/pkg/simulator"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	gnmiapi "github.com/openconfig/gnmi/proto/gnmi"
	p4rtapi "github.com/p4lang/p4runtime/go/p4/v1"

	"google.golang.org/grpc"
)

var log = logging.GetLogger("northbound", "device")

// Service implements gNMI and P4Runtime services for a specified device
type Service struct {
	northbound.Service
	deviceID   simapi.DeviceID
	simulation *simulator.Simulation
}

// Register registers the gNMI and P4Runtime with the given gRPC server
func (s Service) Register(r *grpc.Server) {
	p4rtapi.RegisterP4RuntimeServer(r, p4rtsim.NewServer(s.deviceID, s.simulation))
	gnmiapi.RegisterGNMIServer(r, gnmisim.NewServer(s.deviceID, s.simulation))
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
	a.server = northbound.NewServer(northbound.NewServerCfg(
		"",
		"",
		"",
		int16(deviceSim.Device.ControlPort),
		true,
		northbound.SecurityConfig{
			AuthenticationEnabled: false,
			AuthorizationEnabled:  false,
		}))
	a.server.AddService(Service{
		deviceID:   deviceSim.Device.ID,
		simulation: simulation,
	})

	doneCh := make(chan error)
	go func() {
		err := a.server.Serve(func(started string) {
			log.Infof("Device %s: Started simulated device NBI on %s", deviceSim.Device.ID, started)
			close(doneCh)
		})
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
