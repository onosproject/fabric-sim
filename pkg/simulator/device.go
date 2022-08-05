// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/onosproject/fabric-sim/pkg/simulator/entries"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/genproto/googleapis/rpc/code"
	"strconv"
	"sync"
)

// DeviceSimulator simulates a single device
type DeviceSimulator struct {
	Device *simapi.Device
	Ports  map[simapi.PortID]*simapi.Port
	Agent  DeviceAgent

	lock                     sync.RWMutex
	forwardingPipelineConfig *p4api.ForwardingPipelineConfig
	responders               []StreamResponder
	roleElections            map[string]*p4api.Uint128
	simulation               *Simulation
	sdnPorts                 map[uint32]*simapi.Port

	tables   *entries.Tables
	counters *entries.Counters
	meters   *entries.Meters
}

// NewDeviceSimulator initializes a new device simulator
func NewDeviceSimulator(device *simapi.Device, agent DeviceAgent, simulation *Simulation) *DeviceSimulator {
	log.Infof("Device %s: Creating simulator", device.ID)

	// Build ports and SDN ports maps
	ports := make(map[simapi.PortID]*simapi.Port)
	sdnPorts := make(map[uint32]*simapi.Port)
	for _, port := range device.Ports {
		ports[port.ID] = port
		sdnPorts[port.InternalNumber] = port
	}

	// Construct and return simulator from the device and the port map
	return &DeviceSimulator{
		Device:        device,
		Ports:         ports,
		Agent:         agent,
		roleElections: make(map[string]*p4api.Uint128),
		sdnPorts:      sdnPorts,
		simulation:    simulation,
	}
}

// Start spawns the device simulator background tasks and its agent API server, also in the background
func (ds *DeviceSimulator) Start(simulation *Simulation) error {
	log.Infof("Device %s: Starting simulator", ds.Device.ID)

	// Start any background simulation tasks

	// Starts the simulated device agent
	err := ds.Agent.Start(simulation, ds)
	if err != nil {
		log.Errorf("Device %s: Unable to run simulator: %+v", ds.Device.ID, err)
		return err
	}
	return nil
}

// Stop stops the device simulation agent and stops any background simulation tasks
func (ds *DeviceSimulator) Stop(mode simapi.StopMode) {
	log.Infof("Device %s: Stopping simulator using %s", ds.Device.ID, mode)
	if err := ds.Agent.Stop(mode); err != nil {
		log.Errorf("Device %s: Unable to stop simulator: %+v", ds.Device.ID, err)
	}

	// Stop any background simulation tasks
}

// EnablePort enables the specified simulated device port
func (ds *DeviceSimulator) EnablePort(id simapi.PortID) error {
	log.Infof("Device %s: Enabling port %s", ds.Device.ID, id)
	// TODO: Implement this
	// Look for any links or interfaces using this port and enable them
	return nil
}

// DisablePort disables the specified simulated device port
func (ds *DeviceSimulator) DisablePort(id simapi.PortID, mode simapi.StopMode) error {
	log.Infof("Device %s: Disabling port %s using %s", ds.Device.ID, id, mode)
	// TODO: Implement this
	// Look for any links or interfaces using this port and disable them
	return nil
}

// IsMaster returns an error if the given election ID is not the master for the specified device (chassis) and role.
func (ds *DeviceSimulator) IsMaster(chassisID uint64, role string, electionID *p4api.Uint128) error {
	if chassisID != ds.Device.ChassisID {
		return errors.NewConflict("Incorrect device ID: %d", chassisID)
	}
	winningElectionID, ok := ds.roleElections[role]
	if !ok || winningElectionID.High != electionID.High || winningElectionID.Low != electionID.Low {
		return errors.NewUnauthorized("Not master for role %s on device ID: %d", role, chassisID)
	}
	return nil
}

// RecordRoleElection checks the given election ID for the specified role and records it
// if the given election ID is larger than a previously recorded election ID for the same
// role; returns the winning election ID for the role or nil if a master for this role and
// election ID is already claimed
func (ds *DeviceSimulator) RecordRoleElection(role *p4api.Role, electionID *p4api.Uint128) *p4api.Uint128 {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	roleName := ""
	if role != nil {
		roleName = role.Name
	}

	maxID, ok := ds.roleElections[roleName]
	if !ok || maxID.High < electionID.High || (maxID.High == electionID.High && maxID.Low < electionID.Low) {
		ds.roleElections[roleName] = electionID
		return electionID
	} else if maxID.High == electionID.High && maxID.Low == electionID.Low {
		return nil // this role and election ID has already been claimed
	}
	return maxID
}

