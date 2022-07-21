// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fabricsim

import (
	"context"
	"github.com/onosproject/fabric-sim/pkg/simulator"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
)

// GetDevices returns a list of simulated devices; switches and IPUs
func (s *Server) GetDevices(ctx context.Context, request *simapi.GetDevicesRequest) (*simapi.GetDevicesResponse, error) {
	sims := s.Simulation.GetDeviceSimulators()
	devices := make([]*simapi.Device, 0, len(sims))
	for _, sim := range sims {
		devices = append(devices, sim.Device)
	}
	return &simapi.GetDevicesResponse{Devices: devices}, nil
}

// GetDevice returns the specified simulated device
func (s *Server) GetDevice(ctx context.Context, request *simapi.GetDeviceRequest) (*simapi.GetDeviceResponse, error) {
	sim, err := s.Simulation.GetDeviceSimulator(request.ID)
	if err != nil {
		return nil, err
	}
	return &simapi.GetDeviceResponse{Device: sim.Device}, nil
}

// AddDevice creates and registers a new simulated device
func (s *Server) AddDevice(ctx context.Context, request *simapi.AddDeviceRequest) (*simapi.AddDeviceResponse, error) {
	if _, err := s.Simulation.AddDeviceSimulator(request.Device); err != nil {
		return nil, err
	}
	return &simapi.AddDeviceResponse{}, nil
}

// RemoveDevice stops and removes the specified simulated device
func (s *Server) RemoveDevice(ctx context.Context, request *simapi.RemoveDeviceRequest) (*simapi.RemoveDeviceResponse, error) {
	if err := s.Simulation.RemoveDeviceSimulator(request.ID); err != nil {
		return nil, err
	}
	return &simapi.RemoveDeviceResponse{}, nil
}

// StartDevice starts the specified simulated device
func (s *Server) StartDevice(ctx context.Context, request *simapi.StartDeviceRequest) (*simapi.StartDeviceResponse, error) {
	sim, err := s.Simulation.GetDeviceSimulator(request.ID)
	if err != nil {
		return nil, err
	}
	if err = sim.Start(); err != nil {
		return nil, err
	}
	return &simapi.StartDeviceResponse{}, nil
}

// StopDevice stops the specified simulated device
func (s *Server) StopDevice(ctx context.Context, request *simapi.StopDeviceRequest) (*simapi.StopDeviceResponse, error) {
	sim, err := s.Simulation.GetDeviceSimulator(request.ID)
	if err != nil {
		return nil, err
	}
	sim.Stop(request.Mode)
	return &simapi.StopDeviceResponse{}, nil
}

// EnablePort enables the specified simulated device port
func (s *Server) EnablePort(ctx context.Context, request *simapi.EnablePortRequest) (*simapi.EnablePortResponse, error) {
	deviceID, err := simulator.ExtractDeviceID(request.ID)
	if err != nil {
		return nil, err
	}
	sim, err := s.Simulation.GetDeviceSimulator(deviceID)
	if err != nil {
		return nil, err
	}
	if err = sim.EnablePort(request.ID); err != nil {
		return nil, err
	}
	return &simapi.EnablePortResponse{}, nil
}

// DisablePort disables the specified simulated device port
func (s *Server) DisablePort(ctx context.Context, request *simapi.DisablePortRequest) (*simapi.DisablePortResponse, error) {
	deviceID, err := simulator.ExtractDeviceID(request.ID)
	if err != nil {
		return nil, err
	}
	sim, err := s.Simulation.GetDeviceSimulator(deviceID)
	if err != nil {
		return nil, err
	}
	if err = sim.DisablePort(request.ID, request.Mode); err != nil {
		return nil, err
	}
	return &simapi.DisablePortResponse{}, nil
}
