// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package entries

import (
	"github.com/onosproject/onos-lib-go/pkg/errors"
	p4info "github.com/p4lang/p4runtime/go/p4/config/v1"
	p4api "github.com/p4lang/p4runtime/go/p4/v1"
)

// Counters represents a set of P4 counters
type Counters struct {
	counters map[uint32]*p4api.CounterEntry
}

// NewCounters creates a new counters store
func NewCounters(c []*p4info.Counter) *Counters {
	return &Counters{
		counters: make(map[uint32]*p4api.CounterEntry),
	}
}

// ModifyCounterEntry modifies the specified table entry in its appropriate table
func (cs *Counters) ModifyCounterEntry(entry *p4api.CounterEntry, insert bool) error {
	entry, ok := cs.counters[entry.CounterId]
	if ok && insert {
		// If the entry exists, and we're supposed to do a new insert, raise error
		return errors.NewAlreadyExists("....")
	} else if !ok && !insert {
		// If the entry doesn't exist, and we're supposed to modify, raise error
		return errors.NewNotFound("....")
	}
	cs.counters[entry.CounterId] = entry
	return nil
}

// RemoveCounterEntry removes the specified table entry
func (cs *Counters) RemoveCounterEntry(entry *p4api.CounterEntry) error {
	delete(cs.counters, entry.CounterId)
	return nil
}
