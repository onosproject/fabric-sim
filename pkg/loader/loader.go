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

func createDevices(conn *grpc.ClientConn, devices []Device) error {
	deviceClient := simapi.NewDeviceServiceClient(conn)
	ctx := context.Background()
	for _, d := range devices {
		device := constructDevice(d)
		if _, err := deviceClient.AddDevice(ctx, &simapi.AddDeviceRequest{Device: device}); err != nil {
			log.Errorf("Unable to create simulated device: %+v", err)
		}

		if !d.Stopped {
			if _, err := deviceClient.StartDevice(ctx, &simapi.StartDeviceRequest{ID: device.ID}); err != nil {
				log.Errorf("Unable to start agent for simulated device: %+v", err)
			}
		}
	}
	return nil
}

func constructDevice(d Device) *simapi.Device {
	ports := make([]*simapi.Port, 0, len(d.Ports))
	for _, p := range d.Ports {
		port := &simapi.Port{
			ID:             simapi.PortID(fmt.Sprintf("%s/%d", d.ID, p.Number)),
			Name:           fmt.Sprintf("%d", p.Number),
			Number:         p.Number,
			InternalNumber: p.SDNNumber,
			Speed:          p.Speed,
		}
		ports = append(ports, port)
	}
	deviceType := simapi.DeviceType_SWITCH
	if d.Type == "ipu" {
		deviceType = simapi.DeviceType_IPU
	}
	return &simapi.Device{
		ID:          simapi.DeviceID(d.ID),
		Type:        deviceType,
		Ports:       ports,
		ControlPort: d.AgentPort,
	}
}

func createLinks(conn *grpc.ClientConn, devices []Link) error {
	//linkClient := simapi.NewLinkServiceClient(conn)
	return nil
}

func createHosts(conn *grpc.ClientConn, devices []Host) error {
	//hostClient := simapi.NewHostServiceClient(conn)
	return nil
}