// RunMastershipArbitration processes the specified arbitration update
func (ds *DeviceSimulator) RunMastershipArbitration(role *p4api.Role, electionID *p4api.Uint128) error {
	log.Debugf("Device %s: running mastership arbitration for role %s and electionID %+v", ds.Device.ID, role, electionID)

	// Record the role and election ID, return the winning (highest) election ID for the role
	maxElectionID := ds.RecordRoleElection(role, electionID)

	ds.lock.RLock()
	defer ds.lock.RUnlock()

	// TODO: generate failed precondition (conflict) error or not found error if device ID does not match
	// TODO: handle voluntary mastership downgrade
	// TODO: handle role.config promotion

	// If we cannot locate the responder with the max election ID, then this means the previous
	// master has left and we need to return NOT_FOUND code to all existing responders for this role
	failCode := code.Code_NOT_FOUND
	if maxElectionID == nil {
		failCode = code.Code_INVALID_ARGUMENT
	} else {
		for _, r := range ds.responders {
			if r.IsMaster(role, maxElectionID) {
				failCode = code.Code_ALREADY_EXISTS
				break
			}
		}
	}

	// Notify all responders for the role
	for _, r := range ds.responders {
		r.SendMastershipArbitration(role, maxElectionID, failCode)
	}

	return nil
}

// AddStreamResponder adds the given stream responder to the specified device
func (ds *DeviceSimulator) AddStreamResponder(responder StreamResponder) {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	ds.responders = append(ds.responders, responder)
}

// RemoveStreamResponder removes the specified stream responder to the specified device
func (ds *DeviceSimulator) RemoveStreamResponder(responder StreamResponder) {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	i := 0
	for _, r := range ds.responders {
		if r == responder {
			ds.responders[i] = r
			i++
		}
	}
	ds.responders = ds.responders[:i]
}

// SendToAllResponders sends the specified message to all responders
func (ds *DeviceSimulator) SendToAllResponders(response *p4api.StreamMessageResponse) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()
	for _, r := range ds.responders {
		r.Send(response)
	}
}

// SetPipelineConfig sets the forwarding pipeline configuration for the device
func (ds *DeviceSimulator) SetPipelineConfig(fpc *p4api.ForwardingPipelineConfig) error {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	ds.forwardingPipelineConfig = fpc

	// Create the required entities, e.g. tables, counters, meters, etc.
	info := fpc.P4Info
	ds.tables = entries.NewTables(info.Tables)
	ds.counters = entries.NewCounters(info.Counters)
	ds.meters = entries.NewMeters(info.Meters)
	return nil
}

// GetPipelineConfig sets the forwarding pipeline configuration for the device
func (ds *DeviceSimulator) GetPipelineConfig() *p4api.ForwardingPipelineConfig {
	return ds.forwardingPipelineConfig
}

// ProcessPacketOut handles the specified packet out message
func (ds *DeviceSimulator) ProcessPacketOut(packetOut *p4api.PacketOut, responder StreamResponder) error {
	log.Infof("Device %s: received packet out: %+v", ds.Device.ID, packetOut)

	// Start by decoding the packet
	packet := gopacket.NewPacket(packetOut.Payload, layers.LayerTypeLinkLayerDiscovery, gopacket.Default)

	// See if this is an LLDP packet and process it if so
	if lldpLayer := packet.Layer(layers.LayerTypeLinkLayerDiscovery); lldpLayer != nil {
		ds.processLLDPPacket(lldpLayer.(*layers.LinkLayerDiscovery), packetOut)
	}

	// Process ARP packet
	// Process DHCP packet
	// ...
	return nil
}

// ProcessDigestAck handles the specified digest list ack message
func (ds *DeviceSimulator) ProcessDigestAck(ack *p4api.DigestListAck, responder StreamResponder) error {
	log.Infof("Device %s: received digest ack: %+v", ds.Device.ID, ack)
	// TODO: Implement this
	return nil
}

// Processes the LLDP packet-out by emitting it encapsulated as a packet-in on the simulated device which is
// adjacent to this device on the link (if any) connected to the port given in the LLDP packet
func (ds *DeviceSimulator) processLLDPPacket(lldp *layers.LinkLayerDiscovery, packetOut *p4api.PacketOut) {
	log.Debugf("Device %s: processing LLDP packet: %+v", ds.Device.ID, lldp)

	// TODO: Add filtering based on device table contents
	portID := portNumberFromLLDP(lldp.PortID)

	// Find the port corresponding to the specified port ID, which is the internal (SDN) port number
	port, ok := ds.sdnPorts[portID]
	if !ok {
		log.Warnf("Device %s: Port %d not found", ds.Device.ID, portID)
		return
	}

	// Check if the given port has a link originating from it
	if link := ds.simulation.GetLinkFromPort(port.ID); link != nil {
		// Now that we found the link, let's emit a packet out on all the responders associated with
		// the destination device
		tgtDeviceID, err := ExtractDeviceID(link.TgtID)
		if err != nil {
			log.Warnf("Device %s: %s", ds.Device.ID, err)
			return
		}

		tgtDevice, ok := ds.simulation.deviceSimulators[tgtDeviceID]
		if !ok {
			log.Warnf("Device %s: Unable to locate link target device %s", ds.Device.ID, tgtDeviceID)
		}

		packetIn := &p4api.StreamMessageResponse{
			Update: &p4api.StreamMessageResponse_Packet{
				Packet: &p4api.PacketIn{
					Payload: packetOut.Payload,
				},
			},
		}
		tgtDevice.SendToAllResponders(packetIn)
	}
}

