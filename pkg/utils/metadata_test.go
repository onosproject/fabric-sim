// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPacketOutMetadata(t *testing.T) {
	info, err := LoadP4Info("../../pipelines/fabric-spgw-int.p4info.txt")
	assert.NoError(t, err)

	codec := NewControllerMetadataCodec(info)

	pom := PacketOutMetadata{EgressPort: 213}
	md := codec.EncodePacketOutMetadata(&pom)
	assert.Len(t, md, 2)

	pom1 := codec.DecodePacketOutMetadata(md)
	assert.Equal(t, pom.EgressPort, pom1.EgressPort)

	pom = PacketOutMetadata{EgressPort: 413}
	md = codec.EncodePacketOutMetadata(&pom)
	assert.Len(t, md, 2)

	pom1 = codec.DecodePacketOutMetadata(md)
	assert.Equal(t, pom.EgressPort, pom1.EgressPort)
}

func TestPacketInMetadata(t *testing.T) {
	info, err := LoadP4Info("../../pipelines/fabric-spgw-int.p4info.txt")
	assert.NoError(t, err)

	codec := NewControllerMetadataCodec(info)

	pim := PacketInMetadata{IngressPort: 243, RoleAgentID: 15}
	md := codec.EncodePacketInMetadata(&pim)
	assert.Len(t, md, 3)

	pom1 := codec.DecodePacketInMetadata(md)
	assert.Equal(t, pim.IngressPort, pom1.IngressPort)
	assert.Equal(t, pim.RoleAgentID, pom1.RoleAgentID)

	pim = PacketInMetadata{IngressPort: 343, RoleAgentID: 1}
	md = codec.EncodePacketInMetadata(&pim)
	assert.Len(t, md, 3)

	pom1 = codec.DecodePacketInMetadata(md)
	assert.Equal(t, pim.IngressPort, pom1.IngressPort)
	assert.Equal(t, pim.RoleAgentID, pom1.RoleAgentID)
}
