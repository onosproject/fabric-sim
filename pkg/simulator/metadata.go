// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"encoding/binary"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
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
}

// DecodePacketOutMetadata decodes the received metadata into an internal structure
func (c *ControllerMetadataCodec) DecodePacketOutMetadata(md []*p4api.PacketMetadata) *PacketOutMetadata {
	pom := &PacketOutMetadata{}
	for _, m := range md {
		if m.MetadataId == c.egress.id {
			pom.EgressPort = parseMetadataValue(m.Value)
		}
	}
	return pom
}

// EncodePacketOutMetadata encodes the metadata into an external representation
func (c *ControllerMetadataCodec) EncodePacketOutMetadata(pom *PacketOutMetadata) []*p4api.PacketMetadata {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, pom.EgressPort)
	b = trimToBitwidth(b, c.egress.size)
	log.Infof("%+v", b)

	return []*p4api.PacketMetadata{
		{MetadataId: c.egress.id, Value: b},
		{MetadataId: c.opad.id, Value: []byte{0}},
	}
}

// DecodePacketInMetadata decodes the received metadata into an internal structure
func (c *ControllerMetadataCodec) DecodePacketInMetadata(md []*p4api.PacketMetadata) *PacketInMetadata {
	pim := &PacketInMetadata{}
	for _, m := range md {
		if m.MetadataId == c.ingress.id {
			pim.IngressPort = parseMetadataValue(m.Value)
		}
	}
	return pim
}

// EncodePacketInMetadata encodes the metadata into an external representation
func (c *ControllerMetadataCodec) EncodePacketInMetadata(pim *PacketInMetadata) []*p4api.PacketMetadata {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, pim.IngressPort)
	b = trimToBitwidth(b, c.ingress.size)
	log.Infof("%+v", b)

	return []*p4api.PacketMetadata{
		{MetadataId: c.ingress.id, Value: b},
		{MetadataId: c.ipad.id, Value: []byte{0}},
	}
}

func trimToBitwidth(b []byte, bits int32) []byte {
	byteCount := int(bits) / 2
	ni := len(b) - byteCount
	for ; ni < len(b); ni++ {
		if b[ni] != 0 {
			break
		}
	}
	return b[ni:]
}

func parseMetadataValue(value []byte) uint32 {
	b := make([]byte, 4)
	offset := len(b) - len(value)
	for i := 0; i < len(value); i++ {
		b[offset+i] = value[i]
	}
	return binary.BigEndian.Uint32(b)
}
