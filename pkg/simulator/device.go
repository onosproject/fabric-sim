// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"context"
	"encoding/binary"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/onosproject/fabric-sim/pkg/simulator/config"
	"github.com/onosproject/fabric-sim/pkg/simulator/entries"
	"github.com/onosproject/fabric-sim/pkg/utils"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/openconfig/gnmi/proto/gnmi"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/genproto/googleapis/rpc/code"
	"strings"
	"sync"
	"time"
)

// DeviceSimulator simulates a single device
type DeviceSimulator struct {
	Device *simapi.Device
	Ports  map[simapi.PortID]*simapi.Port
	Agent  DeviceAgent

	lock                     sync.RWMutex
	forwardingPipelineConfig *p4api.ForwardingPipelineConfig
	streamResponders         []StreamResponder
	subscribeResponders      []SubscribeResponder
	roleElections            map[string]*p4api.Uint128
	simulation               *Simulation
	sdnPorts                 map[uint32]*simapi.Port

	tables   *entries.Tables
	counters *entries.Counters
	meters   *entries.Meters

	config     *config.Node
	codec      *utils.ControllerMetadataCodec
	puntToCPU  map[layers.EthernetType]bool
	cpuActions map[uint32]*p4info.Action
	cpuTables  map[uint32]*cpuTable

	cancel context.CancelFunc
}

// Auxiliary structure to track table that has CPU related actions and the field match related to ETH type
type cpuTable struct {
	table          *entries.Table
	ethTypeFieldID uint32
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

	cfg := config.NewSwitchConfig(ports)

	device.PipelineInfo = &simapi.PipelineInfo{}

	// Construct and return simulator from the device and the port map
	return &DeviceSimulator{
		Device: device,
		Ports:  ports,
		Agent:  agent,
		forwardingPipelineConfig: &p4api.ForwardingPipelineConfig{
			P4Info:         &p4info.P4Info{},
			P4DeviceConfig: []byte{},
			Cookie:         &p4api.ForwardingPipelineConfig_Cookie{Cookie: 0},
		},
		roleElections: make(map[string]*p4api.Uint128),
		sdnPorts:      sdnPorts,
		simulation:    simulation,
		config:        cfg,
		puntToCPU:     make(map[layers.EthernetType]bool),
		cpuActions:    make(map[uint32]*p4info.Action),
		cpuTables:     make(map[uint32]*cpuTable),
	}
}

// Tables returns the device tables store
func (ds *DeviceSimulator) Tables() *entries.Tables {
	return ds.tables
}

// Counters returns the device counters store
func (ds *DeviceSimulator) Counters() *entries.Counters {
	return ds.counters
}

// Meters returns the device meters store
func (ds *DeviceSimulator) Meters() *entries.Meters {
	return ds.meters
}

// SnapshotStats snapshots any dynamic device stats, e.g. pipeline info
func (ds *DeviceSimulator) SnapshotStats() *DeviceSimulator {
	ds.snapshotTables()
	return ds
}

// Start spawns the device simulator background tasks and its agent API server, also in the background
func (ds *DeviceSimulator) Start(simulation *Simulation) error {
	log.Infof("Device %s: Starting simulator", ds.Device.ID)

	// Start any background simulation tasks
	ctx, cancel := context.WithCancel(context.Background())
	ds.cancel = cancel
	config.SimulateTrafficCounters(ctx, 4*time.Second, ds.config)

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
	// Stop any background simulation tasks
	if ds.cancel != nil {
		ds.cancel()
	}

	log.Infof("Device %s: Stopping simulator using %s", ds.Device.ID, mode)
	if err := ds.Agent.Stop(mode); err != nil {
		log.Errorf("Device %s: Unable to stop simulator: %+v", ds.Device.ID, err)
	}
}

// EnablePort enables the specified simulated device port
func (ds *DeviceSimulator) EnablePort(id simapi.PortID) error {
	log.Infof("Device %s: Enabling port %s", ds.Device.ID, id)
	return ds.setPortStatus(id, simapi.LinkStatus_LINK_UP)
}

