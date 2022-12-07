// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package basic

import (
	"context"
	"github.com/onosproject/fabric-sim/pkg/utils"
	"github.com/onosproject/fabric-sim/test/client"
	simapi "github.com/onosproject/onos-api/go/onos/fabricsim"
	"github.com/onosproject/onos-net-lib/pkg/p4utils"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
	"github.com/stretchr/testify/assert"
	"io"
	"math/rand"
	"sync"
	"testing"
)

// TestReadWrite loads simulator with access fabric topology, primes all switches forwarding pipeline config
// and then writes entries into all their tables and reads them back
func (s *TestSuite) TestReadWrite(t *testing.T) {
	devices, _, _ := LoadAndValidate(t, "topologies/access.yaml", 3+6, (3*3*6+3*2)*2, 3*20,
		func(*simapi.Device) int { return 32 }, func(*simapi.Host) int { return 2 })
	defer CleanUp(t)

	info, err := p4utils.LoadP4Info("pipelines/p4info.txt")
	assert.NoError(t, err)

	totalEntries := 100
	var wg sync.WaitGroup
	for _, device := range devices {
		wg.Add(1)
		go ApplyPipelineConfigAndWriteEntries(context.Background(), t, &wg, info, device, totalEntries)
	}
	wg.Wait()

	for _, device := range devices {
		wg.Add(1)
		go ReadEntries(context.Background(), t, &wg, device, totalEntries)
	}

	wg.Wait()
}

// ReadEntries reads all device's tables' entries and makes sure their total count is as expected
func ReadEntries(ctx context.Context, t *testing.T, wg *sync.WaitGroup, device *simapi.Device, totalEntries int) {
	defer wg.Done()

	t.Logf("Connecting to agent for device %s", device.ID)
	p4Client, p4conn := GetP4Client(t, device)
	defer p4conn.Close()

	entities := make([]*p4api.Entity, 0, totalEntries)
	// Read all tables' entries
	stream, err := p4Client.Read(ctx, &p4api.ReadRequest{DeviceId: device.ChassisID, Entities: []*p4api.Entity{{
		Entity: &p4api.Entity_TableEntry{TableEntry: &p4api.TableEntry{TableId: 0}},
	}}})
	assert.NoError(t, err)

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if assert.NoError(t, err) {
			entities = append(entities, msg.Entities...)
		}
	}
	assert.Len(t, entities, totalEntries)
}

// ApplyPipelineConfigAndWriteEntries negotiates mastership, applies pipeline config and writes a slew of table entries
// to all the device's tables
func ApplyPipelineConfigAndWriteEntries(ctx context.Context, t *testing.T, wg *sync.WaitGroup,
	info *p4info.P4Info, device *simapi.Device, totalEntries int) {
	defer wg.Done()

	t.Logf("Connecting to agent for device %s", device.ID)
	p4Client, p4conn := GetP4Client(t, device)
	defer p4conn.Close()

	// Open message stream and negotiate mastership for default (no) role
	t.Logf("Negotiating mastership for device %s", device.ID)
	stream, err := p4Client.StreamChannel(ctx)
	assert.NoError(t, err)

	err = stream.Send(p4utils.CreateMastershipArbitration(&p4api.Uint128{High: 0, Low: 1}, nil))
	assert.NoError(t, err)

	msg, err := stream.Recv()
	assert.NoError(t, err)
	assert.Equal(t, int32(0), msg.GetArbitration().Status.Code)

	cookie := uint64(rand.Int63())
	arbitration := msg.GetArbitration()
	err = ApplyPipelineConfig(ctx, p4Client, arbitration.DeviceId, "", arbitration.ElectionId, cookie, info)
	assert.NoError(t, err)

	err = GenerateAndWriteTableEntries(ctx, p4Client, &p4api.WriteRequest{
		DeviceId:   msg.GetArbitration().DeviceId,
		ElectionId: msg.GetArbitration().ElectionId,
		Atomicity:  p4api.WriteRequest_CONTINUE_ON_ERROR,
	}, info, totalEntries)
	assert.NoError(t, err)

	// Use the simulator API to check the device pipeline info
	conn, err := client.CreateConnection()
	assert.NoError(t, err)
	defer conn.Close()

	deviceClient := simapi.NewDeviceServiceClient(conn)
	resp, err := deviceClient.GetDevice(ctx, &simapi.GetDeviceRequest{ID: device.ID})
	assert.NoError(t, err)

	pi := resp.Device.PipelineInfo
	assert.Equal(t, cookie, pi.Cookie)
	assert.True(t, len(pi.P4Info) > 1024)
	assert.Len(t, pi.Tables, len(info.Tables))
	assert.Len(t, pi.Counters, len(info.Counters))
	assert.Len(t, pi.Meters, len(info.Meters))
}

// ApplyPipelineConfig applies the forwarding pipeline configuration using the given device P4RT client
func ApplyPipelineConfig(ctx context.Context, client p4api.P4RuntimeClient, deviceID uint64, role string, electionID *p4api.Uint128,
	cookie uint64, info *p4info.P4Info) error {
	_, err := client.SetForwardingPipelineConfig(ctx, &p4api.SetForwardingPipelineConfigRequest{
		DeviceId:   deviceID,
		Role:       "",
		ElectionId: electionID,
		Action:     p4api.SetForwardingPipelineConfigRequest_VERIFY_AND_COMMIT,
		Config: &p4api.ForwardingPipelineConfig{
			P4Info:         info,
			P4DeviceConfig: utils.RandomBytes(2048),
			Cookie:         &p4api.ForwardingPipelineConfig_Cookie{Cookie: cookie},
		},
	})
	return err
}

// GenerateAndWriteTableEntries generates specified number of entries spread randomly between all the device tables and inserts them
func GenerateAndWriteTableEntries(ctx context.Context, client p4api.P4RuntimeClient, request *p4api.WriteRequest, info *p4info.P4Info, count int) error {
	request.Updates = make([]*p4api.Update, count)
	tl := int32(len(info.Tables))
	for i := 0; i < count; i++ {
		tableInfo := info.Tables[rand.Int31n(tl)]
		for tableInfo.Size < 128 || tableInfo.IsConstTable {
			tableInfo = info.Tables[rand.Int31n(tl)]
		}
		entry := utils.GenerateTableEntry(tableInfo, 123, nil)
		update := &p4api.Update{Type: p4api.Update_INSERT, Entity: &p4api.Entity{Entity: &p4api.Entity_TableEntry{TableEntry: entry}}}
		request.Updates = append(request.Updates, update)
	}

	_, err := client.Write(ctx, request)
	return err
}
