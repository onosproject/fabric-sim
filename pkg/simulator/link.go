// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
)

// LinkSimulator simulates a single link
type LinkSimulator struct {
	Link *simapi.Link
	// TODO: Add references to the link-specific processors
}

// NewLinkSimulator initializes a new device simulator
func NewLinkSimulator(link *simapi.Link) *LinkSimulator {
	log.Infof("Link %s: Creating simulator", link.ID)
	return &LinkSimulator{Link: link}
}

// TODO: Additional simulation logic goes here
