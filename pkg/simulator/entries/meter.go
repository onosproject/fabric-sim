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
	meters := &Meters{
		meters: make(map[uint32]*Meter, len(info)),
	}
	for _, mi := range info {
		meters.meters[mi.Preamble.Id] = NewMeter(mi)
	}
	return meters
}

// NewMeter creates a new meter and all its cell entries
func NewMeter(info *p4info.Meter) *Meter {
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

// ModifyMeterEntry modifies the specified meter entry cell
func (cs *Meters) ModifyMeterEntry(entry *p4api.MeterEntry, insert bool) error {
	if insert {
		return errors.NewInvalid("Meter cannot be inserted")
	}

	meter, ok := cs.meters[entry.MeterId]
	if !ok {
		return errors.NewNotFound("Meter not found")
	}
	if entry.Index == nil || entry.Index.Index < 0 || int(entry.Index.Index) >= len(meter.cells) {
		return errors.NewNotFound("Meter index out of bounds")
	}

	meter.cells[entry.Index.Index] = entry
	return nil
}
