// SPDX-FileCopyrightText: 2020-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package config contains the configuration tree
package config

import (
	"fmt"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/openconfig/gnmi/proto/gnmi"
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
