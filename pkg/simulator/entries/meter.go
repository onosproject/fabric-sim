// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package entries

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
)

// Meters represents a set of P4 meters
type Meters struct {
	meters map[uint32]*p4api.MeterEntry
}

// NewMeters creates a new meters store
func NewMeters(c []*p4info.Meter) *Meters {
	return &Meters{
		meters: make(map[uint32]*p4api.MeterEntry),
	}
}

// ModifyMeterEntry modifies the specified table entry in its appropriate table
func (cs *Meters) ModifyMeterEntry(entry *p4api.MeterEntry, insert bool) error {
	entry, ok := cs.meters[entry.MeterId]
	if ok && insert {
		// If the entry exists, and we're supposed to do a new insert, raise error
		return errors.NewAlreadyExists("....")
	} else if !ok && !insert {
		// If the entry doesn't exist, and we're supposed to modify, raise error
		return errors.NewNotFound("....")
	}
	cs.meters[entry.MeterId] = entry
	return nil
}

// RemoveMeterEntry removes the specified table entry
func (cs *Meters) RemoveMeterEntry(entry *p4api.MeterEntry) error {
	delete(cs.meters, entry.MeterId)
	return nil
}
