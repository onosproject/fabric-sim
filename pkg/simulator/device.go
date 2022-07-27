// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-lib-go/pkg/errors"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"strconv"
	"sync"
)

var log = logging.GetLogger("simulator", "device")

// DeviceSimulator simulates a single device
type DeviceSimulator struct {
	Device                   *simapi.Device
	Ports                    map[simapi.PortID]*simapi.Port
	Agent                    DeviceAgent
	ForwardingPipelineConfig *p4api.ForwardingPipelineConfig

	lock          sync.RWMutex
	roleElections map[uint64]*p4api.Uint128
	responders    []StreamResponder
	simulation    *Simulation
	sdnPorts      map[uint32]*simapi.Port
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
		roleElections: make(map[uint64]*p4api.Uint128),
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

// RecordRoleElection checks the given election ID for the specified role and records it
// if the given election ID is larger than a previously recorded election ID for the same
// role. It returns error (if election for role not secured) and the latest election ID for the role.
func (ds *DeviceSimulator) RecordRoleElection(role *p4api.Role, electionID *p4api.Uint128) (*p4api.Uint128, error) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	roleID := uint64(0)
	if role != nil {
		roleID = role.Id
	}

	maxID, ok := ds.roleElections[roleID]
	if !ok || isNewMaster(maxID, electionID) {
		ds.roleElections[roleID] = electionID
		return electionID, nil
	}
	return maxID, errors.NewInvalid("Mastership for role %d has not been secured with election ID %d",
		roleID, electionID)
}

func isNewMaster(current *p4api.Uint128, new *p4api.Uint128) bool {
	return current.High < new.High || (current.High == new.High && current.Low < new.Low)
}

// ProcessMastershipArbitration processes the specified arbitration update
func (ds *DeviceSimulator) ProcessMastershipArbitration(arbitration *p4api.MasterArbitrationUpdate, responder StreamResponder) {
	log.Debugf("Device %s: received mastership arbitration: %+v", ds.Device.ID, arbitration)

	electionStatus := &status.Status{Code: int32(code.Code_OK)}
	maxElectionID, err := ds.RecordRoleElection(arbitration.Role, arbitration.ElectionId)
	if err != nil {
		electionStatus.Code = int32(code.Code_PERMISSION_DENIED)
		electionStatus.Message = err.Error()
	}
	// Respond directly to the responder corresponding to the stream from where we received the message
	responder.Send(&p4api.StreamMessageResponse{
		Update: &p4api.StreamMessageResponse_Arbitration{
			Arbitration: &p4api.MasterArbitrationUpdate{
				DeviceId:   arbitration.DeviceId,
				Role:       arbitration.Role,
				ElectionId: maxElectionID,
				Status:     electionStatus,
			},
		},
	})

	// FIXME: Respond to other stream responders as well
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

// ProcessPacketOut handles the specified packet out message
func (ds *DeviceSimulator) ProcessPacketOut(packetOut *p4api.PacketOut, responder StreamResponder) {
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
}

// ProcessDigestAck handles the specified digest list ack message
func (ds *DeviceSimulator) ProcessDigestAck(ack *p4api.DigestListAck, responder StreamResponder) {
	log.Infof("Device %s: received digest ack: %+v", ds.Device.ID, ack)
	// TODO: Implement this
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

// TODO: Additional simulation logic goes here
