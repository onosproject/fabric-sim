// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

// Package gnmi implements the northbound gNMI service for the configuration subsystem.
package gnmi

import (
	gnmisim "github.com/onosproject/fabric-sim/pkg/northbound/device/gnmi/v2"
	p4rtsim "github.com/onosproject/fabric-sim/pkg/northbound/device/p4runtime/v1"
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
	deviceID simapi.DeviceID
}

// NewService allocates a Service struct with the given parameters
func NewService() Service {
	return Service{}
}

// Register registers the gNMI and P4Runtime with the given gRPC server
func (s Service) Register(r *grpc.Server) {
	p4rtapi.RegisterP4RuntimeServer(r, &p4rtsim.Server{})
	gnmiapi.RegisterGNMIServer(r, &gnmisim.Server{})
	log.Debugf("Device %s: P4Runtime and gNMI registered", s.deviceID)
}