// DisablePort disables the specified simulated device port
func (ds *DeviceSimulator) DisablePort(id simapi.PortID, mode simapi.StopMode) error {
	log.Infof("Device %s: Disabling port %s using %s", ds.Device.ID, id, mode)
	return ds.setPortStatus(id, simapi.LinkStatus_LINK_DOWN)
}

func (ds *DeviceSimulator) setPortStatus(id simapi.PortID, linkStatus simapi.LinkStatus) error {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	port, ok := ds.Ports[id]
	if !ok {
		log.Warnf("Device %s: Port %s not found", ds.Device.ID, id)
		return errors.NewNotFound("port %s not found", id)
	}

	// Look for any links or interfaces using this port and disable them
	switch linkStatus {
	case simapi.LinkStatus_LINK_UP:
		port.Enabled = true
	default:
		port.Enabled = false
	}
	if ln, ok := ds.simulation.usedEgressPorts[id]; ok {
		if ln.link != nil {
			ln.link.Status = linkStatus
		}
	}
	return nil
}

// IsMaster returns an error if the given election ID is not the master for the specified device (chassis) and role.
func (ds *DeviceSimulator) IsMaster(chassisID uint64, role string, electionID *p4api.Uint128) error {
	if chassisID != ds.Device.ChassisID {
		return errors.NewConflict("incorrect device ID: %d", chassisID)
	}
	winningElectionID, ok := ds.roleElections[role]
	if !ok || winningElectionID.High != electionID.High || winningElectionID.Low != electionID.Low {
		return errors.NewUnauthorized("not master for role %s on device ID: %d", role, chassisID)
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
		for _, r := range ds.streamResponders {
			if r.IsMaster(role, maxElectionID) {
				failCode = code.Code_ALREADY_EXISTS
				break
			}
		}
	}

	// Notify all responders for the role
	for _, r := range ds.streamResponders {
		r.SendMastershipArbitration(role, maxElectionID, failCode)
	}

	return nil
}

// AddStreamResponder adds the given stream responder to the specified device
func (ds *DeviceSimulator) AddStreamResponder(responder StreamResponder) {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	ds.streamResponders = append(ds.streamResponders, responder)
}

// RemoveStreamResponder removes the specified stream responder from the specified device
func (ds *DeviceSimulator) RemoveStreamResponder(responder StreamResponder) {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	i := 0
	for _, r := range ds.streamResponders {
		if r == responder {
			ds.streamResponders[i] = ds.streamResponders[len(ds.streamResponders)-1]
			ds.streamResponders[len(ds.streamResponders)-1] = nil
			ds.streamResponders = ds.streamResponders[:len(ds.streamResponders)-1] // Truncate
			return
		}
		i++
	}
}

// SendToAllResponders sends the specified message to all responders
func (ds *DeviceSimulator) SendToAllResponders(response *p4api.StreamMessageResponse) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()
	for _, r := range ds.streamResponders {
		r.Send(response)
	}
}

// AddSubscribeResponder adds the given subscribe responder to the specified device
func (ds *DeviceSimulator) AddSubscribeResponder(responder SubscribeResponder) {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	ds.subscribeResponders = append(ds.subscribeResponders, responder)
}

// RemoveSubscribeResponder removes the specified subscribe responder from the specified device
func (ds *DeviceSimulator) RemoveSubscribeResponder(responder SubscribeResponder) {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	i := 0
	for _, r := range ds.subscribeResponders {
		if r == responder {
			ds.subscribeResponders[i] = ds.subscribeResponders[len(ds.subscribeResponders)-1]
			ds.subscribeResponders[len(ds.subscribeResponders)-1] = nil
			ds.subscribeResponders = ds.subscribeResponders[:len(ds.subscribeResponders)-1] // Truncate
			return
		}
		i++
	}
}

