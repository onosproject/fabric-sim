// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fabricsim

import (
	"context"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
)

// GetHosts returns list of all simulated hosts
func (s *Server) GetHosts(ctx context.Context, request *simapi.GetHostsRequest) (*simapi.GetHostsResponse, error) {
	//TODO implement me
	panic("implement me")
}

// GetHost returns the specified simulated host
func (s *Server) GetHost(ctx context.Context, request *simapi.GetHostRequest) (*simapi.GetHostResponse, error) {
	//TODO implement me
	panic("implement me")
}

// AddHost creates and registers the specified simulated host
func (s *Server) AddHost(ctx context.Context, request *simapi.AddHostRequest) (*simapi.AddHostResponse, error) {
	//TODO implement me
	panic("implement me")
}

// RemoveHost removes the specified simulated host
func (s *Server) RemoveHost(ctx context.Context, request *simapi.RemoveHostRequest) (*simapi.RemoveHostResponse, error) {
	//TODO implement me
	panic("implement me")
}
