// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"bytes"
	"context"
	"encoding/binary"
	gogo "github.com/gogo/protobuf/types"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/onosproject/fabric-sim/pkg/simulator/config"
	"github.com/onosproject/fabric-sim/pkg/simulator/entries"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-api/go/onos/misc"
	"github.com/onosproject/onos-api/go/onos/stratum"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-net-lib/pkg/configtree"
	utils "github.com/onosproject/onos-net-lib/pkg/gnmiutils"
	"github.com/onosproject/onos-net-lib/pkg/p4utils"
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
	configtree.Configurable
	configtree.GNMIConfigurable

	Device *simapi.Device
	Ports  map[simapi.PortID]*simapi.Port
	Agent  DeviceAgent

	lock                     sync.RWMutex
	forwardingPipelineConfig *p4api.ForwardingPipelineConfig
	streamResponders         []StreamResponder
	subscribeResponders      []SubscribeResponder
	roleConfigs              map[string]*roleConfig
	simulation               *Simulation
	sdnPorts                 map[uint32]*simapi.Port

	tables   *entries.Tables
	counters *entries.Counters
	meters   *entries.Meters
	profiles *entries.ActionProfiles
	pre      *entries.PacketReplication

	config     *configtree.Node
	codec      *p4utils.ControllerMetadataCodec
	puntToCPU  map[layers.EthernetType]uint32
	cpuActions map[uint32]*cpuAction
	cpuTables  map[uint32]*cpuTable

	cancel context.CancelFunc

	ioStatsLock sync.RWMutex
}

// IOStats represents cumulative I/O stats
type IOStats struct {
	InBytes     uint32
	InMessages  uint32
	OutBytes    uint32
	OutMessages uint32
	SinceTime   time.Time
}

type roleConfig struct {
	electionID *p4api.Uint128
	config     *stratum.P4RoleConfig
}

// Auxiliary structure to track table that has CPU related actions and the field match related to ETH type
type cpuTable struct {
	table          *entries.Table
	ethTypeFieldID uint32
}

// Auxiliary structure to track punt/copy to CPU actions and their associated role agent ID parameter ID
type cpuAction struct {
	action              *p4info.Action
	roleAgentIDParamID  uint32
	roleAgentIDBitwidth int32
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
	device.Connections = make([]*misc.Connection, 0)
	device.IOStats = &misc.IOStats{FirstUpdateTime: uint64(time.Now().UnixNano())}

	// Construct and return simulator from the device and the port map
	dsim := &DeviceSimulator{
		GNMIConfigurable: *configtree.NewGNMIConfigurable(cfg),
		Device:           device,
		Ports:            ports,
		Agent:            agent,
		forwardingPipelineConfig: &p4api.ForwardingPipelineConfig{
			P4Info:         &p4info.P4Info{},
			P4DeviceConfig: []byte{},
			Cookie:         &p4api.ForwardingPipelineConfig_Cookie{Cookie: 0},
		},
		roleConfigs: make(map[string]*roleConfig),
		sdnPorts:    sdnPorts,
		simulation:  simulation,
		config:      cfg,
		puntToCPU:   make(map[layers.EthernetType]uint32),
		cpuActions:  make(map[uint32]*cpuAction),
		cpuTables:   make(map[uint32]*cpuTable),
	}
	dsim.GNMIConfigurable.Configurable = dsim
	return dsim
}

// UpdateIOStats updates the device I/O stats
func (ds *DeviceSimulator) UpdateIOStats(byteCount int, input bool) {
	ds.ioStatsLock.Lock()
	defer ds.ioStatsLock.Unlock()
	stats := ds.Device.IOStats
	if input {
		stats.InMessages++
		stats.InBytes += uint32(byteCount)
	} else {
		stats.OutMessages++
		stats.OutBytes += uint32(byteCount)
	}
	stats.LastUpdateTime = uint64(time.Now().UnixNano())
}