// SetPipelineConfig sets the forwarding pipeline configuration for the device
func (ds *DeviceSimulator) SetPipelineConfig(fpc *p4api.ForwardingPipelineConfig) error {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	ds.forwardingPipelineConfig = fpc

	// Update the device pipeline info
	ds.Device.PipelineInfo = &simapi.PipelineInfo{
		Cookie: fpc.Cookie.Cookie,
		P4Info: utils.P4InfoBytes(fpc.P4Info),
	}

	ds.codec = utils.NewControllerMetadataCodec(fpc.P4Info)

	// Create the required entities, e.g. tables, counters, meters, etc.
	info := fpc.P4Info
	ds.tables = entries.NewTables(info.Tables)
	ds.counters = entries.NewCounters(info.Counters)
	ds.meters = entries.NewMeters(info.Meters)

	ds.findPuntToCPUTables()

	// Snapshot the initial state of the pipeline information stats
	ds.snapshotTables()
	ds.snapshotCounters()
	ds.snapshotMeters()
	return nil
}

func (ds *DeviceSimulator) snapshotTables() {
	if ds.tables != nil {
		tables := ds.tables.Tables()
		infos := make([]*simapi.EntitiesInfo, 0, len(tables))
		for _, table := range tables {
			infos = append(infos, &simapi.EntitiesInfo{ID: table.ID(), Size_: uint32(table.Size()), Name: table.Name()})
		}
		ds.Device.PipelineInfo.Tables = infos
	}
}

func (ds *DeviceSimulator) snapshotCounters() {
	if ds.counters != nil {
		counters := ds.counters.Counters()
		infos := make([]*simapi.EntitiesInfo, 0, len(counters))
		for _, counter := range counters {
			infos = append(infos, &simapi.EntitiesInfo{ID: counter.ID(), Size_: uint32(counter.Size())})
		}
		ds.Device.PipelineInfo.Counters = infos
	}
}

func (ds *DeviceSimulator) snapshotMeters() {
	if ds.meters != nil {
		meters := ds.meters.Meters()
		infos := make([]*simapi.EntitiesInfo, 0, len(meters))
		for _, meter := range meters {
			infos = append(infos, &simapi.EntitiesInfo{ID: meter.ID(), Size_: uint32(meter.Size())})
		}
		ds.Device.PipelineInfo.Meters = infos
	}
}

// GetPipelineConfig sets the forwarding pipeline configuration for the device
func (ds *DeviceSimulator) GetPipelineConfig() *p4api.ForwardingPipelineConfig {
	return ds.forwardingPipelineConfig
}

