// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
)

// HostSimulator simulates a single host
type HostSimulator struct {
	Host *simapi.Host
	// TODO: Add references to the host-specific processors
}

// NewHostSimulator initializes a new device simulator
func NewHostSimulator(host *simapi.Host) *HostSimulator {
	log.Infof("Host %s: Creating simulator", host.ID)
	return &HostSimulator{Host: host}
}

// TODO: Additional simulation logic goes here
