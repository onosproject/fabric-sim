// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fabricsim

import (
	"context"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/errors"
)

// GetLinks returns list of all simulated links
func (s *Server) GetLinks(ctx context.Context, request *simapi.GetLinksRequest) (*simapi.GetLinksResponse, error) {
	sims := s.simulation.GetLinkSimulators()
	links := make([]*simapi.Link, 0, len(sims))
	for _, sim := range sims {
		links = append(links, sim.Link)
	}
	return &simapi.GetLinksResponse{Links: links}, nil
}

// GetLink returns the specified simulated link
func (s *Server) GetLink(ctx context.Context, request *simapi.GetLinkRequest) (*simapi.GetLinkResponse, error) {
	sim, err := s.simulation.GetLinkSimulator(request.ID)
	if err != nil {
		return nil, errors.Status(err).Err()
	}
	return &simapi.GetLinkResponse{Link: sim.Link}, nil
}

// AddLink creates and registers the specified simulated link
func (s *Server) AddLink(ctx context.Context, request *simapi.AddLinkRequest) (*simapi.AddLinkResponse, error) {
	log.Infof("Received add link request: %+v", request)
	if _, err := s.simulation.AddLinkSimulator(request.Link); err != nil {
		return nil, errors.Status(err).Err()
	}
	log.Infof("Sending add link response")
	return &simapi.AddLinkResponse{}, nil
}

// RemoveLink removes the specified simulated link
func (s *Server) RemoveLink(ctx context.Context, request *simapi.RemoveLinkRequest) (*simapi.RemoveLinkResponse, error) {
	if err := s.simulation.RemoveLinkSimulator(request.ID); err != nil {
		return nil, errors.Status(err).Err()
	}
	return &simapi.RemoveLinkResponse{}, nil
}
