// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fabricsim

import (
	"context"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
)

// GetIOStats returns a list of aggregate I/O time-series statistics accumulated by the simulator.
func (s *Server) GetIOStats(ctx context.Context, request *simapi.GetIOStatsRequest) (*simapi.GetIOStatsResponse, error) {
	return &simapi.GetIOStatsResponse{Stats: s.simulation.Collector.GetIOStats()}, nil
}
