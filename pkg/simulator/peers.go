// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Representing a connection to a peer fabric-sim instance
type peerSimulator struct {
	domain string
	conn   *grpc.ClientConn
	client simapi.DeviceServiceClient
}

// Gets an existing peer simulator client context for the given domain or creates a new one
func (s *Simulation) getPeer(domain string) (*peerSimulator, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	peer, ok := s.peers[domain]
	if !ok {
		peer = &peerSimulator{domain: domain}
		s.peers[domain] = peer
	}

	if peer.client == nil {
		opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		conn, err := grpc.Dial(domain, opts...)
		if err != nil {
			return nil, err
		}
		peer.conn = conn
		peer.client = simapi.NewDeviceServiceClient(conn)
	}
	return peer, nil
}
