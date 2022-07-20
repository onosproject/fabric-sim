// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fabricsim

import (
	"context"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
)

// GetLinks returns list of all simulated links
func (s *Server) GetLinks(ctx context.Context, request *simapi.GetLinksRequest) (*simapi.GetLinksResponse, error) {
	//TODO implement me
	panic("implement me")
}

// GetLink returns the specified simulated link
func (s *Server) GetLink(ctx context.Context, request *simapi.GetLinkRequest) (*simapi.GetLinkResponse, error) {
	//TODO implement me
	panic("implement me")
}

// AddLink creates and registers the specified simulated link
func (s *Server) AddLink(ctx context.Context, request *simapi.AddLinkRequest) (*simapi.AddLinkRequest, error) {
	//TODO implement me
	panic("implement me")
}

// RemoveLink removes the specified simulated link
func (s *Server) RemoveLink(ctx context.Context, request *simapi.RemoveLinkRequest) (*simapi.RemoveLinkRequest, error) {
	//TODO implement me
	panic("implement me")
}
