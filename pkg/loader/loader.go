// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"context"
	"fmt"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"google.golang.org/grpc"
)

var log = logging.GetLogger()

// LoadTopology loads the specified YAML file and creates the prescribed simulated topology entities
// using the fabric simulator API client.
func LoadTopology(conn *grpc.ClientConn, topologyPath string) error {
	topology := &Topology{}
	if err := LoadTopologyFile(topologyPath, topology); err != nil {
		return err
	}

	log.Infof("%+v", topology)

	if err := createDevices(conn, topology.Devices); err != nil {
		return err
	}

	if err := createLinks(conn, topology.Links); err != nil {
		return err
	}

	if err := createHosts(conn, topology.Hosts); err != nil {
		return err
	}
	return nil
}

// Create all simulated devices
func createDevices(conn *grpc.ClientConn, devices []Device) error {
	deviceClient := simapi.NewDeviceServiceClient(conn)
	ctx := context.Background()
	for _, dd := range devices {
		device := constructDevice(dd)
		if _, err := deviceClient.AddDevice(ctx, &simapi.AddDeviceRequest{Device: device}); err != nil {
			log.Errorf("Unable to create simulated device: %+v", err)
		}

		if !dd.Stopped {
			if _, err := deviceClient.StartDevice(ctx, &simapi.StartDeviceRequest{ID: device.ID}); err != nil {
				log.Errorf("Unable to start agent for simulated device: %+v", err)
			}
		}
	}
	return nil
}

func constructDevice(dd Device) *simapi.Device {
	ports := make([]*simapi.Port, 0, len(dd.Ports))
	for _, pd := range dd.Ports {
		port := &simapi.Port{
			ID:             simapi.PortID(fmt.Sprintf("%s/%d", dd.ID, pd.Number)),
			Name:           fmt.Sprintf("%d", pd.Number),
			Number:         pd.Number,
			InternalNumber: pd.SDNNumber,
			Speed:          pd.Speed,
		}
		ports = append(ports, port)
	}
	deviceType := simapi.DeviceType_SWITCH
	if dd.Type == "ipu" {
		deviceType = simapi.DeviceType_IPU
	}
	return &simapi.Device{
		ID:          simapi.DeviceID(dd.ID),
		Type:        deviceType,
		Ports:       ports,
		ControlPort: dd.AgentPort,
	}
}

// Create all simulated links
func createLinks(conn *grpc.ClientConn, links []Link) error {
	linkClient := simapi.NewLinkServiceClient(conn)
	ctx := context.Background()
	for _, ld := range links {
		link := constructLink(ld)
		if _, err := linkClient.AddLink(ctx, &simapi.AddLinkRequest{Link: link}); err != nil {
			log.Errorf("Unable to create simulated link: %+v", err)
		}
		if !ld.Unidirectional {
			link = constructReverseLink(ld)
			if _, err := linkClient.AddLink(ctx, &simapi.AddLinkRequest{Link: link}); err != nil {
				log.Errorf("Unable to create simulated link: %+v", err)
			}
		}
	}
	return nil
}

func constructLink(ld Link) *simapi.Link {
	srcID := simapi.PortID(ld.SrcPortID)
	tgtID := simapi.PortID(ld.TgtPortID)
	return &simapi.Link{
		ID:     simapi.NewLinkID(srcID, tgtID),
		SrcID:  srcID,
		TgtID:  tgtID,
		Status: simapi.LinkStatus_LINK_UP,
	}
}

func constructReverseLink(ld Link) *simapi.Link {
	srcID := simapi.PortID(ld.SrcPortID)
	tgtID := simapi.PortID(ld.TgtPortID)
	return &simapi.Link{
		ID:     simapi.NewLinkID(tgtID, srcID),
		SrcID:  tgtID,
		TgtID:  srcID,
		Status: simapi.LinkStatus_LINK_UP,
	}
}

// Create all simulated hosts
func createHosts(conn *grpc.ClientConn, hosts []Host) error {
	hostClient := simapi.NewHostServiceClient(conn)
	ctx := context.Background()
	for _, hd := range hosts {
		host := constructHost(hd)
		if _, err := hostClient.AddHost(ctx, &simapi.AddHostRequest{Host: host}); err != nil {
			log.Errorf("Unable to create simulated host: %+v", err)
		}
	}
	return nil
}

func constructHost(hd Host) *simapi.Host {
	nics := make([]*simapi.NetworkInterface, 0, len(hd.NICs))
	for _, nd := range hd.NICs {
		nic := &simapi.NetworkInterface{
			ID:          simapi.PortID(nd.Port),
			MacAddress:  nd.Mac,
			IpAddress:   nd.IPv4,
			Ipv6Address: nd.IPV6,
			Behavior:    nil,
		}
		nics = append(nics, nic)
	}
	return &simapi.Host{
		ID:         simapi.HostID(hd.ID),
		Interfaces: nics,
	}
}
