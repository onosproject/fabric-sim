// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package fabricsim implements the northbound API of the fabric simulator
package fabricsim

import (
	"github.com/onosproject/fabric-sim/pkg/simulator"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/onosproject/onos-lib-go/pkg/northbound"
	"google.golang.org/grpc"
)

var log = logging.GetLogger("northbound", "fabricsim")

// Service implements the fabric simulator NB gRPC
type Service struct {
	northbound.Service
	Simulation *simulator.Simulation
}

// NewService allocates a Service struct with the given parameters
func NewService() Service {
	return Service{}
}

// Register registers the server with grpc
func (s Service) Register(r *grpc.Server) {
	server := &Server{
		Simulation: s.Simulation,
	}
	simapi.RegisterDeviceServiceServer(r, server)
	simapi.RegisterLinkServiceServer(r, server)
	simapi.RegisterHostServiceServer(r, server)
	log.Debug("Fabric API services registered")
}

// Server implements the grpc fabric simulator service
type Server struct {
	Simulation *simulator.Simulation
}
