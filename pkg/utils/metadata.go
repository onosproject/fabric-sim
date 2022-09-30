// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"encoding/binary"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"math"
)

type meta struct {
	id   uint32
	size int32
}

// ControllerMetadataCodec allows basic encoding and decoding of packet out/in metadata
type ControllerMetadataCodec struct {
	egress  meta
	opad    meta
	ingress meta
	ipad    meta
	roleid  meta
}

// NewControllerMetadataCodec creates a new codec from the supplied P4 info
func NewControllerMetadataCodec(info *p4info.P4Info) *ControllerMetadataCodec {
	cmc := &ControllerMetadataCodec{}
	for _, cpm := range info.ControllerPacketMetadata {
		switch cpm.Preamble.Name {
		case "packet_out":
			for _, m := range cpm.Metadata {
				switch m.Name {
				case "egress_port":
					copyMeta(m, &cmc.egress)
				case "_pad":
					copyMeta(m, &cmc.opad)
				}
			}
		case "packet_in":
			for _, m := range cpm.Metadata {
				switch m.Name {
				case "ingress_port":
					copyMeta(m, &cmc.ingress)
				case "role_agent_id":
					copyMeta(m, &cmc.roleid)
				case "_pad":
					copyMeta(m, &cmc.ipad)
				}
			}
		}
	}
	return cmc
}

func copyMeta(md *p4info.ControllerPacketMetadata_Metadata, m *meta) {
	m.id = md.Id
	m.size = md.Bitwidth
}

// PacketOutMetadata carries basic packet-out metadata contents
type PacketOutMetadata struct {
	EgressPort uint32
}

// PacketInMetadata carries basic packet-in metadata contents
type PacketInMetadata struct {
	IngressPort uint32
	RoleAgentID uint32
}

// DecodePacketOutMetadata decodes the received metadata into an internal structure
func (c *ControllerMetadataCodec) DecodePacketOutMetadata(md []*p4api.PacketMetadata) *PacketOutMetadata {
	pom := &PacketOutMetadata{}
	for _, m := range md {
		if m.MetadataId == c.egress.id {
			pom.EgressPort = DecodeValueAsUint32(m.Value)
		}
	}
	return pom
}

// EncodePacketOutMetadata encodes the metadata into an external representation
func (c *ControllerMetadataCodec) EncodePacketOutMetadata(pom *PacketOutMetadata) []*p4api.PacketMetadata {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, pom.EgressPort)
	b = TrimToBitwidth(b, c.egress.size)
	metadata := []*p4api.PacketMetadata{{MetadataId: c.egress.id, Value: b}}
	if c.opad.id != 0 {
		metadata = append(metadata, &p4api.PacketMetadata{MetadataId: c.opad.id, Value: []byte{0}})
	}
	return metadata

}

// DecodePacketInMetadata decodes the received metadata into an internal structure
func (c *ControllerMetadataCodec) DecodePacketInMetadata(md []*p4api.PacketMetadata) *PacketInMetadata {
	pim := &PacketInMetadata{}
	for _, m := range md {
		if m.MetadataId == c.ingress.id {
			pim.IngressPort = DecodeValueAsUint32(m.Value)
		} else if m.MetadataId == c.roleid.id {
			pim.RoleAgentID = DecodeValueAsUint32(m.Value)
		}
	}
	return pim
}

// EncodePacketInMetadata encodes the metadata into an external representation
func (c *ControllerMetadataCodec) EncodePacketInMetadata(pim *PacketInMetadata) []*p4api.PacketMetadata {
	b1 := make([]byte, 4)
	binary.BigEndian.PutUint32(b1, pim.IngressPort)
	b1 = TrimToBitwidth(b1, c.ingress.size)

	b2 := make([]byte, 4)
	binary.BigEndian.PutUint32(b2, pim.RoleAgentID)
	b2 = TrimToBitwidth(b2, c.roleid.size)

	metadata := []*p4api.PacketMetadata{{MetadataId: c.ingress.id, Value: b1}, {MetadataId: c.roleid.id, Value: b2}}

	// Tack on padding if needed
	if c.ipad.id != 0 {
		metadata = append(metadata, &p4api.PacketMetadata{MetadataId: c.ipad.id, Value: []byte{0}})
	}
	return metadata
}

// TrimToBitwidth trims the specified bytes to the specified width
func TrimToBitwidth(b []byte, bits int32) []byte {
	byteCount := int(math.Ceil(float64(bits) / 8.0)) // compute bytes needed from the bit-width
	ni := len(b) - byteCount
	if ni < 0 {
		return b
	}
	for ; ni < len(b); ni++ {
		if b[ni] != 0 {
			break
		}
	}
	return b[ni:]
}

// DecodeValueAsUint32 decodes the specified bytes as uint32 value
func DecodeValueAsUint32(value []byte) uint32 {
	b := make([]byte, 4)
	offset := len(b) - len(value)
	if offset >= 0 {
		for i := 0; i < len(value); i++ {
			b[offset+i] = value[i]
		}
	}
	return binary.BigEndian.Uint32(b)
}