// Decodes the specified LLDP port ID into an internal SDN port number
func portNumberFromLLDP(id layers.LLDPPortID) uint32 {
	if i, err := strconv.ParseUint(string(id.ID), 10, 32); err == nil {
		return uint32(i)
	}
	return 0
}

// ProcessWrite processes the specified batch of updates
func (ds *DeviceSimulator) ProcessWrite(atomicity p4api.WriteRequest_Atomicity, updates []*p4api.Update) error {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	for _, update := range updates {
		switch {
		case update.Type == p4api.Update_INSERT:
			if err := ds.processModify(update, true); err != nil {
				return err
			}
		case update.Type == p4api.Update_MODIFY:
			if err := ds.processModify(update, false); err != nil {
				return err
			}
		case update.Type == p4api.Update_DELETE:
			if err := ds.processDelete(update); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ds *DeviceSimulator) processModify(update *p4api.Update, isInsert bool) error {
	entity := update.Entity
	var err error
	switch {
	case entity.GetTableEntry() != nil:
		err = ds.tables.ModifyTableEntry(entity.GetTableEntry(), isInsert)
	case entity.GetCounterEntry() != nil:
		err = ds.counters.ModifyCounterEntry(entity.GetCounterEntry(), isInsert)
	case entity.GetDirectCounterEntry() != nil:
		err = ds.tables.ModifyDirectCounterEntry(entity.GetDirectCounterEntry(), isInsert)
	case entity.GetMeterEntry() != nil:
		err = ds.meters.ModifyMeterEntry(entity.GetMeterEntry(), isInsert)
	case entity.GetDirectMeterEntry() != nil:
		err = ds.tables.ModifyDirectMeterEntry(entity.GetDirectMeterEntry(), isInsert)

	case entity.GetRegisterEntry() != nil:
	case entity.GetValueSetEntry() != nil:
	case entity.GetActionProfileGroup() != nil:
	case entity.GetActionProfileMember() != nil:
	case entity.GetDigestEntry() != nil:
	case entity.GetPacketReplicationEngineEntry() != nil:
	case entity.GetExternEntry() != nil:
	default:
	}
	return err
}

func (ds *DeviceSimulator) processDelete(update *p4api.Update) error {
	entity := update.Entity
	var err error
	switch {
	case entity.GetTableEntry() != nil:
		err = ds.tables.RemoveTableEntry(entity.GetTableEntry())
	case entity.GetCounterEntry() != nil:
		return errors.NewInvalid("Counter cannot be deleted")
	case entity.GetDirectCounterEntry() != nil:
		err = errors.NewInvalid("Direct counter entry cannot be deleted")
	case entity.GetMeterEntry() != nil:
		return errors.NewInvalid("Meter cannot be deleted")
	case entity.GetDirectMeterEntry() != nil:
		err = errors.NewInvalid("Direct meter entry cannot be deleted")

	case entity.GetRegisterEntry() != nil:
	case entity.GetValueSetEntry() != nil:
	case entity.GetActionProfileGroup() != nil:
	case entity.GetActionProfileMember() != nil:
	case entity.GetDigestEntry() != nil:
	case entity.GetPacketReplicationEngineEntry() != nil:
	case entity.GetExternEntry() != nil:
	default:
	}
	return err
}

// ProcessRead executes the read of the specified set of requests, returning accumulated results via the supplied sender
func (ds *DeviceSimulator) ProcessRead(requests []*p4api.Entity, sender entries.BatchSender) []error {
	ds.lock.RLock()
	defer ds.lock.RUnlock()

	// Allocate the same number of errors as there are requests - expressed as entities
	errors := make([]error, len(requests))

	for i, request := range requests {
		errors[i] = ds.processRead(request, sender)
	}
	return errors
}

// Executes the read of the specified request, returning accumulated results via the supplied sender
func (ds *DeviceSimulator) processRead(request *p4api.Entity, sender entries.BatchSender) error {
	switch {
	case request.GetTableEntry() != nil:
		return ds.tables.ReadTableEntries(request.GetTableEntry(), entries.ReadTableEntry, sender)
	case request.GetCounterEntry() != nil:
	case request.GetDirectCounterEntry() != nil:
		return ds.tables.ReadTableEntries(request.GetTableEntry(), entries.ReadDirectCounter, sender)
	case request.GetMeterEntry() != nil:
	case request.GetDirectMeterEntry() != nil:
		return ds.tables.ReadTableEntries(request.GetTableEntry(), entries.ReadDirectMeter, sender)

	case request.GetRegisterEntry() != nil:
	case request.GetValueSetEntry() != nil:
	case request.GetActionProfileGroup() != nil:
	case request.GetActionProfileMember() != nil:
	case request.GetDigestEntry() != nil:
	case request.GetPacketReplicationEngineEntry() != nil:
	case request.GetExternEntry() != nil:
	default:
	}
	return nil
}

// TODO: Additional simulation logic goes here
