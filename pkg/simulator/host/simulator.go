// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package host implements the Host simulator control logic
package host

import (
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("simulator", "host")

// Simulator simulates a single host
type Simulator struct {
	Host *simapi.Host
	// TODO: Add references to the host-specific processors
}

// NewHostSimulator initializes a new device simulator
func NewHostSimulator(host *simapi.Host) *Simulator {
	log.Infof("Host %s: Creating simulator", host.ID)
	sim := Simulator{
		Host: host,
	}
	return &sim
}

// TODO: Additional simulation logic goes here
