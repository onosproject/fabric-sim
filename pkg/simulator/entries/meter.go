// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package entries

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
)

// Meter represents all cells of a specific meter
type Meter struct {
	info  *p4info.Meter
	cells []*p4api.MeterEntry
}

// Meters represents a set of P4 meters
type Meters struct {
	meters map[uint32]*Meter
}

// NewMeters creates a new meters store
func NewMeters(info []*p4info.Meter) *Meters {
	ms := &Meters{
		meters: make(map[uint32]*Meter, len(info)),
	}
	for _, mi := range info {
		ms.meters[mi.Preamble.Id] = ms.NewMeter(mi)
	}
	return ms
}

// NewMeter creates a new meter and all its cell entries
func (ms *Meters) NewMeter(info *p4info.Meter) *Meter {
	cells := make([]*p4api.MeterEntry, info.Size)
	for i := 0; i < int(info.Size); i++ {
		// TODO: properly setup the meter spec
		cells[i] = &p4api.MeterEntry{MeterId: info.Preamble.Id, Index: &p4api.Index{Index: int64(i)}}
	}
	return &Meter{
		info:  info,
		cells: cells,
	}
}

// Meters returns the list of meters
func (ms *Meters) Meters() []*Meter {
	meters := make([]*Meter, 0, len(ms.meters))
	for _, meter := range ms.meters {
		meters = append(meters, meter)
	}
	return meters
}

// ModifyMeterEntry modifies the specified meter entry cell
func (ms *Meters) ModifyMeterEntry(entry *p4api.MeterEntry, insert bool) error {
	if insert {
		return errors.NewInvalid("meter cannot be inserted")
	}

	meter, ok := ms.meters[entry.MeterId]
	if !ok {
		return errors.NewNotFound("meter not found")
	}
	if entry.Index == nil || entry.Index.Index < 0 || int(entry.Index.Index) >= len(meter.cells) {
		return errors.NewNotFound("meter index out of bounds")
	}

	meter.cells[entry.Index.Index] = entry
	return nil
}

// ID returns the meter ID
func (m *Meter) ID() uint32 {
	return m.info.Preamble.Id
}

// Size returns the number of cells for the meter
func (m *Meter) Size() int {
	return len(m.cells)
}

// Name returns the meter name
func (m *Meter) Name() string {
	return m.info.Preamble.Name
}

// Cell returns the specified cell of the meter
func (m *Meter) Cell(index int64) *p4api.MeterEntry {
	return m.cells[index]
}
