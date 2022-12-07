// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package config contains the configuration tree
package config

import (
	"context"
	"fmt"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-net-lib/pkg/gnmiutils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"math/rand"
	"time"
)

// NewSwitchConfig creates a new switch skeleton configuration and returns its root node
func NewSwitchConfig(ports map[simapi.PortID]*simapi.Port) *Node {
	rootNode := NewRoot()

	interfacesNode := rootNode.Add("interfaces", nil, nil)
	index := uint64(1)
	for _, port := range ports {
		name := port.Name
		if len(name) == 0 {
			name = fmt.Sprintf("%d", port.Number)
		}
		interfaceNode := interfacesNode.Add("interface", map[string]string{"name": name}, nil)

		interfaceNode.AddPath("state/ifindex",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_UintVal{UintVal: uint64(port.Number)}})
		interfaceNode.AddPath("state/id",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_UintVal{UintVal: uint64(port.InternalNumber)}})

		portStatus := "UP"
		if !port.Enabled {
			portStatus = "DOWN"
		}
		interfaceNode.AddPath("state/oper-status",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: portStatus}})

		interfaceNode.AddPath("state/last-change",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_UintVal{UintVal: 0}})

		interfaceNode.AddPath("config/enabled",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_BoolVal{BoolVal: port.Enabled}})
		interfaceNode.AddPath("ethernet/config/port-speed",
			&gnmi.TypedValue{Value: &gnmi.TypedValue_StringVal{StringVal: port.Speed}})

		index = index + 1

		addCounters(interfaceNode)
	}

	return rootNode
}

var supportedCounters = []string{
	"in-octets",
	"out-octets",
	"in-discards",
	"in-fcs-errors",
	"out-discards",
	"in-errors",
	"out-errors",
	"in-unicast-pkts",
	"in-broadcast-pkts",
	"in-multicast-pkts",
	"in-unknown-protos",
	"out-unicast-pkts",
	"out-broadcast-pkts",
	"out-multicast-pkts",
}

func addCounters(node *Node) {
	countersNode := node.AddPath("state/counters", nil)
	for _, counter := range supportedCounters {
		countersNode.Add(counter, nil, &gnmi.TypedValue{Value: &gnmi.TypedValue_IntVal{IntVal: 0}})
	}
}

// Auxiliary structure to aid in simulating increasing traffic counters
type portData struct {
	lastUpdate time.Time
	bytesIn    *Node
	bytesOut   *Node
	packetsIn  *Node
	packetsOut *Node
}

func (d *portData) simulateTrafficCounter() {
	delta := time.Since(d.lastUpdate).Milliseconds()

	// Simulate increases in packets-in and bytes-in in some reasonable proportion to each other
	pin := packetsAmount(delta, d.packetsIn.Value().GetUintVal())
	d.packetsIn.Value().Value = &gnmi.TypedValue_UintVal{UintVal: pin}
	bin := bytesAmount(delta, d.bytesIn.Value().GetUintVal(), pin)
	d.bytesIn.Value().Value = &gnmi.TypedValue_UintVal{UintVal: bin}

	// Simulate increases in packets-out and bytes-out in some reasonable proportion to each other
	pout := packetsAmount(delta, d.packetsOut.Value().GetUintVal())
	d.packetsOut.Value().Value = &gnmi.TypedValue_UintVal{UintVal: pout}
	bout := bytesAmount(delta, d.bytesOut.Value().GetUintVal(), pout)
	d.bytesOut.Value().Value = &gnmi.TypedValue_UintVal{UintVal: bout}
}

const (
	packetsPerMSMin   = 1
	packetsPerMSMax   = 20
	bytesPerPacketMin = 60
	bytesPerPacketMax = 1500
)

// Generates a random increase in packets based on time delta in milliseconds
func packetsAmount(millis int64, value uint64) uint64 {
	return value + rand.Uint64()%uint64(millis)*(packetsPerMSMin+rand.Uint64()%(packetsPerMSMax-packetsPerMSMin))
}

// Generates a random increase in bytes based on time delta in milliseconds and a number of packets in the same period
func bytesAmount(millis int64, value uint64, packets uint64) uint64 {
	if packets > 0 {
		return value + rand.Uint64()%(packets*(bytesPerPacketMin+rand.Uint64()%(bytesPerPacketMax-bytesPerPacketMin)))
	}
	return value
}

// SimulateTrafficCounters simulates a select set of traffic-related counters for all ports under the given
// root configuration
func SimulateTrafficCounters(ctx context.Context, delay time.Duration, node *Node) {
	portCounters := findCountersToSimulate(node)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
				simulateTrafficCounters(portCounters)
			}
		}
	}()
}

func simulateTrafficCounters(counters map[string]*portData) {
	for _, data := range counters {
		data.simulateTrafficCounter()
	}
}

func findCountersToSimulate(node *Node) map[string]*portData {
	portCounters := make(map[string]*portData)
	for _, n := range node.FindAll("interfaces/interface[name=...]/state/counters") {
		if isSimulated(n.Name()) {
			if portData := getPortData(portCounters, n); portData != nil {
				switch n.Name() {
				case "in-octets":
					portData.bytesIn = n
				case "out-octets":
					portData.bytesOut = n
				case "in-unicast-pkts":
					portData.packetsIn = n
				case "out-unicast-pkts":
					portData.packetsOut = n
				}
			}
		}
	}
	return portCounters
}

func getPortData(counters map[string]*portData, node *Node) *portData {
	segments := gnmiutils.SplitPath(node.Path())
	if len(segments) > 4 {
		_, key, _ := gnmiutils.NameKey(segments[1])
		if portID, ok := key["name"]; ok {
			data, ok := counters[portID]
			if !ok {
				data = &portData{lastUpdate: time.Now()}
				counters[portID] = data
			}
			return data
		}
	}
	return nil
}

func isSimulated(name string) bool {
	return name == "in-octets" || name == "out-octets" || name == "in-unicast-pkts" || name == "out-unicast-pkts"
}