func (ds *DeviceSimulator) addAndResetStats(now uint64, total *misc.IOStats) {
	ds.ioStatsLock.Lock()
	defer ds.ioStatsLock.Unlock()
	stats := ds.Device.IOStats

	total.InBytes += stats.InBytes
	total.InMessages += stats.InMessages
	total.OutBytes += stats.OutBytes
	total.OutMessages += stats.OutMessages

	stats.InBytes = 0
	stats.InMessages = 0
	stats.OutBytes = 0
	stats.OutMessages = 0
	stats.FirstUpdateTime = now
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
	ds.snapshotGroups()
	ds.snapshotMulticast()
	ds.snapshotCloneSessions()
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
	rolleWinner, ok := ds.roleConfigs[role]
	if !ok || rolleWinner.electionID.High != electionID.High || rolleWinner.electionID.Low != electionID.Low {
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

	winner, ok := ds.roleConfigs[roleName]
	if !ok || winner.electionID.High < electionID.High || (winner.electionID.High == electionID.High && winner.electionID.Low < electionID.Low) {
		ds.roleConfigs[roleName] = ds.getRoleConfig(role, electionID)
		return electionID
	} else if winner.electionID.High == electionID.High && winner.electionID.Low == electionID.Low {
		return nil // this role and election ID has already been claimed
	}
	return winner.electionID
}

func (ds *DeviceSimulator) getRoleConfig(role *p4api.Role, electionID *p4api.Uint128) *roleConfig {
	rc := &stratum.P4RoleConfig{}
	if role != nil && role.Config != nil {
		any := &gogo.Any{
			TypeUrl: role.Config.TypeUrl,
			Value:   role.Config.Value,
		}
		_ = gogo.UnmarshalAny(any, rc)
		log.Debugf("Device %s: rc: %+v; any: %+v", ds.Device.ID, rc, role.Config)
	}
	return &roleConfig{electionID: electionID, config: rc}
}

// RunMastershipArbitration processes the specified arbitration update
func (ds *DeviceSimulator) RunMastershipArbitration(role *p4api.Role, electionID *p4api.Uint128) error {
	log.Infof("Device %s: running mastership arbitration for role %s and electionID %+v", ds.Device.ID, role, electionID)

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
	ds.Device.Connections = append(ds.Device.Connections, responder.GetConnection())
	ds.Device.TotalConnections++
}

// RemoveStreamResponder removes the specified stream responder from the specified device
func (ds *DeviceSimulator) RemoveStreamResponder(responder StreamResponder) {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	ds.removeConnection(responder.GetConnection())
	for i, r := range ds.streamResponders {
		if r == responder {
			ds.streamResponders = append(ds.streamResponders[:i], ds.streamResponders[i+1:]...)
			return
		}
	}
}

func (ds *DeviceSimulator) removeConnection(connection *misc.Connection) {
	log.Infof("Device %s: Removing %s connection from %s...", ds.Device.ID, connection.Protocol, connection.FromAddress)
	for i, c := range ds.Device.Connections {
		if c.FromAddress == connection.FromAddress {
			log.Infof("Device %s: Removed %s connection from %s", ds.Device.ID, connection.Protocol, connection.FromAddress)
			ds.Device.Connections = append(ds.Device.Connections[:i], ds.Device.Connections[i+1:]...)
			return
		}
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
	ds.Device.Connections = append(ds.Device.Connections, responder.GetConnection())
	ds.Device.TotalConnections++
}

// RemoveSubscribeResponder removes the specified subscribe responder from the specified device
func (ds *DeviceSimulator) RemoveSubscribeResponder(responder SubscribeResponder) {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	ds.removeConnection(responder.GetConnection())
	for i, r := range ds.subscribeResponders {
		if r == responder {
			ds.subscribeResponders = append(ds.subscribeResponders[:i], ds.subscribeResponders[i+1:]...)
			return
		}
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
		P4Info: p4utils.P4InfoBytes(fpc.P4Info),
	}

	ds.codec = p4utils.NewControllerMetadataCodec(fpc.P4Info)

	// Create the required entities, e.g. tables, counters, meters, etc.
	info := fpc.P4Info
	ds.tables = entries.NewTables(info.Tables)
	ds.counters = entries.NewCounters(info.Counters)
	ds.meters = entries.NewMeters(info.Meters)
	ds.profiles = entries.NewActionProfiles(info.ActionProfiles)
	ds.pre = entries.NewPacketReplication()

	ds.findPuntToCPUTables()

	// Snapshot the initial state of the pipeline information stats
	ds.snapshotTables()
	ds.snapshotCounters()
	ds.snapshotMeters()
	ds.snapshotGroups()
	ds.snapshotMulticast()
	ds.snapshotCloneSessions()
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

func (ds *DeviceSimulator) snapshotGroups() {
	if ds.profiles != nil {
		groups := ds.profiles.Groups()
		infos := make([]*simapi.EntitiesInfo, 0, len(groups))
		for _, group := range groups {
			infos = append(infos, &simapi.EntitiesInfo{ID: group.ID(), Size_: uint32(group.Size()), Name: group.Name()})
		}
		ds.Device.PipelineInfo.Groups = infos
	}
}

func (ds *DeviceSimulator) snapshotMulticast() {
	if ds.pre != nil {
		groups := ds.pre.MulticastGroups()
		infos := make([]*simapi.EntitiesInfo, 0, len(groups))
		for _, group := range groups {
			infos = append(infos, &simapi.EntitiesInfo{ID: group.MulticastGroupId, Size_: uint32(len(group.Replicas)), Name: "PRE.MulticastGroup"})
		}
		ds.Device.PipelineInfo.MulticastGroups = infos
	}
}

func (ds *DeviceSimulator) snapshotCloneSessions() {
	if ds.pre != nil {
		sessions := ds.pre.CloneSessions()
		infos := make([]*simapi.EntitiesInfo, 0, len(sessions))
		for _, session := range sessions {
			infos = append(infos, &simapi.EntitiesInfo{ID: session.SessionId, Size_: uint32(len(session.Replicas)), Name: "PRE.CloneSession"})
		}
		ds.Device.PipelineInfo.CloneSessions = infos
	}
}

// GetPipelineConfig returns a copy of the forwarding pipeline configuration for the device
func (ds *DeviceSimulator) GetPipelineConfig() *p4api.ForwardingPipelineConfig {
	return &p4api.ForwardingPipelineConfig{
		P4Info:         ds.forwardingPipelineConfig.P4Info,
		P4DeviceConfig: ds.forwardingPipelineConfig.P4DeviceConfig,
		Cookie:         ds.forwardingPipelineConfig.Cookie,
	}
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

	// See if this is an LLDP packet and process it if so
	if lldpLayer := packet.Layer(layers.LayerTypeLinkLayerDiscovery); lldpLayer != nil {
		ds.processLLDPPacket(packet, packetOut, pom)
	}

	// Process ARP packets
	if arpLayer := packet.Layer(layers.LayerTypeARP); arpLayer != nil {
		// TODO: Implement recording ARP response packets
		log.Infof("Device %s: arpLayer=%+v", ds.Device.ID, arpLayer.(*layers.ARP))
	}

	// Process DHCP packets
	if dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4); dhcpLayer != nil {
		// TODO: Implement recording DHCP response packets
		log.Infof("Device %s: dhcpLayer=%+v", ds.Device.ID, dhcpLayer.(*layers.DHCPv4))
	}

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
func (ds *DeviceSimulator) processLLDPPacket(packet gopacket.Packet, packetOut *p4api.PacketOut, pom *p4utils.PacketOutMetadata) {
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

		if roleAgentID, ok := tgtDevice.HasPuntRuleForEthType(layers.EthernetTypeLinkLayerDiscovery); ok {
			tgtDevice.SendPacketIn(packetOut.Payload, &p4utils.PacketInMetadata{
				IngressPort: ingressPort.InternalNumber,
				RoleAgentID: roleAgentID,
			})
		}
	}
}

// SendPacketIn emits packet in with the specified packet payload and ingress port metadata,
// to all current responders (streams) associated with this device
func (ds *DeviceSimulator) SendPacketIn(packet []byte, md *p4utils.PacketInMetadata) {
	if ds.codec == nil {
		log.Debugf("Device %s: Unable to send packet-in, pipeline config not set yet", ds.Device.ID)
		return
	}
	metadata := ds.codec.EncodePacketInMetadata(md)
	packetIn := &p4api.StreamMessageResponse{
		Update: &p4api.StreamMessageResponse_Packet{
			Packet: &p4api.PacketIn{
				Payload:  packet,
				Metadata: metadata,
			},
		},
	}

	ds.lock.RLock()
	defer ds.lock.RUnlock()
	for _, r := range ds.streamResponders {
		if matchesMetaData(r.GetRoleConfig(), metadata) {
			r.Send(packetIn)
		}
	}
}

func matchesMetaData(roleConfig *stratum.P4RoleConfig, metadata []*p4api.PacketMetadata) bool {
	if roleConfig == nil || (roleConfig.ReceivesPacketIns && roleConfig.PacketInFilter == nil) {
		return true
	}
	if roleConfig.ReceivesPacketIns {
		for _, md := range metadata {
			if md.MetadataId == roleConfig.PacketInFilter.MetadataId {
				return bytes.Equal(md.Value, roleConfig.PacketInFilter.Value)
			}
		}
	}
	return false
}

// ProcessWrite processes the specified batch of updates
func (ds *DeviceSimulator) ProcessWrite(atomicity p4api.WriteRequest_Atomicity, updates []*p4api.Update) error {
	ds.lock.Lock()
	defer ds.lock.Unlock()
	if ds.forwardingPipelineConfig == nil {
		return errors.NewUnavailable("Device %s: Pipeline configuration not set yet", ds.Device.ID)
	}

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

	case entity.GetActionProfileGroup() != nil:
		err = ds.profiles.ModifyActionProfileGroup(entity.GetActionProfileGroup(), isInsert)
	case entity.GetActionProfileMember() != nil:
		err = ds.profiles.ModifyActionProfileMember(entity.GetActionProfileMember(), isInsert)

	case entity.GetPacketReplicationEngineEntry() != nil:
		switch {
		case entity.GetPacketReplicationEngineEntry().GetMulticastGroupEntry() != nil:
			err = ds.pre.ModifyMulticastGroupEntry(entity.GetPacketReplicationEngineEntry().GetMulticastGroupEntry(), isInsert)
		case entity.GetPacketReplicationEngineEntry().GetCloneSessionEntry() != nil:
			err = ds.pre.ModifyCloneSessionEntry(entity.GetPacketReplicationEngineEntry().GetCloneSessionEntry(), isInsert)
		}

	case entity.GetRegisterEntry() != nil:
		log.Warnf("Device %s: RegisterEntry write is not supported yet: %+v", ds.Device.ID, entity.GetRegisterEntry())
	case entity.GetValueSetEntry() != nil:
		log.Warnf("Device %s: ValueSetEntry write is not supported yet: %+v", ds.Device.ID, entity.GetValueSetEntry())
	case entity.GetDigestEntry() != nil:
		log.Warnf("Device %s: DigestEntry write is not supported yet: %+v", ds.Device.ID, entity.GetDigestEntry())
	case entity.GetExternEntry() != nil:
		log.Warnf("Device %s: ExternEntry write is not supported yet: %+v", ds.Device.ID, entity.GetExternEntry())
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

	case entity.GetActionProfileGroup() != nil:
		err = ds.profiles.DeleteActionProfileGroup(entity.GetActionProfileGroup())
	case entity.GetActionProfileMember() != nil:
		err = ds.profiles.DeleteActionProfileMember(entity.GetActionProfileMember())

	case entity.GetPacketReplicationEngineEntry() != nil:
		switch {
		case entity.GetPacketReplicationEngineEntry().GetMulticastGroupEntry() != nil:
			err = ds.pre.DeleteMulticastGroupEntry(entity.GetPacketReplicationEngineEntry().GetMulticastGroupEntry())
		case entity.GetPacketReplicationEngineEntry().GetCloneSessionEntry() != nil:
			err = ds.pre.DeleteCloneSessionEntry(entity.GetPacketReplicationEngineEntry().GetCloneSessionEntry())
		}

	case entity.GetRegisterEntry() != nil:
	case entity.GetValueSetEntry() != nil:
	case entity.GetDigestEntry() != nil:
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

	case request.GetActionProfileGroup() != nil:
		return ds.profiles.ReadActionProfileGroups(request.GetActionProfileGroup(), sender)
	case request.GetActionProfileMember() != nil:
		return ds.profiles.ReadActionProfileMembers(request.GetActionProfileMember(), sender)

	case request.GetPacketReplicationEngineEntry() != nil:
		switch {
		case request.GetPacketReplicationEngineEntry().GetMulticastGroupEntry() != nil:
			return ds.pre.ReadMulticastGroupEntries(request.GetPacketReplicationEngineEntry().GetMulticastGroupEntry(), sender)
		case request.GetPacketReplicationEngineEntry().GetCloneSessionEntry() != nil:
			return ds.pre.ReadCloneSessionEntries(request.GetPacketReplicationEngineEntry().GetCloneSessionEntry(), sender)
		}

	case request.GetRegisterEntry() != nil:
	case request.GetValueSetEntry() != nil:
	case request.GetDigestEntry() != nil:
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
func toNotification(prefix *gnmi.Path, nodes []*configtree.Node) *gnmi.Notification {
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
func toUpdate(node *configtree.Node) *gnmi.Update {
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
func (ds *DeviceSimulator) HasPuntRuleForEthType(ethType layers.EthernetType) (uint32, bool) {
	ds.lock.RLock()
	defer ds.lock.RUnlock()
	roleAgentID, ok := ds.puntToCPU[ethType]
	return roleAgentID, ok
}

// Searches all tables with "acl" or "ACL" in their name and looks for rules with action punt to CPU
// and registers the matching ETH type from the match and the role agent ID from the action parameter
func (ds *DeviceSimulator) checkPuntToCPU() {
	ds.puntToCPU = make(map[layers.EthernetType]uint32)
	for _, table := range ds.cpuTables {
		// Search entries for all CPU related tables
		for _, entry := range table.table.Entries() {
			action := entry.Action.GetAction()
			if action != nil {
				if cpuAction, ok := ds.cpuActions[action.ActionId]; ok {
					// If entry has a CPU related action, find the match referencing ethType exact match
					for _, match := range entry.Match {
						if match.FieldId == table.ethTypeFieldID && match.GetTernary() != nil {
							// Record that this ethType has a punt-to-cpu (or related) action
							ethType := binary.BigEndian.Uint16(match.GetTernary().Value)
							ds.puntToCPU[layers.EthernetType(ethType)] = findRoleAgentID(action, cpuAction)
						}
					}
				}
			}
		}
	}
	log.Debugf("Device %s: puntToCPU=%+v", ds.Device.ID, ds.puntToCPU)
}

// Extract the role agent ID field value from the action parameters
func findRoleAgentID(action *p4api.Action, ca *cpuAction) uint32 {
	for _, param := range action.Params {
		if param.ParamId == ca.roleAgentIDParamID {
			return p4utils.DecodeValueAsUint32(param.Value)
		}
	}
	return 0
}

// Finds all tables that have CPU-related action references and creates auxiliary search structures to
// facilitate speedy check for punt rules after table modifications.
func (ds *DeviceSimulator) findPuntToCPUTables() {
	ds.cpuActions = make(map[uint32]*cpuAction)
	for _, action := range ds.forwardingPipelineConfig.P4Info.Actions {
		if strings.Contains(action.Preamble.Name, "_to_cpu") {
			pid, bw := ds.findRoleAgentParameterID(action)
			ds.cpuActions[action.Preamble.Id] = &cpuAction{
				action:              action,
				roleAgentIDParamID:  pid,
				roleAgentIDBitwidth: bw,
			}
		}
	}
	// for k, v := range ds.cpuActions {log.Infof("cpuAction %+v => %+v", k, v)}

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
	//for k, v := range ds.cpuTables {log.Infof("cpuTable %+v => %+v", k, v)}
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

func (ds *DeviceSimulator) findRoleAgentParameterID(action *p4info.Action) (uint32, int32) {
	for _, param := range action.Params {
		if param.Name == "set_role_agent_id" {
			return param.Id, param.Bitwidth
		}
	}
	return 0, 0
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

// UpdateConfig should be called after the configuration tree has been updated to save the configuration and
// to reflect it back to the controller's Config structure for easy access.
func (ds *DeviceSimulator) UpdateConfig() {
	// no-op here
}

// RefreshConfig refreshes the config tree state from any relevant external source state
func (ds *DeviceSimulator) RefreshConfig() {
	// no-op here
}
