// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package gnmi implements the northbound gNMI service for the configuration subsystem.
package gnmi

import (
	"context"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
)

var log = logging.GetLogger("northbound", "device", "gnmi")

// Server implements the P4Runtime API
type Server struct {
	deviceID simapi.DeviceID
}

func (s *Server) Capabilities(ctx context.Context, request *gnmi.CapabilityRequest) (*gnmi.CapabilityResponse, error) {
	log.Infof("Device %s: gNMI capabilities have been requested", s.deviceID)
	panic("implement me")
}

func (s *Server) Get(ctx context.Context, request *gnmi.GetRequest) (*gnmi.GetResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s *Server) Set(ctx context.Context, request *gnmi.SetRequest) (*gnmi.SetResponse, error) {
	log.Infof("Device %s: gNMI configuration has been set", s.deviceID)
	panic("implement me")
}

func (s *Server) Subscribe(server gnmi.GNMI_SubscribeServer) error {
	//TODO implement me
	panic("implement me")
}
