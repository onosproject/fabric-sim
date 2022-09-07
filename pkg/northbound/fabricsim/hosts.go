// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fabricsim

import (
	"context"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/errors"
)

// GetHosts returns list of all simulated hosts
func (s *Server) GetHosts(ctx context.Context, request *simapi.GetHostsRequest) (*simapi.GetHostsResponse, error) {
	sims := s.simulation.GetHostSimulators()
	hosts := make([]*simapi.Host, 0, len(sims))
	for _, sim := range sims {
		hosts = append(hosts, sim.Host)
	}
	return &simapi.GetHostsResponse{Hosts: hosts}, nil
}

// GetHost returns the specified simulated host
func (s *Server) GetHost(ctx context.Context, request *simapi.GetHostRequest) (*simapi.GetHostResponse, error) {
	sim, err := s.simulation.GetHostSimulator(request.ID)
	if err != nil {
		return nil, errors.Status(err).Err()
	}
	return &simapi.GetHostResponse{Host: sim.Host}, nil
}

// AddHost creates and registers the specified simulated host
func (s *Server) AddHost(ctx context.Context, request *simapi.AddHostRequest) (*simapi.AddHostResponse, error) {
	if _, err := s.simulation.AddHostSimulator(request.Host); err != nil {
		return nil, errors.Status(err).Err()
	}
	return &simapi.AddHostResponse{}, nil
}

// RemoveHost removes the specified simulated host
func (s *Server) RemoveHost(ctx context.Context, request *simapi.RemoveHostRequest) (*simapi.RemoveHostResponse, error) {
	if err := s.simulation.RemoveHostSimulator(request.ID); err != nil {
		return nil, errors.Status(err).Err()
	}
	return &simapi.RemoveHostResponse{}, nil
}

// EmitARPs triggers the specified host NIC to send ARP requests for a set of IP addresses
func (s *Server) EmitARPs(ctx context.Context, request *simapi.EmitARPsRequest) (*simapi.EmitARPsResponse, error) {
	if err := s.simulation.EmitARPs(request.ID, request.MacAddress, request.IpAddresses); err != nil {
		return nil, errors.Status(err).Err()
	}
	return &simapi.EmitARPsResponse{}, nil
}
