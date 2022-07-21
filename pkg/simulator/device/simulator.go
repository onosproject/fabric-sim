// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package device implements the device simulator control logic
package device

import (
	simnb "github.com/onosproject/fabric-sim/pkg/northbound/device"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
)

var log = logging.GetLogger("simulator", "device")

// Simulator simulates a single device
type Simulator struct {
	Device *simapi.Device
	Agent  *northbound.Server
	Ports  map[simapi.PortID]*simapi.Port
}

// NewDeviceSimulator initializes a new device simulator
func NewDeviceSimulator(device *simapi.Device) *Simulator {
	log.Infof("Device %s: Creating simulator", device.ID)

	// Build a port map
	ports := make(map[simapi.PortID]*simapi.Port)
	for _, port := range device.Ports {
		ports[port.ID] = port
	}

	// Construct and return simulator from the device and the port map
	sim := Simulator{
		Device: device,
		Ports:  ports,
	}
	return &sim
}

// Start spawns the device simulator background tasks and its agent API server, also in the background
func (ds *Simulator) Start() error {
	log.Infof("Device %s: Starting simulator", ds.Device.ID)

	// Start any background simulation tasks

	// Starts the simulated device agent
	err := ds.startSimulationAgent()
	if err != nil {
		log.Errorf("Device %s: Unable to run simulator: %+v", ds.Device.ID, err)
		return err
	}
	return nil
}

// startSimulationAgent starts the simulated device gRPC server
func (ds *Simulator) startSimulationAgent() error {
	ds.Agent = northbound.NewServer(northbound.NewServerCfg(
		"",
		"",
		"",
		int16(ds.Device.ControlPort),
		true,
		northbound.SecurityConfig{
			AuthenticationEnabled: false,
			AuthorizationEnabled:  false,
		}))
	ds.Agent.AddService(simnb.Service{
		DeviceID: ds.Device.ID,
	})

	doneCh := make(chan error)
	go func() {
		err := ds.Agent.Serve(func(started string) {
			log.Infof("Device %s: Started simulated device NBI on ", ds.Device.ID, started)
			close(doneCh)
		})
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}

// Stop stops the device simulation agent and stops any background simulation tasks
func (ds *Simulator) Stop(mode simapi.StopMode) {
	log.Infof("Device %s: Stopping simulator using %s", ds.Device.ID, mode)
	if mode == simapi.StopMode_ORDERLY_STOP {
		ds.Agent.GracefulStop()
	} else {
		// FIXME: This is not sufficiently chaotic
		ds.Agent.Stop()
	}

	// Stop any background simulation tasks
}

// TODO: Additional simulation logic goes here
