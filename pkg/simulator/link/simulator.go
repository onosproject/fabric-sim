// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package link implements the link simulator control logic
package link

import (
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("simulator", "link")

// Simulator simulates a single link
type Simulator struct {
	Link *simapi.Link
	// TODO: Add references to the link-specific processors
}

// NewLinkSimulator initializes a new device simulator
func NewLinkSimulator(link *simapi.Link) *Simulator {
	log.Infof("Link %s: Creating simulator", link.ID)
	sim := Simulator{
		Link: link,
	}
	return &sim
}

// TODO: Additional simulation logic goes here
