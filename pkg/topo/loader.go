// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package topo

import (
	"context"
	"fmt"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"google.golang.org/grpc"
)

// LoadTopology loads the specified YAML file and creates the prescribed simulated topology entities
// using the fabric simulator API client.
func LoadTopology(conn *grpc.ClientConn, topologyPath string) error {
	log.Infof("Loading topology from %s", topologyPath)
	topology := &Topology{}

	if err := LoadTopologyFile(topologyPath, topology); err != nil {
		return err
	}

	log.Debugf("Devices: %d; links: %d; hosts: %d",
		len(topology.Devices), len(topology.Links), len(topology.Hosts))

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

// LoadTopologyFile loads the specified topology YAML file
func LoadTopologyFile(path string, topology *Topology) error {
	cfg, err := readConfig(path)
	if err != nil {
		return err
	}
	return cfg.Unmarshal(topology)
}

// Create all simulated Devices
func createDevices(conn *grpc.ClientConn, devices []Device) error {
	deviceClient := simapi.NewDeviceServiceClient(conn)
	ctx := context.Background()
	for _, dd := range devices {
		device := ConstructDevice(dd)
		if _, err := deviceClient.AddDevice(ctx, &simapi.AddDeviceRequest{Device: device}); err != nil {
			log.Errorf("Unable to create simulated device: %+v", err)
			return err
		}

		if !dd.Stopped {
			if _, err := deviceClient.StartDevice(ctx, &simapi.StartDeviceRequest{ID: device.ID}); err != nil {
				log.Errorf("Unable to start agent for simulated device: %+v", err)
				return err
			}
		}
	}
	return nil
}

// ConstructDevice creates a device from the specified device YAML descriptor
func ConstructDevice(dd Device) *simapi.Device {
	ports := make([]*simapi.Port, 0, len(dd.Ports))
	for _, pd := range dd.Ports {
		internalNumber := pd.Number // default internal (SDN) port number to the external number
		if pd.SDNNumber != 0 {
			internalNumber = pd.SDNNumber
		}
		port := &simapi.Port{
			ID:             simapi.PortID(fmt.Sprintf("%s/%d", dd.ID, pd.Number)),
			Name:           fmt.Sprintf("%d", pd.Number),
			Number:         pd.Number,
			InternalNumber: internalNumber,
			Speed:          pd.Speed,
			Enabled:        true,
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
		link := ConstructLink(ld)
		if _, err := linkClient.AddLink(ctx, &simapi.AddLinkRequest{Link: link}); err != nil {
			log.Errorf("Unable to create simulated link: %+v", err)
			return err
		}
		if !ld.Unidirectional {
			reverselink := constructReverseLink(ld)
			if _, err := linkClient.AddLink(ctx, &simapi.AddLinkRequest{Link: reverselink}); err != nil {
				log.Errorf("Unable to create simulated link: %+v", err)
				return err
			}
		}
	}
	return nil
}

// ConstructLink creates a link from the specified link YAML descriptor
func ConstructLink(ld Link) *simapi.Link {
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
		host := ConstructHost(hd)
		if _, err := hostClient.AddHost(ctx, &simapi.AddHostRequest{Host: host}); err != nil {
			log.Errorf("Unable to create simulated host: %+v", err)
			return err
		}
	}
	return nil
}

// ConstructHost creates a host from the specified host YAML descriptor
func ConstructHost(hd Host) *simapi.Host {
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

// ClearTopology removes all Devices, links and hosts from the simulator
func ClearTopology(conn *grpc.ClientConn) error {
	log.Info("Clearing entire topology")
	if err := removeAllHosts(conn); err != nil {
		return err
	}
	if err := removeAllLinks(conn); err != nil {
		return err
	}
	if err := removeAllDevices(conn); err != nil {
		return err
	}
	return nil
}

func removeAllHosts(conn *grpc.ClientConn) error {
	hostClient := simapi.NewHostServiceClient(conn)
	ctx := context.Background()
	resp, err := hostClient.GetHosts(ctx, &simapi.GetHostsRequest{})
	if err != nil {
		return err
	}

	for _, host := range resp.Hosts {
		if _, err = hostClient.RemoveHost(ctx, &simapi.RemoveHostRequest{ID: host.ID}); err != nil {
			return err
		}
	}
	return nil
}

func removeAllLinks(conn *grpc.ClientConn) error {
	linkClient := simapi.NewLinkServiceClient(conn)
	ctx := context.Background()
	resp, err := linkClient.GetLinks(ctx, &simapi.GetLinksRequest{})
	if err != nil {
		return err
	}

	for _, link := range resp.Links {
		if _, err = linkClient.RemoveLink(ctx, &simapi.RemoveLinkRequest{ID: link.ID}); err != nil {
			return err
		}
	}
	return nil
}

func removeAllDevices(conn *grpc.ClientConn) error {
	deviceClient := simapi.NewDeviceServiceClient(conn)
	ctx := context.Background()
	resp, err := deviceClient.GetDevices(ctx, &simapi.GetDevicesRequest{})
	if err != nil {
		return err
	}

	for _, device := range resp.Devices {
		if _, err = deviceClient.RemoveDevice(ctx, &simapi.RemoveDeviceRequest{ID: device.ID}); err != nil {
			return err
		}
	}
	return nil
}
