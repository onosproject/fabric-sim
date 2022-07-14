// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fabricsim

import (
	"context"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
)

func (s *Server) GetDevices(ctx context.Context, request *simapi.GetDevicesRequest) (*simapi.GetDevicesResponse, error) {
	sims := s.Simulation.GetDeviceSimulators()
	devices := make([]*simapi.Device, 0, len(sims))
	for _, sim := range sims {
		devices = append(devices, sim.Device)
	}
	return &simapi.GetDevicesResponse{Devices: devices}, nil
}

func (s *Server) GetDevice(ctx context.Context, request *simapi.GetDeviceRequest) (*simapi.GetDeviceResponse, error) {
	sim, err := s.Simulation.GetDeviceSimulator(request.ID)
	if err != nil {
		return nil, err
	}
	return &simapi.GetDeviceResponse{Device: sim.Device}, nil
}

func (s *Server) AddDevice(ctx context.Context, request *simapi.AddDeviceRequest) (*simapi.AddDeviceResponse, error) {
	if _, err := s.Simulation.AddDeviceSimulator(request.Device); err != nil {
		return nil, err
	}
	return &simapi.AddDeviceResponse{}, nil
}

func (s *Server) RemoveDevice(ctx context.Context, request *simapi.RemoveDeviceRequest) (*simapi.RemoveDeviceResponse, error) {
	if err := s.Simulation.RemoveDeviceSimulator(request.ID); err != nil {
		return nil, err
	}
	return &simapi.RemoveDeviceResponse{}, nil
}

func (s *Server) StopDevice(ctx context.Context, request *simapi.StopDeviceRequest) (*simapi.StopDeviceResponse, error) {
	sim, err := s.Simulation.GetDeviceSimulator(request.ID)
	if err != nil {
		return nil, err
	}
	sim.Stop(request.Mode)
	return &simapi.StopDeviceResponse{}, nil
}

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

func (s *Server) DisablePort(ctx context.Context, request *simapi.DisablePortRequest) (*simapi.DisablePortResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s *Server) EnablePort(ctx context.Context, request *simapi.EnablePortRequest) (*simapi.EnablePortResponse, error) {
	//TODO implement me
	panic("implement me")
}