// ProcessPacketOut handles the specified packet out message
func (ds *DeviceSimulator) ProcessPacketOut(packetOut *p4api.PacketOut, responder StreamResponder) error {
	log.Debugf("Device %s: received packet out: %+v", ds.Device.ID, packetOut)

	if ds.codec == nil {
		log.Errorf("Device %s: pipeline config not set", ds.Device.ID)
		return errors.NewInvalid("pipeline config not set yet for %d", ds.Device.ID)
	}

	// Extract the packet-out metadata
	pom := ds.codec.DecodePacketOutMetadata(packetOut.Metadata)

	// Start by decoding the packet
	packet := gopacket.NewPacket(packetOut.Payload, layers.LayerTypeEthernet, gopacket.Default)

	log.Debugf("metadata: %+v; packet: %+v; lldp: %+v", pom, packet, packet.Layer(layers.LayerTypeLinkLayerDiscovery))

	// See if this is an LLDP packet and process it if so
	if lldpLayer := packet.Layer(layers.LayerTypeLinkLayerDiscovery); lldpLayer != nil {
		ds.processLLDPPacket(packet, packetOut, pom)
	}

	// Process ARP packets
	// Process DHCP packets
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
func (ds *DeviceSimulator) processLLDPPacket(packet gopacket.Packet, packetOut *p4api.PacketOut, pom *utils.PacketOutMetadata) {
	log.Debugf("Device %s: processing LLDP packet: %+v", ds.Device.ID, packet)

	// Find the port corresponding to the specified port ID, which is the internal (SDN) port number
	egressPort, ok := ds.sdnPorts[pom.EgressPort]
	if !ok {
		log.Warnf("Device %s: Port %d not found", ds.Device.ID, pom.EgressPort)
		return
	}

	// Check if the egress port is enabled, if not, bail
	if !egressPort.Enabled {
		log.Debugf("Device %s: Port %s is presently disabled", ds.Device.ID, egressPort.ID)
		return
	}

	// Check if the given port has a link originating from it
	if link := ds.simulation.GetLinkFromPort(egressPort.ID); link != nil {
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

		ingressPort, ok := tgtDevice.Ports[link.TgtID]
		if !ok {
			log.Warnf("Device %s: Unable to locate target port %s", tgtDeviceID, link.TgtID)
		}

		if tgtDevice.HasPuntRuleForEthType(layers.EthernetTypeLinkLayerDiscovery) {
			tgtDevice.SendPacketIn(packetOut.Payload, ingressPort.InternalNumber)
		}
	}
}

// SendPacketIn emits packet in with the specified packet payload and ingress port metadata,
// to all current responders (streams) associated with this device
func (ds *DeviceSimulator) SendPacketIn(packet []byte, ingressPort uint32) {
	if ds.codec == nil {
		log.Debugf("Device %s: Unable to send packet-in, pipeline config not set yet", ds.Device.ID)
		return
	}
	packetIn := &p4api.StreamMessageResponse{
		Update: &p4api.StreamMessageResponse_Packet{
			Packet: &p4api.PacketIn{
				Payload:  packet,
				Metadata: ds.codec.EncodePacketInMetadata(&utils.PacketInMetadata{IngressPort: ingressPort}),
			},
		},
	}
	ds.SendToAllResponders(packetIn)
}

// ProcessWrite processes the specified batch of updates
func (ds *DeviceSimulator) ProcessWrite(atomicity p4api.WriteRequest_Atomicity, updates []*p4api.Update) error {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	for _, update := range updates {
		switch {
		case update.Type == p4api.Update_INSERT:
			if err := ds.processModify(update, true); err != nil {
				log.Warnf("Device %s: Unable to insert entry: %+v", ds.Device.ID, err)
				return err
			}
		case update.Type == p4api.Update_MODIFY:
			if err := ds.processModify(update, false); err != nil {
				log.Warnf("Device %s: Unable to update entry: %+v", ds.Device.ID, err)
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
		if err == nil {
			ds.checkPuntToCPU()
		}
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
		if err == nil {
			ds.checkPuntToCPU()
		}
	case entity.GetCounterEntry() != nil:
		return errors.NewInvalid("counter cannot be deleted")
	case entity.GetDirectCounterEntry() != nil:
		err = errors.NewInvalid("direct counter entry cannot be deleted")
	case entity.GetMeterEntry() != nil:
		return errors.NewInvalid("meter cannot be deleted")
	case entity.GetDirectMeterEntry() != nil:
		err = errors.NewInvalid("direct meter entry cannot be deleted")

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

// ProcessConfigGet handles the configuration get request
func (ds *DeviceSimulator) ProcessConfigGet(prefix *gnmi.Path, paths []*gnmi.Path) ([]*gnmi.Notification, error) {
	notifications := make([]*gnmi.Notification, 0, len(paths))
	rootNode := ds.config
	if prefix != nil {
		ps := utils.ToString(prefix)
		if rootNode = rootNode.GetPath(ps); rootNode == nil {
			return nil, errors.NewInvalid("node with given prefix %s not found", ps)
		}
	}

	for _, path := range paths {
		nodes := rootNode.FindAll(utils.ToString(path))
		if len(nodes) > 0 {
			notifications = append(notifications, toNotification(prefix, nodes))
		}
	}

	// TODO: implement proper error handling
	return notifications, nil
}

// Creates a notification message from the specified nodes
func toNotification(prefix *gnmi.Path, nodes []*config.Node) *gnmi.Notification {
	updates := make([]*gnmi.Update, 0, len(nodes))
	for _, node := range nodes {
		updates = append(updates, toUpdate(node))
	}
	return &gnmi.Notification{
		Timestamp: 0,
		Prefix:    prefix,
		Update:    updates,
	}
}

// Creates an update message from the specified node
func toUpdate(node *config.Node) *gnmi.Update {
	return &gnmi.Update{
		Path:       utils.ToPath(node.Path()),
		Val:        node.Value(),
		Duplicates: 0,
	}
}

// ProcessConfigSet handles the configuration set request
func (ds *DeviceSimulator) ProcessConfigSet(prefix *gnmi.Path,
	updates []*gnmi.Update, replacements []*gnmi.Update, deletes []*gnmi.Path) ([]*gnmi.UpdateResult, error) {
	opCount := len(updates) + len(replacements) + len(deletes)
	if opCount < 1 {
		return nil, errors.Status(errors.NewInvalid("no updates, replace or deletes")).Err()
	}
	results := make([]*gnmi.UpdateResult, 0, opCount)

	rootNode := ds.config
	if prefix != nil {
		ps := utils.ToString(prefix)
		if rootNode = rootNode.GetPath(ps); rootNode == nil {
			return nil, errors.NewInvalid("node with given prefix %s not found", ps)
		}
	}

	for _, path := range deletes {
		rootNode.DeletePath(utils.ToString(path))
	}

	for _, update := range replacements {
		rootNode.ReplacePath(utils.ToString(update.Path), update.Val)
	}

	for _, update := range updates {
		rootNode.AddPath(utils.ToString(update.Path), update.Val)
	}

	// TODO: Implement processing of the new configuration and return proper result; error handling
	return results, nil
}

// HasPuntRuleForEthType returns true if the device has a table with punt-to-CPU action installed in one
// of its tables
func (ds *DeviceSimulator) HasPuntRuleForEthType(ethType layers.EthernetType) bool {
	v, ok := ds.puntToCPU[ethType]
	return ok && v
}

// Searches all tables with "acl" or "ACL" in their name and looks for rules with action punt to CPU
// and registers the matching ETH type
func (ds *DeviceSimulator) checkPuntToCPU() {
	ds.puntToCPU = make(map[layers.EthernetType]bool)
	for _, table := range ds.cpuTables {
		// Search entries for all CPU related tables
		for _, entry := range table.table.Entries() {
			action := entry.Action.GetAction()
			if action != nil {
				if _, ok := ds.cpuActions[action.ActionId]; ok {
					// If entry has a CPU related action, find the match referencing ethType exact match
					for _, match := range entry.Match {
						if match.FieldId == table.ethTypeFieldID && match.GetTernary() != nil {
							// Record that this ethType has a punt-to-cpu (or related) action
							ethType := binary.BigEndian.Uint16(match.GetTernary().Value)
							ds.puntToCPU[layers.EthernetType(ethType)] = true
						}
					}
				}
			}
		}
	}
	log.Infof("Device %s: puntToCPU=%+v", ds.Device.ID, ds.puntToCPU)
}

// Finds all tables that have CPU-related action references and creates auxiliary search structures to
// facilitate speedy check for punt rules after table modifications.
func (ds *DeviceSimulator) findPuntToCPUTables() {
	ds.cpuActions = make(map[uint32]*p4info.Action)
	for _, action := range ds.forwardingPipelineConfig.P4Info.Actions {
		if strings.Contains(action.Preamble.Name, "_to_cpu") {
			ds.cpuActions[action.Preamble.Id] = action
		}
	}

	ds.cpuTables = make(map[uint32]*cpuTable)
	for _, table := range ds.forwardingPipelineConfig.P4Info.Tables {
		if ds.hasCPUAction(table) {
			if f := ds.findEthTypeMatchField(table); f != nil {
				ds.cpuTables[table.Preamble.Id] = &cpuTable{
					table:          ds.tables.Table(table.Preamble.Id),
					ethTypeFieldID: f.Id}
			}
		}
	}

	log.Infof("Device %s: cpuActions=%+v", ds.Device.ID, ds.cpuActions)
	log.Infof("Device %s: cpuTables=%+v", ds.Device.ID, ds.cpuTables)
}

// Returns true if the table has a reference to a CPU related action
func (ds *DeviceSimulator) hasCPUAction(table *p4info.Table) bool {
	for _, aref := range table.ActionRefs {
		if _, ok := ds.cpuActions[aref.Id]; ok {
			return true
		}
	}
	return false
}

// Finds the field match with "ethType" as its name
func (ds *DeviceSimulator) findEthTypeMatchField(table *p4info.Table) *p4info.MatchField {
	for _, field := range table.MatchFields {
		if field.Name == "eth_type" {
			return field
		}
	}
	return nil
}

// TODO: Additional simulation logic goes here
